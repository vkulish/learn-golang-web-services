package main

import (
	"fmt"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	result, err := SearchServer(r)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	fmt.Fprintf(w, "result: %+v", result)
}

func main() {
	LoadTestData()

	http.HandleFunc("/", handler)

	fmt.Println("starting server at :8080")
	http.ListenAndServe(":8080", nil)
}
