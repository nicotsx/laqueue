package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const (
	defaultDbPath = "./laqueue.db"
)

func main() {
	// Parse command line flags
	dbPath := flag.String("db", defaultDbPath, "Path to SQLite database file")
	flag.Parse()

	// Create the directory for the database if it doesn't exist
	dbDir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create directory for database: %v", err)
	}

	// Open or create the SQLite database
	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize the database with required tables
	if err := initDB(db); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	fmt.Println("LaQueue initialized successfully!")
}

// initDB creates the necessary tables if they don't exist
func initDB(db *sql.DB) error {
	// Create the queue table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS queue_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			queue_name TEXT NOT NULL,
			payload BLOB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			scheduled_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status TEXT DEFAULT 'pending',
			attempts INTEGER DEFAULT 0,
			last_attempt_at TIMESTAMP,
			UNIQUE(id, queue_name)
		);
		CREATE INDEX IF NOT EXISTS idx_queue_status ON queue_items (queue_name, status, scheduled_at);
	`)
	return err
} 