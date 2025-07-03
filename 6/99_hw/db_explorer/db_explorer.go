package main

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

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

	TablesNames []string
	Tables      map[string]tableInfo
}

type Response map[string]interface{}

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
		// and do not support using placeholders for
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

func (h *Handler) processGetRequest(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		response := Response{
			"response": Response{
				"tables": h.TablesNames,
			},
		}
		str, err := json.Marshal(response)
		//fmt.Printf("response:%s\b", str)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", errors.Wrap(err, "Could not serialize response to JSON"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(str)
		return
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
