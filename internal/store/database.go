package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Database represents the Atlas PostgreSQL database connection
type Database struct {
	conn *sql.DB
	dsn  string
}

// NewDatabase creates a new database connection to Atlas
func NewDatabase(dsn string) (*Database, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{
		conn: db,
		dsn:  dsn,
	}, nil
}

// Close closes the database connection
func (db *Database) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// DB returns the underlying *sql.DB for queries
func (db *Database) DB() *sql.DB {
	return db.conn
}

// RunMigrations executes all migration files in order
func (db *Database) RunMigrations() error {
	log.Println("Running database migrations...")

	// Create migrations tracking table
	if err := db.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get list of migration files (v2 schema)
	migrations := []string{
		"010_drop_all_tables.sql",
		"011_create_seasons_v2.sql",
		"012_create_teams_v2.sql",
		"013_create_players_v2.sql",
		"014_create_player_team_history.sql",
		"015_create_games_v2.sql",
		"016_create_player_game_stats_v2.sql",
		"017_create_team_game_stats_v2.sql",
		"018_create_odds_mappings_v2.sql",
		"019_create_backfill_jobs_v2.sql",
		"020_create_triggers.sql",
		"021_create_materialized_views.sql",
	}

	// Run each migration
	for _, migration := range migrations {
		if err := db.runMigration(migration); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration, err)
		}
	}

	log.Println("✓ All migrations completed successfully")

	return nil
}

// createMigrationsTable creates a table to track which migrations have been run
func (db *Database) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`
	_, err := db.conn.Exec(query)
	return err
}

// runMigration runs a single migration file if it hasn't been applied yet
func (db *Database) runMigration(filename string) error {
	// Check if already applied
	var exists bool
	err := db.conn.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", filename).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		log.Printf("  ⊘ Skipping %s (already applied)", filename)
		return nil
	}

	// Read migration file from disk
	migrationPath := filepath.Join("infra", "atlas", "migrations", filename)
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		// Try alternate path for Docker container
		migrationPath = filepath.Join("migrations", filename)
		content, err = os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file: %w", err)
		}
	}

	// Execute migration in a transaction
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Record migration as applied
	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", filename); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("  ✓ Applied %s", filename)
	return nil
}

// runSeedData runs seed data files (teams and seasons)
// SeedData inserts initial data into the database
func (db *Database) SeedData() error {
	log.Println("Running seed data...")

	seedFiles := []string{
		"001_teams.sql",
		"002_seasons.sql",
	}

	for _, seedFile := range seedFiles {
		// Read seed file from disk
		seedPath := filepath.Join("infra", "atlas", "seed", seedFile)
		content, err := os.ReadFile(seedPath)
		if err != nil {
			// Try alternate path for Docker container
			seedPath = filepath.Join("seed", seedFile)
			content, err = os.ReadFile(seedPath)
			if err != nil {
				return fmt.Errorf("failed to read seed file %s: %w", seedFile, err)
			}
		}

		if _, err := db.conn.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute seed file %s: %w", seedFile, err)
		}

		log.Printf("  ✓ Seeded %s", seedFile)
	}

	log.Println("✓ Seed data completed successfully")
	return nil
}

// HealthCheck performs a health check on the database
func (db *Database) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return db.conn.PingContext(ctx)
}

