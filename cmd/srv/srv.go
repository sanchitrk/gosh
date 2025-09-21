package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received log stream...")
		// Copy the request body (the logs) to the server's stdout.
		io.Copy(os.Stdout, r.Body)
		w.WriteHeader(http.StatusOK)
	})
	log.Println("Log ingestor server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
