package postgres

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/jafarshop/b2bapi/internal/config"
)

// NewConnection creates a new PostgreSQL database connection
func NewConnection(cfg config.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// RunMigrations runs database migrations
// Note: In production, you'd use golang-migrate CLI or library
// For MVP, we'll provide a simple implementation
func RunMigrations(cfg config.DatabaseConfig) error {
	// For now, migrations should be run manually using golang-migrate CLI
	// or a migration tool. This is a placeholder.
	// In production, you'd use: migrate -path ./migrations -database "postgres://..." up
	return nil
}
