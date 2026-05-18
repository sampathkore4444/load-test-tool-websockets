package main

import (
	"embed"
	"log"
	"net/http"
)

//go:embed ui/*
var uiFS embed.FS

func main() {
	// Create file server for UI assets
	uiHandler := http.FileServer(http.FS(uiFS))
	
	// Handle root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "ui/index.html")
			return
		}
		// For all other paths, serve from embedded filesystem
		uiHandler.ServeHTTP(w, r)
	})

	log.Println("Starting UI server on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatalf("UI server failed to start: %v", err)
	}
}