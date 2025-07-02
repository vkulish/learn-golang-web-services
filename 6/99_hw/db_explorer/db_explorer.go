package main

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

type columnInfo struct {
	Id   int
	Name string
	Type any
}

type tableInfo struct {
	Name    string
	Columns []columnInfo
}

type Handler struct {
	Db *sql.DB

	Tables map[string]tableInfo
}

func NewDbExplorer(db *sql.DB) (Handler, error) {
	var h Handler
	h.Db = db

	// STEP 1: get tables list
	rows, err := h.Db.Query("SHOW TABLES")
	if err != nil {
		return h, errors.Wrap(err, "unable to get tables list from the DB")
	}

	tables := make([]string, 0, 3)
	for rows.Next() {
		//TODO
	}
	rows.Close()

	// STEP 2: get columns info for every table
	for _, tableName := range tables {
		rows, err := h.Db.Query("SHOW FULL COLUMNS FROM ?", tableName)
		if err != nil {
			return h, errors.Wrap(err, fmt.Sprintf("unable to get columns for the table %s", tableName))
		}
		var table tableInfo
		table.Name = tableName
		table.Columns = make([]columnInfo, 0, 3)
		// TODO: fill columns info
		//rawColumn := &sql.ColumnType
		for rows.Next() {
			//err = rows.Scan(rawColumn., &post.Title, &post.Updated)
			//if err == nil {
			//	var column columnInfo
			//
			//}
		}
		// надо закрывать соединение, иначе будет течь
		rows.Close()
		h.Tables[tableName] = table
		rows.Close()
	}

	return h, nil
}

func (h *Handler) processDeleteRequest(w http.ResponseWriter, r *http.Request) {
}

func (h *Handler) processGetRequest(w http.ResponseWriter, r *http.Request) {
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
