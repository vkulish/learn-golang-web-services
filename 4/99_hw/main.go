package main

import (
	"fmt"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	order_field := r.URL.Query().Get("order_field")
	order_by := r.URL.Query().Get("order_by")
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")

	result, err := SearchServer(query, order_field, order_by, limit, offset)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	fmt.Fprintf(w, "result: %v", result)
}

func main() {
	LoadTestData()

	http.HandleFunc("/", handler)

	fmt.Println("starting server at :8080")
	http.ListenAndServe(":8080", nil)
}
