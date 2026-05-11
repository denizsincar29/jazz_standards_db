package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/denizsincar29/jazz_standards_db/config"
	"github.com/denizsincar29/jazz_standards_db/database"
	"github.com/denizsincar29/jazz_standards_db/handlers"
	"github.com/denizsincar29/jazz_standards_db/middleware"
	"github.com/denizsincar29/jazz_standards_db/models"
	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── Test DB Setup ─────────────────────────────────────────────────────────────

func setupTestDB(t *testing.T) {
	t.Helper()

	host := getEnv("TEST_DB_HOST", "localhost")
	port := getEnv("TEST_DB_PORT", "5432")
	user := getEnv("TEST_DB_USER", "jazz")
	pass := getEnv("TEST_DB_PASSWORD", "jazz")
	dbname := getEnv("TEST_DB_NAME", "jazz_test")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, dbname)

	var err error
	database.DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("Skipping DB tests – could not connect to test database: %v", err)
	}

	if err := database.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// Truncate tables in dependency order
	tables := []string{
		"practice_logs", "user_jazz_standards", "user_categories",
		"pass_keys", "personal_pieces", "composed_tunes",
		"jazz_standards", "users",
	}
	for _, tbl := range tables {
		database.DB.Exec("TRUNCATE TABLE " + tbl + " RESTART IDENTITY CASCADE")
	}

	// Minimal config so ntfy doesn't actually fire
	config.AppConfig = &config.Config{NtfyTopic: ""}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ── Route builder ─────────────────────────────────────────────────────────────

func newRouter() *mux.Router {
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()

	api.HandleFunc("/register", handlers.Register).Methods("POST")
	api.HandleFunc("/login", handlers.Login).Methods("POST")
	api.HandleFunc("/logout", middleware.RequireAuth(handlers.Logout)).Methods("POST")

	api.HandleFunc("/users/me", middleware.RequireAuth(handlers.GetMe)).Methods("GET")
	api.HandleFunc("/users/me", middleware.RequireAuth(handlers.UpdateMe)).Methods("PATCH")
	api.HandleFunc("/users", middleware.RequireAdmin(handlers.ListUsers)).Methods("GET")
	api.HandleFunc("/users/{id:[0-9]+}", middleware.RequireAuth(handlers.DeleteUser)).Methods("DELETE")
	api.HandleFunc("/users/{username}/standards",
		middleware.RequireAuth(handlers.GetPublicUserStandards)).Methods("GET")

	api.HandleFunc("/users/me/passkeys", middleware.RequireAuth(handlers.CreatePassKey)).Methods("POST")
	api.HandleFunc("/users/me/passkeys", middleware.RequireAuth(handlers.ListPassKeys)).Methods("GET")
	api.HandleFunc("/users/me/passkeys/{id:[0-9]+}", middleware.RequireAuth(handlers.DeletePassKey)).Methods("DELETE")

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

	api.HandleFunc("/admin/stats", middleware.RequireAdmin(handlers.AdminStats)).Methods("GET")

	api.HandleFunc("/users/me/standards", middleware.RequireAuth(handlers.ListUserStandards)).Methods("GET")
	api.HandleFunc("/users/me/standards/export", middleware.RequireAuth(handlers.ExportMyStandards)).Methods("GET")
	api.HandleFunc("/users/me/standards/{standard_id:[0-9]+}",
		middleware.RequireAuth(handlers.AddUserStandard)).Methods("POST")
	api.HandleFunc("/users/me/standards/{standard_id:[0-9]+}",
		middleware.RequireAuth(handlers.UpdateUserStandard)).Methods("PUT")
	api.HandleFunc("/users/me/standards/{standard_id:[0-9]+}",
		middleware.RequireAuth(handlers.DeleteUserStandard)).Methods("DELETE")
	api.HandleFunc("/users/me/standards/{standard_id:[0-9]+}/practice",
		middleware.RequireAuth(handlers.LogPractice)).Methods("POST")
	api.HandleFunc("/users/me/practice",
		middleware.RequireAuth(handlers.ListPracticeLogs)).Methods("GET")

	api.HandleFunc("/users/me/categories", middleware.RequireAuth(handlers.ListCategories)).Methods("GET")
	api.HandleFunc("/users/me/categories", middleware.RequireAuth(handlers.CreateCategory)).Methods("POST")
	api.HandleFunc("/users/me/categories/{id:[0-9]+}", middleware.RequireAuth(handlers.UpdateCategory)).Methods("PUT")
	api.HandleFunc("/users/me/categories/{id:[0-9]+}", middleware.RequireAuth(handlers.DeleteCategory)).Methods("DELETE")

	api.HandleFunc("/users/me/pieces", middleware.RequireAuth(handlers.ListPersonalPieces)).Methods("GET")
	api.HandleFunc("/users/me/pieces", middleware.RequireAuth(handlers.CreatePersonalPiece)).Methods("POST")
	api.HandleFunc("/users/me/pieces/{id:[0-9]+}", middleware.RequireAuth(handlers.UpdatePersonalPiece)).Methods("PUT")
	api.HandleFunc("/users/me/pieces/{id:[0-9]+}", middleware.RequireAuth(handlers.DeletePersonalPiece)).Methods("DELETE")

	api.HandleFunc("/users/me/compositions", middleware.RequireAuth(handlers.ListComposedTunes)).Methods("GET")
	api.HandleFunc("/users/me/compositions", middleware.RequireAuth(handlers.CreateComposedTune)).Methods("POST")
	api.HandleFunc("/users/me/compositions/{id:[0-9]+}", middleware.RequireAuth(handlers.UpdateComposedTune)).Methods("PUT")
	api.HandleFunc("/users/me/compositions/{id:[0-9]+}", middleware.RequireAuth(handlers.DeleteComposedTune)).Methods("DELETE")

	api.HandleFunc("/shared_tune", handlers.GetSharedTune).Methods("GET")
	api.HandleFunc("/shared_tune/accept", middleware.RequireAuth(handlers.AcceptSharedTune)).Methods("POST")

	return r
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

func doRequest(r *mux.Router, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func parseBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&m); err != nil {
		t.Fatalf("Failed to parse response body: %v (body=%s)", err, rr.Body.String())
	}
	return m
}

func assertStatus(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rr.Code != expected {
		t.Fatalf("Expected status %d, got %d. Body: %s", expected, rr.Code, rr.Body.String())
	}
}

// ── Auth helpers ──────────────────────────────────────────────────────────────

// registerAndLogin registers a user and returns their session token.
func registerAndLogin(t *testing.T, r *mux.Router, username, name, password string) string {
	t.Helper()
	rr := doRequest(r, "POST", "/api/register", map[string]string{
		"username": username, "name": name, "password": password,
	}, "")
	assertStatus(t, rr, http.StatusCreated)
	body := parseBody(t, rr)
	token, ok := body["token"].(string)
	if !ok || token == "" {
		t.Fatalf("No token in register response: %v", body)
	}
	return token
}

// makeAdmin promotes a user to admin in the DB.
func makeAdmin(t *testing.T, username string) {
	t.Helper()
	if err := database.DB.Model(&models.User{}).
		Where("username = ?", username).
		Update("is_admin", true).Error; err != nil {
		t.Fatalf("makeAdmin failed: %v", err)
	}
}

// createApprovedStandard inserts a standard directly (bypassing approval flow).
func createApprovedStandard(t *testing.T, title, composer, style string) uint {
	t.Helper()
	s := models.JazzStandard{
		Title:    title,
		Composer: composer,
		Style:    models.JazzStyle(style),
		Key:      "F",
		Status:   models.StatusApproved,
	}
	if err := database.DB.Create(&s).Error; err != nil {
		t.Fatalf("createApprovedStandard failed: %v", err)
	}
	return s.ID
}
