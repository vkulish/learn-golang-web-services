package main

import (
	"fmt"
	"net/http"
)

func handler(catalog *Catalog, w http.ResponseWriter, r *http.Request) {
	result, err := SearchServer(catalog, r)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	fmt.Fprintf(w, "result: %+v", result)
}

func main() {
	catalog, err := LoadTestData()
	if err != nil {
		fmt.Println("Error loading test data:", err)
		return
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(catalog, w, r)
	})

	fmt.Println("starting server at :8080")
	_ = http.ListenAndServe(":8080", nil)
}
