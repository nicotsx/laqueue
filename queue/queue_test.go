package queue

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Create a temporary database file
	f, err := os.CreateTemp("", "laqueue_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	f.Close()
	dbPath := f.Name()

	// Open the database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Initialize the schema
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
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Return a cleanup function
	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}

func TestEnqueueDequeue(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a queue
	q := New(db, "test_queue")

	// Create a test payload
	type TestPayload struct {
		Message string `json:"message"`
		Value   int    `json:"value"`
	}
	payload := TestPayload{
		Message: "Hello, world!",
		Value:   42,
	}

	// Enqueue the item
	id, err := q.Enqueue(payload)
	if err != nil {
		t.Fatalf("Failed to enqueue item: %v", err)
	}
	if id <= 0 {
		t.Fatalf("Expected a positive ID, got %d", id)
	}

	// Dequeue the item
	item, err := q.Dequeue()
	if err != nil {
		t.Fatalf("Failed to dequeue item: %v", err)
	}
	if item == nil {
		t.Fatal("Expected an item, got nil")
	}

	// Check the item properties
	if item.ID != id {
		t.Errorf("Expected ID %d, got %d", id, item.ID)
	}
	if item.QueueName != "test_queue" {
		t.Errorf("Expected queue name 'test_queue', got '%s'", item.QueueName)
	}
	if item.Status != "processing" {
		t.Errorf("Expected status 'processing', got '%s'", item.Status)
	}
	if item.Attempts != 1 {
		t.Errorf("Expected attempts 1, got %d", item.Attempts)
	}

	// Decode the payload
	var decodedPayload TestPayload
	if err := json.Unmarshal(item.Payload, &decodedPayload); err != nil {
		t.Fatalf("Failed to decode payload: %v", err)
	}
	if decodedPayload.Message != payload.Message {
		t.Errorf("Expected message '%s', got '%s'", payload.Message, decodedPayload.Message)
	}
	if decodedPayload.Value != payload.Value {
		t.Errorf("Expected value %d, got %d", payload.Value, decodedPayload.Value)
	}

	// Mark the item as completed
	if err := q.Complete(id); err != nil {
		t.Fatalf("Failed to mark item as completed: %v", err)
	}

	// Verify that there are no more items to dequeue
	item, err = q.Dequeue()
	if err != nil {
		t.Fatalf("Failed to dequeue item: %v", err)
	}
	if item != nil {
		t.Errorf("Expected no items, got item with ID %d", item.ID)
	}
}

func TestEnqueueWithDelay(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a queue
	q := New(db, "test_queue")

	// Create a test payload
	payload := map[string]string{"message": "delayed item"}

	// Enqueue with a 2-second delay
	id, err := q.EnqueueWithDelay(payload, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to enqueue item with delay: %v", err)
	}

	// Try to dequeue immediately (should be empty)
	item, err := q.Dequeue()
	if err != nil {
		t.Fatalf("Failed to dequeue item: %v", err)
	}
	if item != nil {
		t.Errorf("Expected no items due to delay, got item with ID %d", item.ID)
	}

	// Wait for the delay to pass
	time.Sleep(2100 * time.Millisecond)

	// Now the item should be available
	item, err = q.Dequeue()
	if err != nil {
		t.Fatalf("Failed to dequeue item after delay: %v", err)
	}
	if item == nil {
		t.Fatal("Expected an item after delay, got nil")
	}
	if item.ID != id {
		t.Errorf("Expected ID %d, got %d", id, item.ID)
	}
}

func TestRetryWithDelay(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a queue
	q := New(db, "test_queue")

	// Enqueue an item
	payload := map[string]string{"message": "retry test"}
	id, err := q.Enqueue(payload)
	if err != nil {
		t.Fatalf("Failed to enqueue item: %v", err)
	}

	// Dequeue the item
	item, err := q.Dequeue()
	if err != nil {
		t.Fatalf("Failed to dequeue item: %v", err)
	}
	if item == nil {
		t.Fatal("Expected an item, got nil")
	}

	// Retry with a 1-second delay
	if err := q.RetryWithDelay(id, 1*time.Second); err != nil {
		t.Fatalf("Failed to retry item with delay: %v", err)
	}

	// Try to dequeue immediately (should be empty)
	item, err = q.Dequeue()
	if err != nil {
		t.Fatalf("Failed to dequeue item: %v", err)
	}
	if item != nil {
		t.Errorf("Expected no items due to retry delay, got item with ID %d", item.ID)
	}

	// Wait for the delay to pass
	time.Sleep(1100 * time.Millisecond)

	// Now the item should be available again
	item, err = q.Dequeue()
	if err != nil {
		t.Fatalf("Failed to dequeue item after retry delay: %v", err)
	}
	if item == nil {
		t.Fatal("Expected an item after retry delay, got nil")
	}
	if item.ID != id {
		t.Errorf("Expected ID %d, got %d", id, item.ID)
	}
	if item.Attempts != 2 {
		t.Errorf("Expected attempts 2, got %d", item.Attempts)
	}
}

