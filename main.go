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
	if err := config.Load(); err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	if err := database.Connect(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	basePath := normalizeBasePath(config.AppConfig.BasePath)

	r := mux.NewRouter()
	registerRoutes(r, basePath)

	port := ":" + config.AppConfig.Port
	if basePath != "" {
		log.Printf("Server starting on port %s at base path %s", config.AppConfig.Port, basePath)
		log.Printf("Apache config hint:  ProxyPass %s/ http://localhost:%s%s/", basePath, config.AppConfig.Port, basePath)
	} else {
		log.Printf("Server starting on port %s", config.AppConfig.Port)
	}
	log.Fatal(http.ListenAndServe(port, r))
}

func normalizeBasePath(bp string) string {
	if bp == "" || bp == "/" {
		return ""
	}
	if bp[0] != '/' {
		bp = "/" + bp
	}
	for len(bp) > 1 && bp[len(bp)-1] == '/' {
		bp = bp[:len(bp)-1]
	}
	return bp
}

func registerRoutes(r *mux.Router, basePath string) {
	staticDir := "./static"

	route := func(path string) string {
		return basePath + path
	}

	api := r.PathPrefix(route("/api")).Subrouter()

	// ── Auth ───────────────────────────────────────────────────────────────────
	api.HandleFunc("/register", handlers.Register).Methods("POST")
	api.HandleFunc("/login", handlers.Login).Methods("POST")
	api.HandleFunc("/logout", middleware.RequireAuth(handlers.Logout)).Methods("POST")
	// WebAuthn passkey authentication (no prior session needed)
	api.HandleFunc("/auth/passkey/begin", handlers.BeginAuthentication).Methods("POST")
	api.HandleFunc("/auth/passkey/finish", handlers.FinishAuthentication).Methods("POST")

	// ── Users ──────────────────────────────────────────────────────────────────
	api.HandleFunc("/users/me", middleware.RequireAuth(handlers.GetMe)).Methods("GET")
	api.HandleFunc("/users/me", middleware.RequireAuth(handlers.UpdateMe)).Methods("PATCH")
	api.HandleFunc("/users", middleware.RequireAdmin(handlers.ListUsers)).Methods("GET")
	api.HandleFunc("/users/{id:[0-9]+}", middleware.RequireAuth(handlers.DeleteUser)).Methods("DELETE")

	// ── Pass Keys ─────────────────────────────────────────────────────────────
	// WebAuthn registration (requires session)
	api.HandleFunc("/users/me/passkeys/begin", middleware.RequireAuth(handlers.BeginRegistration)).Methods("POST")
	api.HandleFunc("/users/me/passkeys/finish", middleware.RequireAuth(handlers.FinishRegistration)).Methods("POST")
	// List / delete (unchanged)
	api.HandleFunc("/users/me/passkeys", middleware.RequireAuth(handlers.ListPassKeys)).Methods("GET")
	api.HandleFunc("/users/me/passkeys/{id:[0-9]+}", middleware.RequireAuth(handlers.DeletePassKey)).Methods("DELETE")

	// ── Jazz Standards ────────────────────────────────────────────────────────
	api.HandleFunc("/jazz_standards", middleware.RequireAuth(handlers.CreateStandard)).Methods("POST")
	api.HandleFunc("/jazz_standards", middleware.RequireAuth(handlers.ListStandards)).Methods("GET")
	api.HandleFunc("/jazz_standards/random", middleware.RequireAuth(handlers.RandomStandard)).Methods("GET")
	api.HandleFunc("/jazz_standards/pending", middleware.RequireAdmin(handlers.ListPendingStandards)).Methods("GET")
	api.HandleFunc("/jazz_standards/bulk_import", middleware.RequireAdmin(handlers.BulkImportStandards)).Methods("POST")
	api.HandleFunc("/jazz_standards/{id:[0-9]+}", middleware.RequireAuth(handlers.GetStandard)).Methods("GET")
	api.HandleFunc("/jazz_standards/{id:[0-9]+}", middleware.RequireAdmin(handlers.UpdateStandard)).Methods("PUT")
	api.HandleFunc("/jazz_standards/{id:[0-9]+}", middleware.RequireAdmin(handlers.DeleteStandard)).Methods("DELETE")
	api.HandleFunc("/jazz_standards/{id:[0-9]+}/approve", middleware.RequireAdmin(handlers.ApproveStandard)).Methods("POST")
	api.HandleFunc("/jazz_standards/{id:[0-9]+}/reject", middleware.RequireAdmin(handlers.RejectStandard)).Methods("POST")

	// ── Admin stats ───────────────────────────────────────────────────────────
	api.HandleFunc("/admin/stats", middleware.RequireAdmin(handlers.AdminStats)).Methods("GET")

	// ── User Standards (my list) ──────────────────────────────────────────────
	api.HandleFunc("/users/me/standards", middleware.RequireAuth(handlers.ListUserStandards)).Methods("GET")
	api.HandleFunc("/users/me/standards/export", middleware.RequireAuth(handlers.ExportMyStandards)).Methods("GET")
	api.HandleFunc("/users/me/standards/{standard_id:[0-9]+}",
		middleware.RequireAuth(handlers.AddUserStandard)).Methods("POST")
	api.HandleFunc("/users/me/standards/{standard_id:[0-9]+}",
		middleware.RequireAuth(handlers.UpdateUserStandard)).Methods("PUT")
	api.HandleFunc("/users/me/standards/{standard_id:[0-9]+}",
		middleware.RequireAuth(handlers.DeleteUserStandard)).Methods("DELETE")

	// Public profile – another user's list (only if public_profile=true)
	api.HandleFunc("/users/{username}/standards",
		middleware.RequireAuth(handlers.GetPublicUserStandards)).Methods("GET")

	// ── Practice Logs ─────────────────────────────────────────────────────────
	api.HandleFunc("/users/me/standards/{standard_id:[0-9]+}/practice",
		middleware.RequireAuth(handlers.LogPractice)).Methods("POST")
	api.HandleFunc("/users/me/practice",
		middleware.RequireAuth(handlers.ListPracticeLogs)).Methods("GET")

	// ── Categories ────────────────────────────────────────────────────────────
	api.HandleFunc("/users/me/categories", middleware.RequireAuth(handlers.ListCategories)).Methods("GET")
	api.HandleFunc("/users/me/categories", middleware.RequireAuth(handlers.CreateCategory)).Methods("POST")
	api.HandleFunc("/users/me/categories/{id:[0-9]+}", middleware.RequireAuth(handlers.UpdateCategory)).Methods("PUT")
	api.HandleFunc("/users/me/categories/{id:[0-9]+}", middleware.RequireAuth(handlers.DeleteCategory)).Methods("DELETE")

	// ── Personal Pieces (rare / local tunes) ─────────────────────────────────
	api.HandleFunc("/users/me/pieces", middleware.RequireAuth(handlers.ListPersonalPieces)).Methods("GET")
	api.HandleFunc("/users/me/pieces", middleware.RequireAuth(handlers.CreatePersonalPiece)).Methods("POST")
	api.HandleFunc("/users/me/pieces/{id:[0-9]+}", middleware.RequireAuth(handlers.UpdatePersonalPiece)).Methods("PUT")
	api.HandleFunc("/users/me/pieces/{id:[0-9]+}", middleware.RequireAuth(handlers.DeletePersonalPiece)).Methods("DELETE")

	// ── Composed Tunes ────────────────────────────────────────────────────────
	api.HandleFunc("/users/me/compositions", middleware.RequireAuth(handlers.ListComposedTunes)).Methods("GET")
	api.HandleFunc("/users/me/compositions", middleware.RequireAuth(handlers.CreateComposedTune)).Methods("POST")
	api.HandleFunc("/users/me/compositions/{id:[0-9]+}", middleware.RequireAuth(handlers.UpdateComposedTune)).Methods("PUT")
	api.HandleFunc("/users/me/compositions/{id:[0-9]+}", middleware.RequireAuth(handlers.DeleteComposedTune)).Methods("DELETE")

	// ── Shared Tune (public endpoint) ─────────────────────────────────────────
	// GET  /api/shared_tune?id=<shareID>        – view a shared composition
	// POST /api/shared_tune/accept?id=<shareID> – accept it into your personal pieces
	api.HandleFunc("/shared_tune", handlers.GetSharedTune).Methods("GET")
	api.HandleFunc("/shared_tune/accept", middleware.RequireAuth(handlers.AcceptSharedTune)).Methods("POST")

	// ── Static / PWA ──────────────────────────────────────────────────────────
	if _, err := os.Stat(staticDir); err == nil {
		r.HandleFunc(route("/sw.js"), func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/javascript")
			http.ServeFile(w, req, filepath.Join(staticDir, "sw.js"))
		})
		r.HandleFunc(route("/manifest.json"), func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/manifest+json")
			http.ServeFile(w, req, filepath.Join(staticDir, "manifest.json"))
		})

		if os.Getenv("TEST_API") != "" {
			r.HandleFunc(route("/testapi"), func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				http.ServeFile(w, req, filepath.Join(staticDir, "testapi.html"))
			})
			log.Println("API testing page enabled at /testapi")
		}

		staticFileServer := http.FileServer(http.Dir(staticDir))
		if basePath != "" {
			r.PathPrefix(route("/static/")).Handler(
				http.StripPrefix(basePath+"/static/", staticFileServer))
		} else {
			r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFileServer))
		}

		if basePath != "" {
			r.HandleFunc(basePath, handlers.ServeIndexHTML)
			r.HandleFunc(basePath+"/", handlers.ServeIndexHTML)
			r.PathPrefix(basePath + "/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				rel := req.URL.Path[len(basePath):]
				fp := filepath.Join(staticDir, rel)
				if _, err := os.Stat(fp); err == nil {
					http.ServeFile(w, req, fp)
					return
				}
				handlers.ServeIndexHTML(w, req)
			})
		} else {
			r.HandleFunc("/", handlers.ServeIndexHTML)
			r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				fp := filepath.Join(staticDir, req.URL.Path)
				if _, err := os.Stat(fp); err == nil {
					http.ServeFile(w, req, fp)
					return
				}
				handlers.ServeIndexHTML(w, req)
			})
		}
	}
}
