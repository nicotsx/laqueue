package queue

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// LaQueue represents a queue backed by SQLite
type LaQueue struct {
	db        *sql.DB
	queueName string
}

// QueueItem represents an item in the queue
type QueueItem struct {
	ID            int64      `json:"id"`
	QueueName     string     `json:"queue_name"`
	Payload       []byte     `json:"payload"`
	CreatedAt     time.Time  `json:"created_at"`
	ScheduledAt   time.Time  `json:"scheduled_at"`
	Status        string     `json:"status"`
	Attempts      int        `json:"attempts"`
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`
}

// New creates a new LaQueue instance
func New(db *sql.DB, queueName string) *LaQueue {
	return &LaQueue{
		db:        db,
		queueName: queueName,
	}
}

// Enqueue adds a new item to the queue
func (q *LaQueue) Enqueue(payload any) (int64, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	result, err := q.db.Exec(
		`INSERT INTO queue_items (queue_name, payload) VALUES (?, ?)`,
		q.queueName, payloadBytes,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// EnqueueWithDelay adds a new item to the queue with a specified delay
func (q *LaQueue) EnqueueWithDelay(payload any, delay time.Duration) (int64, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	scheduledAt := time.Now().Add(delay)

	result, err := q.db.Exec(
		`INSERT INTO queue_items (queue_name, payload, scheduled_at) VALUES (?, ?, ?)`,
		q.queueName, payloadBytes, scheduledAt,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// Dequeue retrieves and claims the next available item from the queue
func (q *LaQueue) Dequeue() (*QueueItem, error) {
	tx, err := q.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var item QueueItem
	now := time.Now()

	err = tx.QueryRow(`
		SELECT id, queue_name, payload, created_at, scheduled_at, status, attempts, last_attempt_at
		FROM queue_items
		WHERE queue_name = ? AND status = 'pending' AND scheduled_at <= ?
		ORDER BY scheduled_at ASC
		LIMIT 1
	`, q.queueName, now).Scan(
		&item.ID, &item.QueueName, &item.Payload, &item.CreatedAt,
		&item.ScheduledAt, &item.Status, &item.Attempts, &item.LastAttemptAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No items in queue
		}
		return nil, err
	}

	// Mark the item as processing
	_, err = tx.Exec(`
		UPDATE queue_items
		SET status = 'processing', attempts = attempts + 1, last_attempt_at = ?
		WHERE id = ? AND queue_name = ?
	`, now, item.ID, q.queueName)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	item.Status = "processing"
	item.Attempts++
	item.LastAttemptAt = &now

	return &item, nil
}

// Complete marks a queue item as completed
func (q *LaQueue) Complete(id int64) error {
	_, err := q.db.Exec(`
		UPDATE queue_items
		SET status = 'completed'
		WHERE id = ? AND queue_name = ?
	`, id, q.queueName)
	return err
}

// Fail marks a queue item as failed
func (q *LaQueue) Fail(id int64) error {
	_, err := q.db.Exec(`
		UPDATE queue_items
		SET status = 'failed'
		WHERE id = ? AND queue_name = ?
	`, id, q.queueName)
	return err
}

// RetryWithDelay reschedules a failed item with a delay
func (q *LaQueue) RetryWithDelay(id int64, delay time.Duration) error {
	scheduledAt := time.Now().Add(delay)
	_, err := q.db.Exec(`
		UPDATE queue_items
		SET status = 'pending', scheduled_at = ?
		WHERE id = ? AND queue_name = ?
	`, scheduledAt, id, q.queueName)
	return err
}

// Size returns the number of pending items in the queue
func (q *LaQueue) Size() (int, error) {
	var count int
	now := time.Now()
	err := q.db.QueryRow(`
		SELECT COUNT(*) FROM queue_items
		WHERE queue_name = ? AND status = 'pending' AND scheduled_at <= ?
	`, q.queueName, now).Scan(&count)
	return count, err
}

