package runner

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// LoadRunnerConfig holds configuration for the WebSocket load runner.
type LoadRunnerConfig struct {
	Endpoint        string        // WebSocket endpoint URL
	Connections     int           // Number of concurrent WebSocket connections
	MessagesPerSec  int           // Target messages per second across all connections
	Duration        time.Duration // How long to run the test
	Payload         []byte        // Protobuf-encoded payload to send
	AuthToken       string        // Authentication token (if required)
	Headers         http.Header   // Additional headers for WebSocket connection
}

// LoadRunnerResult holds the results of a load test run.
type LoadRunnerResult struct {
	TotalConnections   int
	SuccessfulConnects int
	FailedConnects     int
	TotalMessagesSent  int64
	TotalMessagesRecv  int64
	TotalErrors        int64
	AvgLatencyMs       float64
	MinLatencyMs       float64
	MaxLatencyMs       float64
	P95LatencyMs       float64
	StartTime          time.Time
	EndTime            time.Time
}

// LoadRunner simulates WebSocket clients for load testing.
type LoadRunner struct {
	config   LoadRunnerConfig
	result   LoadRunnerResult
	mu       sync.Mutex
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	latencies []float64
	latencyMu sync.Mutex
}

// NewLoadRunner creates a new LoadRunner with the given config.
func NewLoadRunner(config LoadRunnerConfig) *LoadRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &LoadRunner{
		config: config,
		result: LoadRunnerResult{
			StartTime: time.Now(),
			MinLatencyMs: 1e9, // Large initial value
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Run starts the load test and returns the result when completed.
func (lr *LoadRunner) Run() LoadRunnerResult {
	// Calculate messages per second per connection
	mpsPerConn := lr.config.MessagesPerSec / lr.config.Connections
	if mpsPerConn == 0 {
		mpsPerConn = 1 // At least 1 message per second per connection if total MPS > 0
	}

	// Launch worker goroutines for each connection
	lr.wg.Add(lr.config.Connections)
	for i := 0; i < lr.config.Connections; i++ {
		go lr.connectionWorker(i, mpsPerConn)
	}

	// Wait for duration or until context is cancelled
	select {
	case <-time.After(lr.config.Duration):
		// Duration elapsed, cancel the context
		lr.cancel()
	case <-lr.ctx.Done():
		// Context cancelled externally
	}

	// Wait for all workers to finish
	lr.wg.Wait()
	lr.result.EndTime = time.Now()

	// Calculate latency statistics
	if len(lr.latencies) > 0 {
		lr.calculateLatencyStats()
	}

	return lr.result
}

// connectionWorker handles a single WebSocket connection.
func (lr *LoadRunner) connectionWorker(id int, mpsPerConn int) {
	defer lr.wg.Done()

	// Prepare WebSocket dialer
	u, err := url.Parse(lr.config.Endpoint)
	if err != nil {
		log.Printf("Worker %d: failed to parse endpoint: %v", id, err)
		lr.mu.Lock()
		lr.result.FailedConnects++
		lr.mu.Unlock()
		return
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// Set up headers including auth token if provided
	header := make(http.Header)
	if lr.config.AuthToken != "" {
		header.Set("Authorization", "Bearer "+lr.config.AuthToken)
	}
	// Copy additional headers from config
	for k, vv := range lr.config.Headers {
		for _, v := range vv {
			header.Add(k, v)
		}
	}

	// Connect to WebSocket endpoint
	conn, resp, err := dialer.Dial(u.String(), header)
	if err != nil {
		log.Printf("Worker %d: failed to connect to WebSocket: %v", id, err)
		lr.mu.Lock()
		lr.result.FailedConnects++
		lr.mu.Unlock()
		if resp != nil {
			log.Printf("Worker %d: response status: %s", id, resp.Status)
		}
		return
	}
	defer conn.Close()

	// Connection successful
	lr.mu.Lock()
	lr.result.SuccessfulConnects++
	lr.mu.Unlock()

	// Create ticker for sending messages at the specified rate
	ticker := time.NewTicker(time.Duration(1e9 / mpsPerConn)) // Nanoseconds per message
	defer ticker.Stop()

	// Goroutine to receive messages (we just count them for now)
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				// Connection closed or error
				lr.mu.Lock()
				lr.result.TotalErrors++
				lr.mu.Unlock()
				return
			}
			lr.mu.Lock()
			lr.result.TotalMessagesRecv++
			lr.mu.Unlock()
		}
	}()

	// Main loop for sending messages
	for {
		select {
		case <-lr.ctx.Done():
			// Context cancelled, exit
			return
		case <-ticker.C:
			// Time to send a message
			start := time.Now()
			err := conn.WriteMessage(websocket.BinaryMessage, lr.config.Payload)
			if err != nil {
				log.Printf("Worker %d: failed to write message: %v", id, err)
				lr.mu.Lock()
				lr.result.TotalErrors++
				lr.mu.Unlock()
				return
			}

			// Calculate latency (simplified - just measures write time, not roundtrip)
			latencyMs := float64(time.Since(start).Nanoseconds()) / 1e6
			lr.latencyMu.Lock()
			lr.latencies = append(lr.latencies, latencyMs)
			lr.latencyMu.Unlock()

			lr.mu.Lock()
			lr.result.TotalMessagesSent++
			// Update min/max latency
			if latencyMs < lr.result.MinLatencyMs {
				lr.result.MinLatencyMs = latencyMs
			}
			if latencyMs > lr.result.MaxLatencyMs {
				lr.result.MaxLatencyMs = latencyMs
			}
			lr.mu.Unlock()
		}
	}
}

// calculateLatencyStats computes average and percentile latencies.
func (lr *LoadRunner) calculateLatencyStats() {
	if len(lr.latencies) == 0 {
		return
	}

	// Calculate average
	var sum float64
	for _, lat := range lr.latencies {
		sum += lat
	}
	lr.result.AvgLatencyMs = sum / float64(len(lr.latencies))

	// Sort for percentile calculation
	sorted := make([]float64, len(lr.latencies))
	copy(sorted, lr.latencies)
	// Simple bubble sort for demonstration (in production use sort.Float64s)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate P95 latency
	p95Index := int(float64(len(sorted)) * 0.95)
	if p95Index >= len(sorted) {
		p95Index = len(sorted) - 1
	}
	lr.result.P95LatencyMs = sorted[p95Index]
}