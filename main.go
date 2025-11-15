package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/denizsincar29/jazz_standards_db/config"
	"github.com/denizsincar29/jazz_standards_db/database"
	"github.com/denizsincar29/jazz_standards_db/handlers"
	"github.com/denizsincar29/jazz_standards_db/middleware"
	"github.com/gorilla/mux"
)

func main() {
	// Load configuration
	if err := config.Load(); err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Connect to database
	if err := database.Connect(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Create router
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Auth routes (public)
	api.HandleFunc("/register", handlers.Register).Methods("POST")
	api.HandleFunc("/login", handlers.Login).Methods("POST")
	api.HandleFunc("/admin", handlers.CreateAdmin).Methods("POST")
	api.HandleFunc("/logout", middleware.RequireAuth(handlers.Logout)).Methods("POST")

	// User routes
	api.HandleFunc("/users/me", middleware.RequireAuth(handlers.GetMe)).Methods("GET")
	api.HandleFunc("/users", middleware.RequireAdmin(handlers.ListUsers)).Methods("GET")
	api.HandleFunc("/users/{id:[0-9]+}", middleware.RequireAuth(handlers.DeleteUser)).Methods("DELETE")

	// Jazz standards routes
	api.HandleFunc("/jazz_standards", middleware.RequireAuth(handlers.CreateStandard)).Methods("POST")
	api.HandleFunc("/jazz_standards", middleware.RequireAuth(handlers.ListStandards)).Methods("GET")
	api.HandleFunc("/jazz_standards/pending", middleware.RequireAdmin(handlers.ListPendingStandards)).Methods("GET")
	api.HandleFunc("/jazz_standards/{id:[0-9]+}", middleware.RequireAuth(handlers.GetStandard)).Methods("GET")
	api.HandleFunc("/jazz_standards/{id:[0-9]+}", middleware.RequireAdmin(handlers.UpdateStandard)).Methods("PUT")
	api.HandleFunc("/jazz_standards/{id:[0-9]+}", middleware.RequireAdmin(handlers.DeleteStandard)).Methods("DELETE")
	api.HandleFunc("/jazz_standards/{id:[0-9]+}/approve", middleware.RequireAdmin(handlers.ApproveStandard)).Methods("POST")
	api.HandleFunc("/jazz_standards/{id:[0-9]+}/reject", middleware.RequireAdmin(handlers.RejectStandard)).Methods("POST")

	// User standards routes
	api.HandleFunc("/users/me/standards", middleware.RequireAuth(handlers.ListUserStandards)).Methods("GET")
	api.HandleFunc("/users/me/standards/{standard_id:[0-9]+}", middleware.RequireAuth(handlers.AddUserStandard)).Methods("POST")
	api.HandleFunc("/users/me/standards/{standard_id:[0-9]+}", middleware.RequireAuth(handlers.UpdateUserStandard)).Methods("PUT")
	api.HandleFunc("/users/me/standards/{standard_id:[0-9]+}", middleware.RequireAuth(handlers.DeleteUserStandard)).Methods("DELETE")

	// Category routes
	api.HandleFunc("/users/me/categories", middleware.RequireAuth(handlers.ListCategories)).Methods("GET")
	api.HandleFunc("/users/me/categories", middleware.RequireAuth(handlers.CreateCategory)).Methods("POST")
	api.HandleFunc("/users/me/categories/{id:[0-9]+}", middleware.RequireAuth(handlers.UpdateCategory)).Methods("PUT")
	api.HandleFunc("/users/me/categories/{id:[0-9]+}", middleware.RequireAuth(handlers.DeleteCategory)).Methods("DELETE")

	// Serve static files (PWA)
	staticDir := "./static"
	if _, err := os.Stat(staticDir); err == nil {
		// Serve service worker (must be at root for PWA)
		r.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/javascript")
			http.ServeFile(w, r, filepath.Join(staticDir, "sw.js"))
		})
		
		// Serve manifest.json
		r.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/manifest+json")
			http.ServeFile(w, r, filepath.Join(staticDir, "manifest.json"))
		})
		
		// Serve static files (CSS, JS, images)
		// Create a file server for the static directory
		staticFileServer := http.FileServer(http.Dir(staticDir))
		r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFileServer))
		
		// Catch-all: Serve index.html for root and any unmatched routes (SPA support)
		// This must be registered LAST to allow other routes to match first
		r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only serve index.html if the path doesn't start with /api
			// and doesn't point to an actual file
			if r.URL.Path != "/" && r.URL.Path != "" {
				// Check if file exists
				filePath := filepath.Join(staticDir, r.URL.Path)
				if _, err := os.Stat(filePath); err == nil {
					http.ServeFile(w, r, filePath)
					return
				}
			}
			// Default to index.html for SPA routing
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
		})
	}

	// Note: BASE_PATH is not used in the application routing.
	// For reverse proxy subpath deployments (e.g., /jazz), Apache should strip the path:
	//   ProxyPass /jazz http://localhost:8000/
	// This way the Go app always sees requests at root, and no special handling is needed.
	// The BASE_PATH config can be removed or used only for generating URLs if needed.

	// Start server
	port := ":" + config.AppConfig.Port
	log.Printf("Server starting on port %s", port)
	if config.AppConfig.BasePath != "" && config.AppConfig.BasePath != "/" {
		log.Printf("Note: BASE_PATH '%s' is set but not used by the application", config.AppConfig.BasePath)
		log.Printf("Configure Apache to strip the path: ProxyPass %s http://localhost:%s/", config.AppConfig.BasePath, config.AppConfig.Port)
	}
	log.Fatal(http.ListenAndServe(port, r))
}
