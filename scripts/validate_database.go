//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/denizsincar29/jazz_standards_db/config"
	"github.com/denizsincar29/jazz_standards_db/database"
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	if err := database.Connect(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	report, err := database.ValidateIntegrity()
	if err != nil {
		log.Fatalf("database validation failed: %v", err)
	}

	if report.OK() {
		fmt.Println("Database integrity check passed")
		return
	}

	fmt.Println("Database integrity check failed")
	fmt.Println(report.String())
	os.Exit(1)
}
