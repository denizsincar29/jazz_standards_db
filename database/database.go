package database

import (
	"fmt"
	"log"

	"github.com/denizsincar29/jazz_standards_db/config"
	"github.com/denizsincar29/jazz_standards_db/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect establishes database connection and runs migrations
func Connect() error {
	var err error
	
	dsn := config.AppConfig.GetDSN()
	
	// Configure GORM logger
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}
	
	if config.AppConfig.Environment == "production" {
		gormConfig.Logger = logger.Default.LogMode(logger.Error)
	}
	
	DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	
	log.Println("Database connection established")
	
	// Run auto-migrations
	if err := AutoMigrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	
	return nil
}

// AutoMigrate runs GORM auto-migrations for all models
func AutoMigrate() error {
	log.Println("Running database migrations...")
	
	err := DB.AutoMigrate(
		&models.User{},
		&models.JazzStandard{},
		&models.Category{},
		&models.UserStandard{},
	)
	
	if err != nil {
		return err
	}
	
	// Create unique constraint for user categories
	DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_category_unique ON user_categories(user_id, name)")
	
	log.Println("Migrations completed successfully")
	return nil
}

// Close closes the database connection
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
