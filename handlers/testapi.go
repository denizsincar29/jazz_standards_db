package handlers

import (
	"net/http"
	"path/filepath"

	"github.com/denizsincar29/jazz_standards_db/config"
)

// ServeTestAPI serves the API testing page (only when TEST_API is enabled)
func ServeTestAPI(w http.ResponseWriter, r *http.Request) {
	if !config.AppConfig.TestAPI {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, filepath.Join("static", "testapi.html"))
}
