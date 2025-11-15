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

	// Get base path from X-Forwarded-Prefix header (set by Apache)
	// or from environment variable
	basePath := r.Header.Get("X-Forwarded-Prefix")
	if basePath == "" {
		basePath = os.Getenv("BASE_PATH")
	}
	
	// Ensure base path ends with / for proper relative URL resolution
	if basePath != "" && basePath != "/" {
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
