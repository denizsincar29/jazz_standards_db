package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	Port        string
	JWTSecret   string
	Environment string
	BasePath    string
	TestAPI     bool
	// ntfy.sh push notifications for admins
	NtfyURL   string // e.g. https://ntfy.sh
	NtfyTopic string // e.g. jazz_admin_alerts
	NtfyToken string // optional Bearer token if topic is protected
	// WebAuthn
	RPID      string   // Relying Party ID – plain domain, e.g. "example.com"
	RPOrigins []string // Allowed origins, e.g. ["https://example.com"]
}

var AppConfig *Config

// Load reads configuration from environment variables (and optional .env)
func Load() error {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	AppConfig = &Config{
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "jazz"),
		DBPassword:  getEnv("DB_PASSWORD", "jazz"),
		DBName:      getEnv("DB_NAME", "jazz"),
		Port:        getEnv("PORT", "8000"),
		JWTSecret:   getEnv("JWT_SECRET", "change-me-in-production"),
		Environment: getEnv("ENVIRONMENT", "development"),
		BasePath:    getEnv("BASE_PATH", ""),
		TestAPI:     getEnv("TEST_API", "") != "",
		NtfyURL:     getEnv("NTFY_URL", "https://ntfy.sh"),
		NtfyTopic:   getEnv("NTFY_TOPIC", ""),
		NtfyToken:   getEnv("NTFY_TOKEN", ""),
		RPID:        getEnv("WEBAUTHN_RPID", "localhost"),
		RPOrigins:   splitCSV(getEnv("WEBAUTHN_ORIGINS", "http://localhost:8000")),
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// GetDSN returns the PostgreSQL DSN.
func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName)
}

// NtfyEnabled returns true when ntfy notifications are configured.
func (c *Config) NtfyEnabled() bool {
	return c.NtfyTopic != ""
}

// splitCSV splits a comma-separated string into a trimmed slice.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
