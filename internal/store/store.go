package store

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrNotFound is returned when a test run is not found.
var ErrNotFound = errors.New("test run not found")

// TestRun represents a single test run.
type TestRun struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"` // QUEUED, RUNNING, COMPLETED, FAILED, CANCELLED
	CreatedAt    time.Time `json:"created_at"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
	ConfigJSON   []byte    `json:"config_json"` // Raw JSON of the test configuration
	ReportPath   string    `json:"report_path,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	// Protobuf schema handling
	ProtoSchemaPath string `json:"proto_schema_path,omitempty"` // Path to uploaded .proto file
	MessageType     string `json:"message_type,omitempty"`      // Message type to send (e.g., "PLAYER_MOVE")
	// OpenCode-specific fields
	GitCommit       string   `json:"git_commit,omitempty"`        // Link test to specific code version
	AISuggested     bool     `json:"ai_suggested,omitempty"`      // Flag tests initiated by AI
	PerformanceBaseline float64 `json:"performance_baseline,omitempty"` // Reference for regression detection
	CodePathsAffected []string `json:"code_paths_affected,omitempty"` // Files modified in this test session
	OpenContextSummary string `json:"opencontext_summary,omitempty"` // Summary of OpenCode session context
}

// TestStore defines the interface for storing test runs.
type TestStore interface {
	CreateTestRun(testRun *TestRun) error
	GetTestRun(id string) (*TestRun, error)
	ListTestRuns() ([]*TestRun, error)
	UpdateTestRun(testRun *TestRun) error
	DeleteTestRun(id string) error
}

// InMemoryTestStore is an in-memory implementation of TestStore.
type InMemoryTestStore struct {
	mu      sync.RWMutex
	tests   map[string]*TestRun
	nextID  int
}

// NewInMemoryTestStore creates a new in-memory test store.
func NewInMemoryTestStore() *InMemoryTestStore {
	return &InMemoryTestStore{
		tests: make(map[string]*TestRun),
		nextID: 1,
	}
}

// CreateTestRun creates a new test run.
func (s *InMemoryTestStore) CreateTestRun(testRun *TestRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if testRun.ID == "" {
		testRun.ID = fmt.Sprintf("test-%d", s.nextID)
		s.nextID++
	}
	testRun.CreatedAt = time.Now()
	s.tests[testRun.ID] = testRun
	return nil
}

// GetTestRun retrieves a test run by ID.
func (s *InMemoryTestStore) GetTestRun(id string) (*TestRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if testRun, exists := s.tests[id]; exists {
		return testRun, nil
	}
	return nil, ErrNotFound
}

// ListTestRuns returns all test runs.
func (s *InMemoryTestStore) ListTestRuns() ([]*TestRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tests := make([]*TestRun, 0, len(s.tests))
	for _, test := range s.tests {
		tests = append(tests, test)
	}
	return tests, nil
}

// UpdateTestRun updates an existing test run.
func (s *InMemoryTestStore) UpdateTestRun(testRun *TestRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tests[testRun.ID]; !exists {
		return ErrNotFound
	}
	s.tests[testRun.ID] = testRun
	return nil
}

// DeleteTestRun deletes a test run by ID.
func (s *InMemoryTestStore) DeleteTestRun(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tests[id]; !exists {
		return ErrNotFound
	}
	delete(s.tests, id)
	return nil
}