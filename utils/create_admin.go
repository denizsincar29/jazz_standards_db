package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/joho/godotenv"
)

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"uniqueIndex;not null"`
	Name     string `gorm:"not null"`
	Password string `gorm:"not null"`
	IsAdmin  bool   `gorm:"default:false"`
}

func main() {
	fmt.Println("Loading configuration from .env...")
	
	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found, using environment variables")
	}

	fmt.Println("\n=== Create Admin User ===\n")

	// Get input
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)
	if username == "" {
		fmt.Println("Error: Username cannot be empty")
		os.Exit(1)
	}

	fmt.Print("Enter display name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		fmt.Println("Error: Display name cannot be empty")
		os.Exit(1)
	}

	// Get password (hidden)
	fmt.Print("Enter password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("\nError reading password:", err)
		os.Exit(1)
	}
	password := string(passwordBytes)
	fmt.Println()

	if password == "" {
		fmt.Println("Error: Password cannot be empty")
		os.Exit(1)
	}

	// Confirm password
	fmt.Print("Confirm password: ")
	confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("\nError reading confirmation:", err)
		os.Exit(1)
	}
	confirm := string(confirmBytes)
	fmt.Println()

	if password != confirm {
		fmt.Println("Error: Passwords do not match")
		os.Exit(1)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Error hashing password:", err)
		os.Exit(1)
	}

	// Connect to database
	fmt.Println("\nConnecting to database...")
	
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "jazz_user")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "jazz_standards")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("Error connecting to database:", err)
		fmt.Println("\nPlease check your database configuration in .env file:")
		fmt.Println("  DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME")
		os.Exit(1)
	}

	fmt.Println("✓ Connected to PostgreSQL")

	// Auto-migrate (ensure table exists)
	if err := db.AutoMigrate(&User{}); err != nil {
		fmt.Println("Error migrating database:", err)
		os.Exit(1)
	}

	// Create or update admin user
	fmt.Println("\nCreating admin user...")
	
	user := User{
		Username: username,
		Name:     name,
		Password: string(hashedPassword),
		IsAdmin:  true,
	}

	// Use ON CONFLICT to update if exists
	result := db.Model(&User{}).
		Where("username = ?", username).
		Assign(map[string]interface{}{
			"name":     name,
			"password": string(hashedPassword),
			"is_admin": true,
		}).
		FirstOrCreate(&user)

	if result.Error != nil {
		fmt.Println("Error creating admin user:", result.Error)
		os.Exit(1)
	}

	fmt.Printf("✓ Admin user '%s' created successfully!\n", username)

	// Show login URL
	basePath := getEnv("BASE_PATH", "")
	loginURL := "http://localhost:8000/"
	if basePath != "" && basePath != "/" {
		loginURL = fmt.Sprintf("http://localhost:8000%s/", basePath)
	}

	fmt.Println("\nYou can now login at:")
	fmt.Printf("  %s\n", loginURL)
	
	if basePath != "" && basePath != "/" {
		fmt.Printf("  (or %s if deployed on your domain)\n", loginURL)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
