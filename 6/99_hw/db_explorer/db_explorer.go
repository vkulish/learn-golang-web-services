package main

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type columnInfo struct {
	Name string
	Type string
}

type tableInfo struct {
	Name    string
	Columns []columnInfo
}

type Handler struct {
	Db *sql.DB

	TablesNames []string // TODO: make it sorted
	Tables      map[string]tableInfo
}

type Response map[string]interface{}

func (resp *Response) Bytes() []byte {
	str, err := json.Marshal(resp)
	if err != nil {
		return nil
	}
	return str
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

	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			rows.Close()
			return h, errors.Wrap(err, "unable to get table name")
		}
		h.TablesNames = append(h.TablesNames, name)
	}
	rows.Close()

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
		var table tableInfo
		table.Name = tableName
		table.Columns = make([]columnInfo, 0, 3)

		// TODO: replace this ugly code by
		// https://golang.org/pkg/database/sql/#Rows.ColumnTypes
		var fields [7]any
		for rows.Next() {
			var (
				name string
				tp   string
			)
			err = rows.Scan(&name, // Field
				&tp,        // Type
				&fields[0], // Collation
				&fields[1], // Null
				&fields[2], // Key
				&fields[3], // Default
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
			//fmt.Printf("\tColumn: %v\n", info)
			table.Columns = append(table.Columns, info)
		}
		fmt.Println("\tGot columns:", len(table.Columns))
		rows.Close()
		h.Tables[tableName] = table
		rows.Close()
	}

	return h, nil
}

func (h *Handler) processDeleteRequest(w http.ResponseWriter, r *http.Request) {
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
		response := Response{
			"error": "unknown table",
		}
		str, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", errors.Wrap(err, "Could not serialize response to JSON"))
			return false
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write(str)
		return false
	}
	return true
}

func (h *Handler) selectRecordSet(w http.ResponseWriter, tableName string, query url.Values) {
	if !h.checkTableExistance(w, tableName) {
		return
	}

	var limit int = 5
	var offset int = 0
	if query.Has("limit") {
		limit, _ = strconv.Atoi(query.Get("limit"))
	}
	if query.Has("offset") {
		offset, _ = strconv.Atoi(query.Get("offset"))
	}

	rows, err := h.Db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", tableName), limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("%s", errors.Wrap(err, "Could not select from table "+tableName))
		return //h, errors.Wrap(err, "unable to get tables list from the DB")
	}
	defer rows.Close()

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
			fmt.Printf("%s", errors.Wrap(err, "Could not scan values from table "+tableName))
			rows.Close()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var valuesMap = make(map[string]interface{})
		for idx, item := range data {
			switch item.(type) {
			case *interface{}:
				ptr := *item.(*interface{})
				switch ptr.(type) {
				case int64:
					valuesMap[tableInfo.Columns[idx].Name] = ptr.(int64)
				case []uint8:
					valuesMap[tableInfo.Columns[idx].Name] = string(ptr.([]uint8))
				default:
					valuesMap[tableInfo.Columns[idx].Name] = nil
					//fmt.Printf("\titem %d type %T\n", idx, T)
				}
			default:
				continue
			}
		}
		records = append(records, valuesMap)
		//response["response"].(Response)["records"] = append(response["response"].(Response)["records"], valuesMap)
	}

	var response = Response{
		"response": Response{
			"records": records,
		},
	}
	//response["response"].(Response)["records"] = records

	w.WriteHeader(http.StatusOK)
	w.Write(response.Bytes())
}

func (h *Handler) processGetRequest(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" { // List all tables
		response := Response{
			"response": Response{
				"tables": h.TablesNames,
			},
		}
		str, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", errors.Wrap(err, "Could not serialize response to JSON"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(str)
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
		//TODO: request DB for a particular item
		rows, err := h.Db.Query("SELECT * FROM ? WHERE id = ?", tableName, itemId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", errors.Wrap(err, fmt.Sprintf("unable to get data from `%s` table", itemId)))
			return
		}
		for rows.Next() {

		}
		rows.Close()
	}
}

func (h *Handler) processPutRequest(w http.ResponseWriter, r *http.Request) {
}

func (h *Handler) processPostRequest(w http.ResponseWriter, r *http.Request) {
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
