//go:build ignore
// +build ignore

// Import jazz standards into the database via the bulk import API.
//
// Usage:
//   ADMIN_TOKEN=<token> API_URL=http://localhost:8000 go run scripts/import_standards.go scripts/standards_seed.json
//
// Or use the built binary:
//   ./import_standards scripts/standards_seed.json

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	Key            string `json:"key,omitempty"`
	AdditionalNote string `json:"additional_note,omitempty"`
	IrealProLink   string `json:"ireal_pro_link,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run scripts/import_standards.go <json_file>")
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

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	var standards []Standard
	if err := json.Unmarshal(data, &standards); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	log.Printf("Importing %d standards via bulk import endpoint…\n", len(standards))

	body, err := json.Marshal(standards)
	if err != nil {
		log.Fatalf("Failed to marshal: %v", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", apiURL+"/api/jazz_standards/bulk_import", bytes.NewBuffer(body))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)
	fmt.Printf("Import complete:\n  Imported: %.0f\n  Skipped:  %.0f\n  Failed:   %.0f\n",
		result["imported"], result["skipped"], result["failed"])
}
