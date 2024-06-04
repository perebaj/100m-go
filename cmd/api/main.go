package main

import (
	"fmt"
	"net/http"
)

func main() {
	// how to test it? curl http://localhost:8080/flamengo
	http.HandleFunc("/flamengo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "aqui é vasco porra")
	})

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
