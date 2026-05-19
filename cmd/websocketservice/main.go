package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	addr      = flag.String("addr", ":8080", "http service address")
	token     = flag.String("token", "test-token", "authentication token")
	grpcAddr  = flag.String("grpc-addr", ":50051", "address of gRPC backend service")
	logger    *zap.Logger
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for simplicity
		},
	}
)

func init() {
	// Initialize logger
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	var err error
	logger, err = config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
}

func main() {
	flag.Parse()
	logger.Info("Starting WebSocket service",
		zap.String("addr", *addr),
		zap.String("token", *token),
		zap.String("grpc-addr", *grpcAddr))

	http.HandleFunc("/ws", serveWs)
	http.HandleFunc("/health", healthHandler)

	// Create HTTP server
	srv := &http.Server{
		Addr:    *addr,
		Handler: nil,
	}

	// Create a channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Run the server in a goroutine
	go func() {
		logger.Info("Starting HTTP server", zap.String("addr", *addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	<-stop
	logger.Info("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// serveWs handles WebSocket requests from clients.
func serveWs(w http.ResponseWriter, r *http.Request) {
	// Check authentication token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing authorization header", http.StatusUnauthorized)
		return
	}

	// Expected format: "Bearer <token>"
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
		return
	}

	tokenValue := authHeader[7:]
	if tokenValue != *token {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Upgrade to WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade to WebSocket", zap.Error(err))
		return
	}
	defer conn.Close()

	logger.Info("New WebSocket connection",
		zap.String("remote_addr", conn.RemoteAddr().String()))

	// Handle messages from client
	for {
		// Read message from client
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		// Log received message
		logger.Info("Received message",
			zap.String("type", fmt.Sprintf("%d", msgType)),
			zap.ByteString("payload", msg))

		// Process the message - in a real app, this would call gRPC services
		response, err := processMessage(msg)
		if err != nil {
			logger.Error("Failed to process message", zap.Error(err))
			// Send error back to client
			conn.WriteMessage(msgType, []byte(`{"error":"internal server error"}`))
			continue
		}

		// Send response back to client
		if err := conn.WriteMessage(msgType, response); err != nil {
			logger.Error("Failed to write message", zap.Error(err))
			break
		}
	}

	logger.Info("WebSocket connection closed",
		zap.String("remote_addr", conn.RemoteAddr().String()))
}

// processMessage simulates processing a protobuf message by calling gRPC services.
// In a real implementation, this would make actual gRPC calls.
func processMessage(msg []byte) ([]byte, error) {
	// Simulate some processing time (variable to simulate different gRPC methods)
	processingTime := time.Duration(5+rand.Intn(15)) * time.Millisecond
	time.Sleep(processingTime)

	// Simulate different gRPC responses based on the message content
	// In a real implementation, we would parse the protobuf message and route to appropriate gRPC method
	var response []byte

	// Simple simulation: if we can unmarshal as JSON, check for event type
	var msgMap map[string]interface{}
	if err := json.Unmarshal(msg, &msgMap); err == nil {
		if eventType, ok := msgMap["event"].(string); ok {
			switch eventType {
			case "PLAYER_MOVE":
				response = []byte(`{"event":"PLAYER_MOVE_ACK","timestamp":"` + time.Now().Format(time.RFC3339) + `","status":"success"}`)
			case "PLAYER_SHOOT":
				response = []byte(`{"event":"PLAYER_SHOOT_ACK","timestamp":"` + time.Now().Format(time.RFC3339) + `","hit":true,"damage":25}`)
			case "PLAYER_CHAT":
				response = []byte(`{"event":"PLAYER_CHAT_ACK","timestamp":"` + time.Now().Format(time.RFC3339) + `","messageId":"msg_` + fmt.Sprintf("%d", time.Now().UnixNano()) + `"}`)
			default:
				// Generic ACK for unknown events
				response = []byte(`{"event":"` + eventType + `_ACK","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`)
			}
		} else {
			// If no event field, just echo with timestamp
			response = []byte(`{"echo":` + string(msg) + `,"timestamp":"` + time.Now().Format(time.RFC3339) + `"}`)
		}
	} else {
		// If not JSON, treat as protobuf and simulate decoding/processing
		// In reality, we would use protobuf library to decode
		response = []byte(`{"decoded_protobuf":true,"timestamp":"` + time.Now().Format(time.RFC3339) + `","size":` + fmt.Sprintf("%d", len(msg)) + `}`)
	}

	return response, nil
}