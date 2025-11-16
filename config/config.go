package config

import (
	"fmt"
	"log"
	"os"

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
}

var AppConfig *Config

// Load reads configuration from environment variables
func Load() error {
	// Try to load .env file, but don't fail if it doesn't exist
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
		TestAPI:     getEnv("TEST_API", "") == "true",
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetDSN returns the database connection string
func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName)
}
