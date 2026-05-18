package api

import (
	"net/http"
	"strings"
)

// HandleTestActions handles POST requests to /api/tests/{id}/start and /api/tests/{id}/stop
func (h *Handler) HandleTestActions(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	
	// Check if path starts with /api/tests/
	if !strings.HasPrefix(path, "/api/tests/") {
		http.Error(w, "invalid path", http.StatusNotFound)
		return
	}
	
	// Remove the prefix
	rest := strings.TrimPrefix(path, "/api/tests/")
	
	// Split the remaining path by '/'
	parts := strings.Split(rest, "/")
	
	// We expect either "{id}" (for GET/DELETE) or "{id}/start" or "{id}/stop"
	if len(parts) == 1 {
		// This is handled by HandleTestByID for GET/DELETE
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if len(parts) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	
	id := parts[0]
	action := parts[1]
	
	if id == "" {
		http.Error(w, "missing test ID", http.StatusBadRequest)
		return
	}
	
	switch action {
	case "start":
		h.handleStartTest(w, r, id)
	case "stop":
		h.handleStopTest(w, r, id)
	default:
		http.Error(w, "invalid action", http.StatusBadRequest)
	}
}