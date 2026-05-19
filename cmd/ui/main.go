package main

import (
	"embed"
	"log"
	"net/http"
)

//go:embed ../ui/*
var uiFS embed.FS

func main() {
	// Create file server for UI assets
	uiHandler := http.FileServer(http.FS(uiFS))

	// Handle all paths by serving from embedded filesystem
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		uiHandler.ServeHTTP(w, r)
	})

	log.Println("Starting UI server on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatalf("UI server failed to start: %v", err)
	}
}