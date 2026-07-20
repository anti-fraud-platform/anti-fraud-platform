package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"anti-fraud/internal/migrator"

	_ "github.com/lib/pq"
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/migrate/main.go <command>")
		fmt.Println("Available commands: up, status")
		os.Exit(1)
	}

	command := os.Args[1]

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "antifraud")
	dbPassword := getEnv("DB_PASSWORD", "antifraud123")
	dbName := getEnv("DB_NAME", "analytics")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open DB connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v. Is it running?", err)
	}

	mig := migrator.New(db, "./migrations")

	switch command {
	case "up":
		fmt.Println("Running migrations UP...")
		if err := mig.Up(); err != nil {
			log.Fatalf("Migration UP failed: %v", err)
		}
		fmt.Println("Migrations completed successfully!")

	case "status":
		fmt.Println("Checking migration status...")
		if err := mig.Up(); err != nil {
			log.Fatalf("Migration check failed: %v", err)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: up, status")
		os.Exit(1)
	}
}
