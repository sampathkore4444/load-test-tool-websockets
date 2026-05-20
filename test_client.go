package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TestRunResponse struct {
	ID   string `json:"testId"`
	Status string `json:"status"`
}

type TestRun struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Status   string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

func main() {
	apiURL := "http://localhost:8080"

	// Create a test
	resp, err := http.Post(apiURL+"/api/tests", "application/json", nil)
	if err != nil {
		fmt.Printf("Failed to create test: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response status: %d\n", resp.StatusCode)
	fmt.Printf("Response body: %s\n", string(body))
	var createResp TestRunResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		fmt.Printf("Failed to unmarshal create response: %v\n", err)
		return
	}
	fmt.Printf("Created test with ID: %s\n", createResp.ID)

	// Start the test
	startResp, err := http.Post(apiURL+"/api/tests/"+createResp.ID+"/start", "", nil)
	if err != nil {
		fmt.Printf("Failed to start test: %v\n", err)
		return
	}
	defer startResp.Body.Close()
	fmt.Printf("Start test response status: %s\n", startResp.Status)

	// Poll for status
	var testRun TestRun
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		getResp, err := http.Get(apiURL + "/api/tests/" + createResp.ID)
		if err != nil {
			fmt.Printf("Failed to get test: %v\n", err)
			return
		}
		body, _ := io.ReadAll(getResp.Body)
		getResp.Body.Close()
		if err := json.Unmarshal(body, &testRun); err != nil {
			fmt.Printf("Failed to unmarshal test run: %v\n", err)
			return
		}
		fmt.Printf("Test status: %s\n", testRun.Status)
		if testRun.Status == "COMPLETED" || testRun.Status == "CANCELLED" || testRun.Status == "FAILED" {
			break
		}
	}
	fmt.Printf("Test final status: %s\n", testRun.Status)
}