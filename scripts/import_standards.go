package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Standard struct {
	Title          string `json:"title"`
	Composer       string `json:"composer"`
	Style          string `json:"style"`
	AdditionalNote string `json:"additional_note,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run import_standards.go <json_file>")
	}

	filename := os.Args[1]
	token := os.Getenv("ADMIN_TOKEN")
	apiURL := os.Getenv("API_URL")

	if token == "" {
		log.Fatal("ADMIN_TOKEN environment variable must be set")
	}

	if apiURL == "" {
		apiURL = "http://localhost:8000"
	}

	// Read JSON file
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Parse JSON
	var standards []Standard
	if err := json.Unmarshal(data, &standards); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	log.Printf("Found %d standards to import\n", len(standards))

	// Import standards
	client := &http.Client{Timeout: 10 * time.Second}
	successCount := 0
	errorCount := 0

	for i, standard := range standards {
		log.Printf("[%d/%d] Importing: %s - %s\n", i+1, len(standards), standard.Title, standard.Composer)

		// Create request body
		body, err := json.Marshal(standard)
		if err != nil {
			log.Printf("  ✗ Failed to marshal: %v\n", err)
			errorCount++
			continue
		}

		// Create HTTP request
		req, err := http.NewRequest("POST", apiURL+"/api/jazz_standards", bytes.NewBuffer(body))
		if err != nil {
			log.Printf("  ✗ Failed to create request: %v\n", err)
			errorCount++
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		// Send request
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("  ✗ Request failed: %v\n", err)
			errorCount++
			continue
		}

		// Check response
		if resp.StatusCode == http.StatusCreated {
			log.Printf("  ✓ Success\n")
			successCount++
		} else if resp.StatusCode == http.StatusConflict {
			log.Printf("  ⊘ Already exists\n")
		} else {
			respBody, _ := io.ReadAll(resp.Body)
			log.Printf("  ✗ Failed (status %d): %s\n", resp.StatusCode, string(respBody))
			errorCount++
		}

		resp.Body.Close()

		// Rate limiting to avoid overwhelming the server
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("\nImport complete:")
	log.Printf("  Success: %d", successCount)
	log.Printf("  Errors:  %d", errorCount)
	log.Printf("  Total:   %d", len(standards))
}
