package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
	path string
}

// Open opens or creates a SQLite database at the specified path
func Open(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	conn, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(1) // SQLite works best with single connection
	conn.SetMaxIdleConns(1)

	db := &DB{
		conn: conn,
		path: dbPath,
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Conn returns the underlying sql.DB connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// Path returns the database file path
func (db *DB) Path() string {
	return db.path
}

// BeginTx starts a new transaction
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.conn.Begin()
}

// ExecTx executes a function within a transaction
func (db *DB) ExecTx(fn func(*sql.Tx) error) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}

	err = fn(tx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// GetVersion returns the current database schema version
func (db *DB) GetVersion() (int, error) {
	var version int
	err := db.conn.QueryRow("SELECT version FROM schema_version ORDER BY applied_at DESC LIMIT 1").Scan(&version)
	if err == sql.ErrNoRows {
		return 0, nil // No migrations applied yet
	}
	if err != nil {
		return 0, err
	}
	return version, nil
}

// Backup creates a backup of the database
func (db *DB) Backup(backupPath string) error {
	// Ensure backup directory exists
	dir := filepath.Dir(backupPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Read source database
	sourceData, err := os.ReadFile(db.path)
	if err != nil {
		return fmt.Errorf("failed to read database: %w", err)
	}

	// Write backup
	if err := os.WriteFile(backupPath, sourceData, 0644); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}

// Vacuum optimizes the database
func (db *DB) Vacuum() error {
	_, err := db.conn.Exec("VACUUM")
	return err
}

// GetStats returns database statistics
func (db *DB) GetStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	tables := []string{
		"accounts",
		"activity_log",
		"error_log",
		"pack_results",
		"cards_pulled",
		"account_collection",
		"wonder_pick_results",
		"mission_completion",
	}

	for _, table := range tables {
		var count int64
		err := db.conn.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			// Table might not exist yet, skip
			continue
		}
		stats[table] = count
	}

	return stats, nil
}
