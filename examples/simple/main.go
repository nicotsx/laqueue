package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nicotsx/laqueue/worker"
)

// Job represents a simple job payload
type Job struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func main() {
	// Open a connection to the SQLite database
	db, err := sql.Open("sqlite3", "./laqueue.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize the database tables
	_, err = db.Exec(`
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
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up a worker to process jobs from the "example" queue
	w := worker.New(db, worker.Config{
		QueueName:  "example",
		Interval:   2 * time.Second,
		MaxRetries: 3,
	}, processJob)

	// Start the worker in a goroutine
	go w.Start(ctx)

	// Add some jobs to the queue
	for i := 0; i < 5; i++ {
		job := Job{
			ID:      fmt.Sprintf("job-%d", i+1),
			Message: fmt.Sprintf("This is job %d", i+1),
		}

		// Add jobs with increasing delays
		id, err := w.EnqueueWithDelay(job, time.Duration(i)*time.Second)
		if err != nil {
			log.Printf("Failed to enqueue job: %v", err)
		} else {
			log.Printf("Enqueued job %s with ID %d", job.ID, id)
		}
	}

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Wait for signals
	<-signalChan
	log.Println("Received interrupt signal, shutting down...")
	cancel()

	// Allow some time for worker to finish processing
	time.Sleep(1 * time.Second)
	log.Println("Shutdown complete")
}

// processJob handles the job payload
func processJob(payload []byte) error {
	var job Job
	if err := json.Unmarshal(payload, &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	log.Printf("Processing job %s: %s", job.ID, job.Message)

	// Simulate some work
	time.Sleep(500 * time.Millisecond)

	// Randomly fail some jobs to demonstrate retry functionality
	if job.ID == "job-3" {
		return fmt.Errorf("simulated failure for job %s", job.ID)
	}

	log.Printf("Successfully processed job %s", job.ID)
	return nil
}

