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
	mux.HandleFunc("/api/tests", handler.HandleTests)
	mux.HandleFunc("/api/tests/", handler.HandleTestByID) // For GET and DELETE on specific test
	mux.HandleFunc("/api/tests/", handler.HandleTestActions) // For POST to /{id}/start and /{id}/stop

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