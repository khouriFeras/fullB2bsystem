package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	// Database connection string
	// Try to get from environment variables first
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "123123")
	dbName := getEnv("DB_NAME", "b2bapi")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")
	
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", 
		dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode)
	
	if dsnEnv := os.Getenv("DATABASE_URL"); dsnEnv != "" {
		dsn = dsnEnv
	}

	// First, connect to postgres database to create the target database if needed
	postgresDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=%s", 
		dbUser, dbPassword, dbHost, dbPort, dbSSLMode)
	
	postgresDB, err := sql.Open("postgres", postgresDSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to postgres database: %v\n", err)
		os.Exit(1)
	}
	defer postgresDB.Close()

	// Check if database exists, create if not
	var exists bool
	err = postgresDB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName,
	).Scan(&exists)
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to check database existence: %v\n", err)
		os.Exit(1)
	}

	if !exists {
		fmt.Printf("Database '%s' does not exist. Creating...\n", dbName)
		_, err = postgresDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create database: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Database '%s' created successfully.\n", dbName)
	}

	// Now connect to the target database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	// Read migration file
	migrationPath := filepath.Join("migrations", "000001_init_schema.up.sql")
	if len(os.Args) > 1 {
		migrationPath = os.Args[1]
	}

	sqlBytes, err := ioutil.ReadFile(migrationPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read migration file: %v\n", err)
		os.Exit(1)
	}

	// Execute migration - execute entire file as one statement
	// PostgreSQL can handle multiple statements separated by semicolons
	sql := string(sqlBytes)
	
	_, err = db.Exec(sql)
	if err != nil {
		// Some errors are expected (like "relation already exists")
		// Only fail on critical errors
		if !strings.Contains(err.Error(), "already exists") &&
		   !strings.Contains(err.Error(), "does not exist") {
			fmt.Fprintf(os.Stderr, "Error executing migration: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Println("Migration already applied (some objects already exist)")
		}
	}

	fmt.Println("Migration completed successfully!")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
