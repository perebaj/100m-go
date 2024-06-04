package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
)

func main() {
	// how to test it? curl http://localhost:4000/flamengo
	http.HandleFunc("/flamengo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Log de teste deploy")
		fmt.Fprintf(w, "aqui é vasco porra")
	})

	//create a health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Log de teste deploy")
		fmt.Fprintf(w, "I'm alive!")
		w.WriteHeader(http.StatusOK)
	})

	port := os.Getenv("PORT")
	intPort, err := strconv.Atoi(port)
	if err != nil {
		intPort = 4000 // default is 4000
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", intPort),
		Handler: nil,
	}
	fmt.Fprintf(w, "Log de teste deploy")
	fmt.Printf("Server is running on port %d\n", intPort)
	srv.ListenAndServe()
}
