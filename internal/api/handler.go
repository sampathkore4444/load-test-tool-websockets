package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"loadtest-tool/internal/runner"
	"loadtest-tool/internal/store"
)

// Handler handles HTTP requests for the test tool.
type Handler struct {
	store store.TestStore
	mu    sync.RWMutex // Protects access to handler methods
}

// NewHandler creates a new Handler with the given test store.
func NewHandler(s store.TestStore) *Handler {
	return &Handler{store: s}
}

// HandleTests handles requests to /api/tests (GET and POST).
func (h *Handler) HandleTests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetTests(w, r)
	case http.MethodPost:
		h.handleCreateTest(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleTestByID handles requests to /api/tests/{id} (GET, POST for start/stop, DELETE).
func (h *Handler) HandleTestByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/tests/"):]
	if id == "" {
		http.Error(w, "missing test ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetTest(w, r, id)
	case http.MethodPost:
		// We'll use subpaths for start and stop: /api/tests/{id}/start and /api/tests/{id}/stop
		// But since we are handling /api/tests/{id}, we need to check for these subpaths in a separate handler.
		// Alternatively, we can handle them here by checking the request URL for additional path.
		// Let's change the routing: we'll handle /api/tests/{id} for GET and DELETE, and have separate handlers for start/stop.
		// For simplicity, we'll adjust: we'll handle /api/tests/{id} for GET and DELETE, and have /api/tests/{id}/start and /api/tests/{id}/stop as separate routes.
		// However, the current setup in main.go routes /api/tests/ to HandleTestByID, so we need to check the full path.
		// Let's do: if the path is exactly /api/tests/{id} then GET, if it's /api/tests/{id}/start then POST for start, etc.
		// We'll change the main.go to route more specifically, but for now, let's handle in this function by checking the URL.
		path := r.URL.Path
		switch {
		case path == "/api/tests/"+id+"/start":
			h.handleStartTest(w, r, id)
		case path == "/api/tests/"+id+"/stop":
			h.handleStopTest(w, r, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case http.MethodDelete:
		h.handleDeleteTest(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetTests returns a list of test runs.
func (h *Handler) handleGetTests(w http.ResponseWriter, r *http.Request) {
	tests, err := h.store.ListTestRuns()
	if err != nil {
		http.Error(w, "failed to list tests", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tests)
}

// handleCreateTest creates a new test run from the request body.
func (h *Handler) handleCreateTest(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (for file upload)
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	var testRun store.TestRun
	// Get form values
	testRun.Name = r.FormValue("testName")
	// TODO: Validate required fields

	// Set initial status to QUEUED
	testRun.Status = "QUEUED"

	// Handle file upload for proto schema
	if file, handler, err := r.FormFile("protoSchema"); err == nil {
		defer file.Close()
		// Create directory if it doesn't exist
		if err := os.MkdirAll("./proto_schemas", 0o755); err != nil {
			http.Error(w, "Unable to create directory for proto schemas", http.StatusInternalServerError)
			return
		}
		// Save the file
		filePath := filepath.Join("./proto_schemas", handler.Filename)
		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Unable to save file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Unable to save file", http.StatusInternalServerError)
			return
		}
		testRun.ProtoSchemaPath = filePath
	} else if err != http.ErrMissingParam {
		// If there was an error that's not about a missing parameter
		http.Error(w, "Error processing file upload", http.StatusBadRequest)
		return
	}

	// Get message type
	testRun.MessageType = r.FormValue("messageType")

	// Build config JSON from form data
	config := map[string]interface{}{
		"endpoint":        r.FormValue("wsEndpoint"),
		"connections":     r.FormValue("virtualUsers"),
		"messagesPerSecond": r.FormValue("messagesPerSecond"),
		"duration":        r.FormValue("duration"),
		"payload":         r.FormValue("payload"),
		"authToken":       r.FormValue("authToken"),
		"headers":         r.FormValue("headers"),
		"eventType":       r.FormValue("eventType"),
		"protoSchemaPath": testRun.ProtoSchemaPath,
		"messageType":     testRun.MessageType,
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		http.Error(w, "Unable to create test configuration", http.StatusInternalServerError)
		return
	}
	testRun.ConfigJSON = configJSON

	if err := h.store.CreateTestRun(&testRun); err != nil {
		http.Error(w, "failed to create test", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"testId": testRun.ID, "status": testRun.Status})
}

// handleGetTest returns a single test run by ID.
func (h *Handler) handleGetTest(w http.ResponseWriter, r *http.Request, id string) {
	testRun, err := h.store.GetTestRun(id)
	if err != nil {
		if err == store.ErrNotFound {
			http.Error(w, "test not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get test", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(testRun)
}

// handleStartTest starts a test run (changes status to RUNNING and triggers the load runner).
func (h *Handler) handleStartTest(w http.ResponseWriter, r *http.Request, id string) {
	testRun, err := h.store.GetTestRun(id)
	if err != nil {
		if err == store.ErrNotFound {
			http.Error(w, "test not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get test", http.StatusInternalServerError)
		return
	}
	if testRun.Status != "QUEUED" {
		http.Error(w, "test cannot be started (not in QUEUED state)", http.StatusBadRequest)
		return
	}

	// Update status to RUNNING
	testRun.Status = "RUNNING"
	testRun.StartedAt = time.Now()
	if err := h.store.UpdateTestRun(testRun); err != nil {
		http.Error(w, "failed to start test", http.StatusInternalServerError)
		return
	}

	// Parse the test configuration to create a load runner config
	var configMap map[string]interface{}
	if err := json.Unmarshal(testRun.ConfigJSON, &configMap); err != nil {
		http.Error(w, "invalid test configuration", http.StatusBadRequest)
		return
	}

	// Extract configuration values with defaults
	endpoint := "ws://localhost:8080/ws" // default
	if v, ok := configMap["endpoint"]; ok {
		if s, ok := v.(string); ok {
			endpoint = s
		}
	}

	connections := 100 // default
	if v, ok := configMap["connections"]; ok {
		if f, ok := v.(float64); ok {
			connections = int(f)
		}
	}

	messagesPerSecond := 1000 // default
	if v, ok := configMap["messagesPerSecond"]; ok {
		if f, ok := v.(float64); ok {
			messagesPerSecond = int(f)
		}
	}

	durationStr := "30s" // default
	if v, ok := configMap["duration"]; ok {
		if s, ok := v.(string); ok {
			durationStr = s
		}
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		duration = 30 * time.Second
	}

	payload := []byte(`{"event":"PLAYER_MOVE"}`) // default payload
	if v, ok := configMap["payload"]; ok {
		if p, ok := v.(map[string]interface{}); ok {
			// Simple JSON payload for now
			payloadBytes, _ := json.Marshal(p)
			payload = payloadBytes
		} else if s, ok := v.(string); ok {
			payload = []byte(s)
		}
	}

	authToken := "" // default
	if v, ok := configMap["authToken"]; ok {
		if s, ok := v.(string); ok {
			authToken = s
		}
	}

	// Create headers
	headers := make(http.Header)
	if v, ok := configMap["headers"]; ok {
		if hMap, ok := v.(map[string]interface{}); ok {
			for k, v := range hMap {
				if s, ok := v.(string); ok {
					headers.Add(k, s)
				}
			}
		}
	}

	// Get protobuf schema info
	protoSchemaPath := ""
	if v, ok := configMap["protoSchemaPath"]; ok {
		if s, ok := v.(string); ok {
			protoSchemaPath = s
		}
	}

	messageType := ""
	if v, ok := configMap["messageType"]; ok {
		if s, ok := v.(string); ok {
			messageType = s
		}
	}

	// Create and start the load runner in a goroutine
	lrConfig := runner.LoadRunnerConfig{
		Endpoint:        endpoint,
		Connections:     connections,
		MessagesPerSec:  messagesPerSecond,
		Duration:        duration,
		Payload:         payload,
		AuthToken:       authToken,
		Headers:         headers,
		ProtoSchemaPath: protoSchemaPath,
		MessageType:     messageType,
	}

	go func() {
		lr := runner.NewLoadRunner(lrConfig)
		lr.Run()

		// Update test run with results
		h.mu.Lock()
		defer h.mu.Unlock()

		// Get the test run again to ensure we have the latest
		updatedTestRun, err := h.store.GetTestRun(id)
		if err != nil {
			log.Printf("Error getting test run %s after test completion: %v", id, err)
			return
		}

		// Update with results (in a real implementation, we'd store metrics separately)
		updatedTestRun.Status = "COMPLETED"
		updatedTestRun.CompletedAt = time.Now()
		// We could store the result in a field or generate a report here
		if err := h.store.UpdateTestRun(updatedTestRun); err != nil {
			log.Printf("Error updating test run %s with results: %v", id, err)
		}
	}()

	// Return the updated test run immediately
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(testRun)
}
		http.Error(w, "failed to get test", http.StatusInternalServerError)
		return
	}
	if testRun.Status != "QUEUED" {
		http.Error(w, "test cannot be started (not in QUEUED state)", http.StatusBadRequest)
		return
	}

	// Update status to RUNNING
	testRun.Status = "RUNNING"
	testRun.StartedAt = time.Now()
	if err := h.store.UpdateTestRun(testRun); err != nil {
		http.Error(w, "failed to start test", http.StatusInternalServerError)
		return
	}

	// Parse the test configuration to create a load runner config
	var configMap map[string]interface{}
	if err := json.Unmarshal(testRun.ConfigJSON, &configMap); err != nil {
		http.Error(w, "invalid test configuration", http.StatusBadRequest)
		return
	}

	// Extract configuration values with defaults
	endpoint := "ws://localhost:8080/ws" // default
	if v, ok := configMap["endpoint"]; ok {
		if s, ok := v.(string); ok {
			endpoint = s
		}
	}

	connections := 100 // default
	if v, ok := configMap["connections"]; ok {
		if f, ok := v.(float64); ok {
			connections = int(f)
		}
	}

	messagesPerSecond := 1000 // default
	if v, ok := configMap["messagesPerSecond"]; ok {
		if f, ok := v.(float64); ok {
			messagesPerSecond = int(f)
		}
	}

	durationStr := "30s" // default
	if v, ok := configMap["duration"]; ok {
		if s, ok := v.(string); ok {
			durationStr = s
		}
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		duration = 30 * time.Second
	}

	payload := []byte(`{"event":"PLAYER_MOVE"}`) // default payload
	if v, ok := configMap["payload"]; ok {
		if p, ok := v.(map[string]interface{}); ok {
			// Simple JSON payload for now
			payloadBytes, _ := json.Marshal(p)
			payload = payloadBytes
		} else if s, ok := v.(string); ok {
			payload = []byte(s)
		}
	}

	authToken := "" // default
	if v, ok := configMap["authToken"]; ok {
		if s, ok := v.(string); ok {
			authToken = s
		}
	}

	// Create headers
	headers := make(http.Header)
	if v, ok := configMap["headers"]; ok {
		if hMap, ok := v.(map[string]interface{}); ok {
			for k, v := range hMap {
				if s, ok := v.(string); ok {
					headers.Add(k, s)
				}
			}
		}
	}

	// Create and start the load runner in a goroutine
	lrConfig := runner.LoadRunnerConfig{
		Endpoint:        endpoint,
		Connections:     connections,
		MessagesPerSec:  messagesPerSecond,
		Duration:        duration,
		Payload:         payload,
		AuthToken:       authToken,
		Headers:         headers,
	}

	go func() {
		lr := runner.NewLoadRunner(lrConfig)
		lr.Run()

		// Update test run with results
		h.mu.Lock()
		defer h.mu.Unlock()

		// Get the test run again to ensure we have the latest
		updatedTestRun, err := h.store.GetTestRun(id)
		if err != nil {
			log.Printf("Error getting test run %s after test completion: %v", id, err)
			return
		}

		// Update with results (in a real implementation, we'd store metrics separately)
		updatedTestRun.Status = "COMPLETED"
		updatedTestRun.CompletedAt = time.Now()
		// We could store the result in a field or generate a report here
		if err := h.store.UpdateTestRun(updatedTestRun); err != nil {
			log.Printf("Error updating test run %s with results: %v", id, err)
		}
	}()

	// Return the updated test run immediately
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(testRun)
}

// handleStopTest stops a test run (changes status to CANCELLED or COMPLETED).
// For simplicity, we'll set to CANCELLED when stopped manually.
func (h *Handler) handleStopTest(w http.ResponseWriter, r *http.Request, id string) {
	testRun, err := h.store.GetTestRun(id)
	if err != nil {
		if err == store.ErrNotFound {
			http.Error(w, "test not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get test", http.StatusInternalServerError)
		return
	}
	if testRun.Status != "RUNNING" {
		http.Error(w, "test cannot be stopped (not in RUNNING state)", http.StatusBadRequest)
		return
	}
	testRun.Status = "CANCELLED"
	testRun.CompletedAt = time.Now()
	if err := h.store.UpdateTestRun(testRun); err != nil {
		http.Error(w, "failed to stop test", http.StatusInternalServerError)
		return
	}
	// TODO: Actually stop the WebSocket load runner process.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(testRun)
}

// handleDeleteTest deletes a test run by ID.
func (h *Handler) handleDeleteTest(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.DeleteTestRun(id); err != nil {
		if err == store.ErrNotFound {
			http.Error(w, "test not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to delete test", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}