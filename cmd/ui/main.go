package main

import (
	"log"
	"net/http"
)

//go:embed ../../ui/index.html
var indexHTML []byte

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})

	log.Println("Starting UI server on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatalf("UI server failed to start: %v", err)
	}
}