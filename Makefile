.PHONY: all server webservice ui run

all: server webservice ui

server:
	go build -o bin/server cmd/server/main.go

webservice:
	go build -o bin/webservice cmd/websocketservice/main.go

ui:
	go build -o bin/ui cmd/ui/main.go

run:
	# Start WebSocket service in background
	./bin/webservice -addr=:8080 -token=test-token &
	WEBSERVICE_PID=$!
	
	# Start API server in background
	./bin/server &
	API_PID=$!
	
	# Start UI server in background
	./bin/ui &
	UI_PID=$!
	
	echo "Services started:"
	echo "  WebSocket service: http://localhost:8080"
	echo "  API server: http://localhost:8080/api"
	echo "  UI: http://localhost:8081"
	echo ""
	echo "Press Ctrl+C to stop all services"
	
	# Wait for interrupt signal
	wait

stop:
	killall webservice server ui 2>/dev/null || true
	echo "All services stopped"