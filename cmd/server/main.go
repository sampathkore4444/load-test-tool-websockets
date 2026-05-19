package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"loadtest-tool/internal/api"
	"loadtest-tool/internal/store"
)

func main() {
	// Initialize store
	testStore := store.NewInMemoryTestStore()

	// Initialize API handler
	handler := api.NewHandler(testStore)

	// Set up HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tests", handler.HandleTests) // GET list, POST create
	mux.HandleFunc("/api/tests/", func(w http.ResponseWriter, r *http.Request) {
		// Handle individual test operations: GET, DELETE, and POST for start/stop
		switch r.Method {
		case http.MethodGet:
			handler.HandleTestByID(w, r)
		case http.MethodDelete:
			handler.HandleTestByID(w, r)
		case http.MethodPost:
			// For start/stop actions, delegate to HandleTestActions
			handler.HandleTestActions(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Create a channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Run the server in a goroutine
	go func() {
		log.Println("Starting server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}