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
	ProtoSchemaPath string        // Path to uploaded .proto file (optional)
	MessageType     string        // Message type to send (e.g., "PLAYER_MOVE") (optional)
	// OpenCode-specific enhancements
	EnableDetailedMetrics bool   // Enable detailed per-message metrics
	MaxMessageSize      int    // Maximum message size to simulate
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
	// OpenCode-specific enhanced metrics
	MessageSizes       []int    // Sizes of messages sent/received
	ProcessingTimes    []int64  // Processing times on server side (if available)
	ConnectionErrors   map[string]int // Errors by type
	MessagesPerSecond  float64  // Actual messages per second achieved
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
	protoEncoder func([]byte) ([]byte, error) // Function to encode JSON to protobuf
	protoDecoder func([]byte) ([]byte, error) // Function to decode protobuf to JSON
}

// NewLoadRunner creates a new LoadRunner with the given config.
func NewLoadRunner(config LoadRunnerConfig) *LoadRunner {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Initialize protobuf encoder/decoder if schema is provided
	var protoEncoder func([]byte) ([]byte, error)
	var protoDecoder func([]byte) ([]byte, error)
	
	if config.ProtoSchemaPath != "" && config.MessageType != "" {
		// In a real implementation, we would:
		// 1. Load the .proto file using protobuf compiler
		// 2. Generate Go code from it (or use dynamic message creation)
		// 3. Create encoder/decoder functions
		// For this implementation, we'll simulate with a simple function
		protoEncoder = func(jsonData []byte) ([]byte, error) {
			// Simulate protobuf encoding - in reality, this would use the protobuf library
			// to encode the JSON data according to the schema
			return jsonData, nil // Just return the JSON as-is for simulation
		}
		protoDecoder = func(protoData []byte) ([]byte, error) {
			// Simulate protobuf decoding
			return protoData, nil // Just return the data as-is for simulation
		}
	} else {
		// No protobuf schema - just pass through JSON
		protoEncoder = func(jsonData []byte) ([]byte, error) {
			return jsonData, nil
		}
		protoDecoder = func(protoData []byte) ([]byte, error) {
			return protoData, nil
		}
	}
	
	// Set default values for OpenCode-specific enhancements
	if config.MaxMessageSize == 0 {
		config.MaxMessageSize = 4096 // Default 4KB
	}
	
	return &LoadRunner{
		config: config,
		result: LoadRunnerResult{
			StartTime: time.Now(),
			MinLatencyMs: 1e9, // Large initial value
			MessageSizes:    make([]int, 0, 1000),
			ProcessingTimes: make([]int64, 0, 1000),
			ConnectionErrors: make(map[string]int),
		},
		ctx:    ctx,
		cancel: cancel,
		protoEncoder: protoEncoder,
		protoDecoder: protoDecoder,
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

	// Calculate actual messages per second
	duration := lr.result.EndTime.Sub(lr.result.StartTime)
	if duration > 0 {
		lr.result.MessagesPerSecond = float64(lr.result.TotalMessagesSent) / duration.Seconds()
	}

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
			_, msg, err := conn.ReadMessage()
			if err != nil {
				// Connection closed or error
				lr.mu.Lock()
				lr.result.TotalErrors++
				lr.mu.Unlock()
				return
			}
			
			// Record message size
			lr.mu.Lock()
			lr.result.MessageSizes = append(lr.result.MessageSizes, len(msg))
			lr.mu.Unlock()
			
			// Decode message if decoder is available (for logging/metrics)
			if lr.protoDecoder != nil {
				decodedMsg, err := lr.protoDecoder(msg)
				if err != nil {
					// If decoding fails, we still count the message but log the error
					log.Printf("Worker %d: failed to decode message: %v", id, err)
				} else {
					// For now, we just use the decoded message for potential logging
					// In a full implementation, we might extract metrics from it
					_ = decodedMsg // Avoid unused variable error
				}
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
			
			// Encode payload using protobuf if encoder is available
			encodedPayload, err := lr.protoEncoder(lr.config.Payload)
			if err != nil {
				log.Printf("Worker %d: failed to encode payload: %v", id, err)
				lr.mu.Lock()
				lr.result.TotalErrors++
				lr.mu.Unlock()
				return
			}
			
			// Record message size
			lr.mu.Lock()
			lr.result.MessageSizes = append(lr.result.MessageSizes, len(lr.config.Payload))
			lr.mu.Unlock()
			
			err = conn.WriteMessage(websocket.BinaryMessage, encodedPayload)
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