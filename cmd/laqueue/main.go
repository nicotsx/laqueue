package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nicotsx/laqueue/queue"
)

func main() {
	// Define command line flags
	dbPathFlag := flag.String("db", "./laqueue.db", "Path to SQLite database file")
	queueNameFlag := flag.String("queue", "default", "Name of the queue to operate on")

	// Define subcommands
	enqueueCmd := flag.NewFlagSet("enqueue", flag.ExitOnError)
	enqueueFile := enqueueCmd.String("file", "", "JSON file containing the payload")
	enqueueJson := enqueueCmd.String("json", "", "JSON string containing the payload")
	enqueueDelay := enqueueCmd.Duration("delay", 0, "Delay before processing (e.g. 5s, 1m, 1h)")

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)

	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listStatus := listCmd.String("status", "", "Filter by status (pending, processing, completed, failed)")
	listLimit := listCmd.Int("limit", 10, "Maximum number of items to show")

	// Parse top-level flags
	flag.Parse()

	// Check if a subcommand was provided
	if len(flag.Args()) == 0 {
		printUsage()
		os.Exit(1)
	}

	// Open the database
	db, err := sql.Open("sqlite3", *dbPathFlag)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize the database schema
	if err := initDatabase(db); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Handle subcommands
	switch flag.Args()[0] {
	case "init":
		initCmd.Parse(flag.Args()[1:])
		fmt.Println("Database initialized successfully")

	case "enqueue":
		enqueueCmd.Parse(flag.Args()[1:])

		var payload any

		// Parse the payload from file or command line
		if *enqueueFile != "" {
			data, err := os.ReadFile(*enqueueFile)
			if err != nil {
				log.Fatalf("Failed to read file: %v", err)
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				log.Fatalf("Failed to parse JSON: %v", err)
			}
		} else if *enqueueJson != "" {
			if err := json.Unmarshal([]byte(*enqueueJson), &payload); err != nil {
				log.Fatalf("Failed to parse JSON: %v", err)
			}
		} else {
			log.Fatal("Either -file or -json must be provided")
		}

		// Create a queue and enqueue the item
		q := queue.New(db, *queueNameFlag)

		var id int64
		var err error

		if *enqueueDelay > 0 {
			id, err = q.EnqueueWithDelay(payload, *enqueueDelay)
		} else {
			id, err = q.Enqueue(payload)
		}

		if err != nil {
			log.Fatalf("Failed to enqueue item: %v", err)
		}

		fmt.Printf("Enqueued item with ID %d to queue '%s'\n", id, *queueNameFlag)

	case "list":
		listCmd.Parse(flag.Args()[1:])

		// Build the query
		query := `
			SELECT id, queue_name, payload, created_at, scheduled_at, status, attempts, last_attempt_at
			FROM queue_items
			WHERE queue_name = ?
		`
		args := []any{*queueNameFlag}

		if *listStatus != "" {
			query += " AND status = ?"
			args = append(args, *listStatus)
		}

		query += " ORDER BY id DESC LIMIT ?"
		args = append(args, *listLimit)

		// Execute the query
		rows, err := db.Query(query, args...)
		if err != nil {
			log.Fatalf("Failed to query database: %v", err)
		}
		defer rows.Close()

		// Print the results
		fmt.Printf("Items in queue '%s':\n", *queueNameFlag)
		fmt.Println("ID\tStatus\tAttempts\tCreated At\tScheduled At\tPayload")
		fmt.Println("--\t------\t--------\t----------\t------------\t-------")

		for rows.Next() {
			var item queue.QueueItem
			if err := rows.Scan(
				&item.ID, &item.QueueName, &item.Payload, &item.CreatedAt,
				&item.ScheduledAt, &item.Status, &item.Attempts, &item.LastAttemptAt,
			); err != nil {
				log.Fatalf("Failed to scan row: %v", err)
			}

			// Pretty print the payload
			var prettyPayload interface{}
			json.Unmarshal(item.Payload, &prettyPayload)
			payloadBytes, _ := json.MarshalIndent(prettyPayload, "", "  ")

			fmt.Printf("%d\t%s\t%d\t%s\t%s\t%s\n",
				item.ID,
				item.Status,
				item.Attempts,
				item.CreatedAt.Format("2006-01-02 15:04:05"),
				item.ScheduledAt.Format("2006-01-02 15:04:05"),
				string(payloadBytes),
			)
		}

		if err := rows.Err(); err != nil {
			log.Fatalf("Error iterating rows: %v", err)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: laqueue [global options] command [command options]")
	fmt.Println("\nGlobal Options:")
	flag.PrintDefaults()

	fmt.Println("\nCommands:")
	fmt.Println("  init                   Initialize the database")
	fmt.Println("  enqueue -file FILE     Enqueue an item from a JSON file")
	fmt.Println("  enqueue -json JSON     Enqueue an item from a JSON string")
	fmt.Println("  list                   List items in the queue")
}

func initDatabase(db *sql.DB) error {
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

