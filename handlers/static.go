package handlers

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var indexTemplate []byte

func init() {
	// Load index.html template at startup
	data, err := ioutil.ReadFile("static/index.html")
	if err != nil {
		log.Printf("Warning: Could not load index.html template: %v", err)
		return
	}
	indexTemplate = data
}

// ServeIndexHTML serves the index.html with BASE_PATH dynamically injected
func ServeIndexHTML(w http.ResponseWriter, r *http.Request) {
	if len(indexTemplate) == 0 {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}

	// Get base path from environment variable
	basePath := os.Getenv("BASE_PATH")
	
	// Normalize base path to end with / for proper relative URL resolution
	if basePath != "" && basePath != "/" {
		// Ensure it starts with / and ends with /
		if basePath[0] != '/' {
			basePath = "/" + basePath
		}
		if !strings.HasSuffix(basePath, "/") {
			basePath += "/"
		}
	} else {
		basePath = "/"
	}

	// Replace {{BASE_PATH}} placeholder with actual base path
	html := bytes.Replace(indexTemplate, []byte("{{BASE_PATH}}"), []byte(basePath), 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(html)
}
