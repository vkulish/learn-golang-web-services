package main

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type columnInfo struct {
	Name         string
	Type         string
	MayBeNull    bool
	IsKey        bool // column is a PK
	DefaultValue any
}

type tableInfo struct {
	Name    string
	Columns []columnInfo
}

func (tbl tableInfo) getPrimaryKeyColumn() string {
	for _, info := range tbl.Columns {
		if info.IsKey {
			return info.Name
		}
	}
	return "id"
}

type Handler struct {
	Db *sql.DB

	TablesNames []string // TODO: make it sorted
	Tables      map[string]tableInfo
}

type DataToExchange map[string]interface{}

func (resp *DataToExchange) Bytes() []byte {
	str, err := json.Marshal(resp)
	if err != nil {
		return nil
	}
	return str
}

func validateItemType(colInfo *columnInfo, value interface{}) bool {
	var typesEqual bool
	switch value.(type) {
	case string:
		typesEqual = colInfo.Type == "varchar(255)" || colInfo.Type == "text"
	case int64:
		typesEqual = colInfo.Type == "int"
	case nil:
		typesEqual = colInfo.MayBeNull
	default:
		typesEqual = false
	}

	return typesEqual
}

func defaultValueForType(columnType string) any {
	switch columnType {
	case "varchar(255)":
		return string(``)
	case "text":
		return string(``)
	case "int":
		return 0
	}
	return nil
}

func NewDbExplorer(db *sql.DB) (Handler, error) {
	var h Handler
	h.Db = db
	h.TablesNames = make([]string, 0, 5)
	h.Tables = make(map[string]tableInfo)

	// STEP 1: get tables list
	rows, err := h.Db.Query("SHOW TABLES")
	if err != nil {
		return h, errors.Wrap(err, "unable to get tables list from the DB")
	}

	defer rows.Close()

	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			rows.Close()
			return h, errors.Wrap(err, "unable to get table name")
		}
		h.TablesNames = append(h.TablesNames, name)
	}

	fmt.Printf("Found tables: %v\n", h.TablesNames)

	// STEP 2: get columns info for each table
	for _, tableName := range h.TablesNames {
		fmt.Println("Getting columns info for table:", tableName)
		// In MySQL, the `SHOW` commands are a bit different
		// and do not support placeholders for
		// identifiers (like table names, column names) in prepared statements.
		// So build the query string by hand, it`s safe here.
		rows, err := h.Db.Query("SHOW FULL COLUMNS FROM " + tableName)
		if err != nil {
			return h, errors.Wrap(err, fmt.Sprintf("unable to get columns for the table %s", tableName))
		}
		defer rows.Close()

		var table tableInfo
		table.Name = tableName
		table.Columns = make([]columnInfo, 0, 3)

		var fields [7]any
		for rows.Next() {
			var (
				name     string
				tp       string
				nullable string
				key      string
				defValue any
			)
			err = rows.Scan(&name, // Field
				&tp,        // Type
				&fields[0], // Collation
				&nullable,  // Null
				&key,       // Key
				&defValue,  // Default
				&fields[4], // Extra
				&fields[5], // Privilges
				&fields[6]) // Comment
			if err != nil {
				return h, errors.Wrap(err, fmt.Sprintf("unable to scan column for the table %s", tableName))
			}
			//fmt.Printf("\tFields: %+v", fields)
			var info columnInfo
			info.Name = name
			info.Type = tp
			info.MayBeNull = nullable == "YES"
			info.IsKey = key == "PRI"
			info.DefaultValue = defValue
			//fmt.Printf("\tColumn: %v, key: %s, default: %v\n", info, key, defValue)
			table.Columns = append(table.Columns, info)
		}
		//fmt.Println("\tGot columns:", len(table.Columns))
		h.Tables[tableName] = table
	}

	return h, nil
}

func (h *Handler) processDeleteRequest(w http.ResponseWriter, r *http.Request) {
	paths := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(paths) != 2 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var tableName = paths[0]
	var itemId = paths[1]
	if !h.checkTableExistance(w, tableName) {
		return
	}

	var pkey = h.Tables[tableName].getPrimaryKeyColumn()
	var deleteStatement = fmt.Sprintf("DELETE FROM %s WHERE %s=?", tableName, pkey)
	result, err := h.Db.Exec(deleteStatement, itemId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("%s", errors.Wrap(err, fmt.Sprintf("unable to delete item id: %s", itemId)))
		return
	}

	deleted, _ := result.RowsAffected()

	var response = DataToExchange{
		"response": DataToExchange{
			"deleted": deleted,
		},
	}

	w.WriteHeader(http.StatusOK)
	w.Write(response.Bytes())
}

func (h *Handler) checkTableExistance(w http.ResponseWriter, tableName string) bool {
	var tableFound bool
	for _, table := range h.TablesNames {
		if table == tableName {
			tableFound = true
			break
		}
	}
	if !tableFound {
		response := DataToExchange{
			"error": "unknown table",
		}
		str, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(errors.Wrap(err, "Could not serialize response to JSON"))
			return false
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write(str)
		return false
	}
	return true
}

func (h *Handler) listTables(w http.ResponseWriter) {
	response := DataToExchange{
		"response": DataToExchange{
			"tables": h.TablesNames,
		},
	}

	w.WriteHeader(http.StatusOK)
	w.Write(response.Bytes())
}

func (h *Handler) processSelectedRows(w http.ResponseWriter, tableName string, rows *sql.Rows) ([]interface{}, error) {
	var tableInfo = h.Tables[tableName]
	var count = len(tableInfo.Columns)
	values := make([]interface{}, count)
	data := make([]interface{}, count)

	var records = make([]interface{}, 0, 5)
	for rows.Next() {
		for i := range tableInfo.Columns {
			data[i] = &values[i]
		}
		err := rows.Scan(data...)
		if err != nil {
			var finalError = errors.Wrap(err, "Could not scan values from table "+tableName)
			fmt.Printf("%s", finalError)
			w.WriteHeader(http.StatusInternalServerError)
			return nil, finalError
		}

		var valuesMap = make(map[string]interface{})
		for idx, item := range data {
			switch item.(type) {
			case *interface{}:
				ptr := *item.(*interface{})
				switch T := ptr.(type) {
				case int64:
					valuesMap[tableInfo.Columns[idx].Name] = T
				case []uint8:
					valuesMap[tableInfo.Columns[idx].Name] = string(T)
				default:
					valuesMap[tableInfo.Columns[idx].Name] = nil
				}
			default:
				continue
			}
		}
		records = append(records, valuesMap)
	}
	return records, nil
}

func (h *Handler) selectRecordSet(w http.ResponseWriter, tableName string, query url.Values) {
	if !h.checkTableExistance(w, tableName) {
		return
	}

	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil {
		limit = 5
	}

	offset, err := strconv.Atoi(query.Get("offset"))
	if err != nil {
		offset = 0
	}

	rows, err := h.Db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", tableName), limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("%s", errors.Wrap(err, "Could not select from table "+tableName))
		return //h, errors.Wrap(err, "unable to get tables list from the DB")
	}
	defer rows.Close()

	records, err := h.processSelectedRows(w, tableName, rows)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	var response = DataToExchange{
		"response": DataToExchange{
			"records": records,
		},
	}

	w.WriteHeader(http.StatusOK)
	w.Write(response.Bytes())
}

func (h *Handler) processGetRequest(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		h.listTables(w)
		return
	}

	paths := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	fmt.Println("\tGot paths:", paths, "len:", len(paths))
	if len(paths) == 1 { // have a form like "GET /$table?limit=5&offset=7"
		var tableName = paths[0]
		h.selectRecordSet(w, tableName, r.URL.Query())
	} else if len(paths) == 2 {
		var tableName = paths[0]
		var itemId = paths[1]
		if !h.checkTableExistance(w, tableName) {
			return
		}

		var pkey = h.Tables[tableName].getPrimaryKeyColumn()
		rows, err := h.Db.Query(fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", tableName, pkey), itemId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println("Got error:", errors.Wrap(err, fmt.Sprintf("unable to get data from `%s` table", itemId)))
			return
		}
		defer rows.Close()

		records, err := h.processSelectedRows(w, tableName, rows)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println("Got error:", err)
			return
		}

		if len(records) == 0 {
			fmt.Println("\tCould not find requested element with ID=" + itemId)
			var response = DataToExchange{
				"error": "record not found",
			}
			w.WriteHeader(http.StatusNotFound)
			w.Write(response.Bytes())
			return
		}

		var response = DataToExchange{
			"response": DataToExchange{
				"record": records[0],
			},
		}

		w.WriteHeader(http.StatusOK)
		w.Write(response.Bytes())
	}
}

func (h *Handler) processPutRequest(w http.ResponseWriter, r *http.Request) {
	paths := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(paths) != 1 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var tableName = paths[0]
	if !h.checkTableExistance(w, tableName) {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	req := DataToExchange{}
	err = json.Unmarshal(body, &req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println("\tUnable to unmarshal body:", err)
		return
	}

	var info = h.Tables[tableName]
	var columnsToInsert string
	var substitutionMask string
	var values = make([]interface{}, 0, len(info.Columns))
	var firstCol bool = true
	for _, colInfo := range info.Columns {
		value, found := req[colInfo.Name]
		if !found {
			// The column has no value presented in
			// the request so generate value when there is
			// no default value provided in the DB.
			if colInfo.DefaultValue != nil {
				value = colInfo.DefaultValue
			} else if !colInfo.MayBeNull {
				value = defaultValueForType(colInfo.Type)
			}
		}

		if !firstCol {
			columnsToInsert += ", "
			substitutionMask += ", "
		}
		firstCol = false

		columnsToInsert += colInfo.Name
		substitutionMask += "?"
		values = append(values, value)
	}

	var insertStatement = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, columnsToInsert, substitutionMask)
	result, err := h.Db.Exec(insertStatement, values...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("%s", errors.Wrap(err, fmt.Sprintf("unable to insert into %s", tableName)))
		return
	}

	lid, _ := result.LastInsertId()
	fmt.Println("\tInserted item id:", lid)

	var pkey = h.Tables[tableName].getPrimaryKeyColumn()
	var response = DataToExchange{
		"response": DataToExchange{
			pkey: lid,
		},
	}

	w.WriteHeader(http.StatusOK)
	w.Write(response.Bytes())
}

func (h *Handler) processPostRequest(w http.ResponseWriter, r *http.Request) {
	paths := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(paths) != 2 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var tableName = paths[0]
	var itemId = paths[1]
	if !h.checkTableExistance(w, tableName) {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer r.Body.Close()

	req := DataToExchange{}
	err = json.Unmarshal(body, &req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println("\tUnable to unmarshal body:", err)
		return
	}

	var reportBadField = func(fldName string) {
		var errResponce = DataToExchange{
			"error": fmt.Sprintf("field %s have invalid type", fldName),
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errResponce.Bytes())
	}

	var info = h.Tables[tableName]
	var pkey = info.getPrimaryKeyColumn()
	var setPattern string
	var values = make([]interface{}, 0, len(info.Columns))
	var firstCol = true
	for _, colInfo := range info.Columns {
		value, ok := req[colInfo.Name]
		if !ok {
			continue
		}

		if colInfo.Name == pkey {
			// Found an attempt to change primary key
			reportBadField(colInfo.Name)
			return
		}

		if !validateItemType(&colInfo, value) {
			// Found a type mismatch
			reportBadField(colInfo.Name)
			return
		}

		if !firstCol {
			setPattern += ", "
		}
		firstCol = false

		setPattern += colInfo.Name + "=?"
		values = append(values, value)
	}
	values = append(values, itemId)

	var updateStatement = fmt.Sprintf("UPDATE %s SET %s WHERE %s=?", tableName, setPattern, pkey)
	result, err := h.Db.Exec(updateStatement, values...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("%s", errors.Wrap(err, fmt.Sprintf("unable to update item id: %s", itemId)))
		return
	}

	updated, _ := result.RowsAffected()
	fmt.Println("\tRows affected:", updated)

	var response = DataToExchange{
		"response": DataToExchange{
			"updated": updated,
		},
	}

	w.WriteHeader(http.StatusOK)
	w.Write(response.Bytes())
}

// Entry point for Handler: routing starts here
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Processing URL:", r.URL.String(), "Method:", r.Method)
	switch r.Method {
	case "DELETE":
		h.processDeleteRequest(w, r)
	case "GET":
		h.processGetRequest(w, r)
	case "PUT":
		h.processPutRequest(w, r)
	case "POST":
		h.processPostRequest(w, r)
	}
}
