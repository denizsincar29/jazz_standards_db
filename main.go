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

	// Get base path from config (e.g., "/jazz" for subpath deployment)
	basePath := config.AppConfig.BasePath
	if basePath != "" && basePath != "/" {
		// Normalize base path: ensure it starts with / and doesn't end with /
		if basePath[0] != '/' {
			basePath = "/" + basePath
		}
		if basePath[len(basePath)-1] == '/' {
			basePath = basePath[:len(basePath)-1]
		}
	} else {
		basePath = ""
	}

	// Create router
	r := mux.NewRouter()

	// Register routes with or without base path
	registerRoutes(r, basePath)

	// Start server
	port := ":" + config.AppConfig.Port
	if basePath != "" {
		log.Printf("Server starting on port %s at base path %s", config.AppConfig.Port, basePath)
		log.Printf("Configure Apache: ProxyPass %s/ http://localhost:%s%s/", basePath, config.AppConfig.Port, basePath)
	} else {
		log.Printf("Server starting on port %s", config.AppConfig.Port)
	}
	log.Fatal(http.ListenAndServe(port, r))
}

func registerRoutes(r *mux.Router, basePath string) {
	staticDir := "./static"

	// Helper function to add base path to route
	route := func(path string) string {
		if basePath == "" {
			return path
		}
		return basePath + path
	}

	// API routes - all under /api
	api := r.PathPrefix(route("/api")).Subrouter()

	// Auth routes (public)
	api.HandleFunc("/register", handlers.Register).Methods("POST")
	api.HandleFunc("/login", handlers.Login).Methods("POST")
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
	if _, err := os.Stat(staticDir); err == nil {
		// Serve service worker (must be accessible at base path for PWA)
		r.HandleFunc(route("/sw.js"), func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/javascript")
			http.ServeFile(w, r, filepath.Join(staticDir, "sw.js"))
		})

		// Serve manifest.json
		r.HandleFunc(route("/manifest.json"), func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/manifest+json")
			http.ServeFile(w, r, filepath.Join(staticDir, "manifest.json"))
		})

		// Serve testapi page (only if TEST_API environment variable is set)
		if os.Getenv("TEST_API") != "" {
			r.HandleFunc(route("/testapi"), func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				http.ServeFile(w, r, filepath.Join(staticDir, "testapi.html"))
			})
			log.Println("API testing page enabled at /testapi (TEST_API environment variable is set)")
		}

		// Serve static files (CSS, JS, images)
		staticFileServer := http.FileServer(http.Dir(staticDir))
		if basePath != "" {
			r.PathPrefix(route("/static/")).Handler(http.StripPrefix(basePath+"/static/", staticFileServer))
		} else {
			r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFileServer))
		}

		// Catch-all: Serve index.html for root and any unmatched routes (SPA support)
		// Handle both exact base path and with trailing slash
		if basePath != "" {
			r.HandleFunc(basePath, handlers.ServeIndexHTML)
			r.HandleFunc(basePath+"/", handlers.ServeIndexHTML)
			// SPA catch-all for any unmatched routes under base path
			r.PathPrefix(basePath + "/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if it's an existing file
				relativePath := r.URL.Path[len(basePath):]
				filePath := filepath.Join(staticDir, relativePath)
				if _, err := os.Stat(filePath); err == nil {
					http.ServeFile(w, r, filePath)
					return
				}
				// Default to index.html for SPA routing
				handlers.ServeIndexHTML(w, r)
			})
		} else {
			r.HandleFunc("/", handlers.ServeIndexHTML)
			// SPA catch-all for any unmatched routes at root
			r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if it's an existing file
				filePath := filepath.Join(staticDir, r.URL.Path)
				if _, err := os.Stat(filePath); err == nil {
					http.ServeFile(w, r, filePath)
					return
				}
				// Default to index.html for SPA routing
				handlers.ServeIndexHTML(w, r)
			})
		}
	}
}
