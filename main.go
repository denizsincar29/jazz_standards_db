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
	api.HandleFunc("/jazz_standards", middleware.RequireAdmin(handlers.CreateStandard)).Methods("POST")
	api.HandleFunc("/jazz_standards", middleware.RequireAuth(handlers.ListStandards)).Methods("GET")
	api.HandleFunc("/jazz_standards/{id:[0-9]+}", middleware.RequireAuth(handlers.GetStandard)).Methods("GET")
	api.HandleFunc("/jazz_standards/{id:[0-9]+}", middleware.RequireAdmin(handlers.UpdateStandard)).Methods("PUT")
	api.HandleFunc("/jazz_standards/{id:[0-9]+}", middleware.RequireAdmin(handlers.DeleteStandard)).Methods("DELETE")

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
		// Serve static files
		r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
		
		// Serve index.html for root and catch-all
		r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
		})
		
		// Serve manifest.json
		r.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(staticDir, "manifest.json"))
		})
		
		// Serve service worker
		r.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(staticDir, "sw.js"))
		})
	}

	// Apply base path if configured (for reverse proxy subpath deployments)
	var handler http.Handler = r
	if config.AppConfig.BasePath != "" && config.AppConfig.BasePath != "/" {
		log.Printf("Using base path: %s", config.AppConfig.BasePath)
		mainRouter := mux.NewRouter()
		mainRouter.PathPrefix(config.AppConfig.BasePath).Handler(http.StripPrefix(config.AppConfig.BasePath, r))
		handler = mainRouter
	}

	// Start server
	port := ":" + config.AppConfig.Port
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(port, handler))
}
