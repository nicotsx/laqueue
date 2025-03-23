package worker

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/nicotsx/laqueue/queue"
)

// ProcessFunc is a function that processes a queue item
type ProcessFunc func(payload []byte) error

// Worker represents a worker that processes queue items
type Worker struct {
	db          *sql.DB
	queue       *queue.LaQueue
	queueName   string
	processFunc ProcessFunc
	interval    time.Duration
	maxRetries  int
}

// Config holds configuration options for the worker
type Config struct {
	QueueName  string
	Interval   time.Duration
	MaxRetries int
}

// New creates a new Worker instance
func New(db *sql.DB, config Config, processFunc ProcessFunc) *Worker {
	if config.Interval == 0 {
		config.Interval = 5 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &Worker{
		db:          db,
		queue:       queue.New(db, config.QueueName),
		queueName:   config.QueueName,
		processFunc: processFunc,
		interval:    config.Interval,
		maxRetries:  config.MaxRetries,
	}
}

// Start begins the worker polling the queue for items to process
func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Printf("Starting worker for queue: %s", w.queueName)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker stopped: %v", ctx.Err())
			return
		case <-ticker.C:
			w.processNext()
		}
	}
}

// processNext attempts to process the next item in the queue
func (w *Worker) processNext() {
	item, err := w.queue.Dequeue()
	if err != nil {
		log.Printf("Error dequeueing item: %v", err)
		return
	}
	if item == nil {
		// No items to process
		return
	}

	log.Printf("Processing item %d from queue", item.ID)

	if err := w.processFunc(item.Payload); err != nil {
		log.Printf("Error processing item %d: %v", item.ID, err)

		if item.Attempts >= w.maxRetries {
			log.Printf("Item %d has failed %d times, marking as failed", item.ID, item.Attempts)
			if err := w.queue.Fail(item.ID); err != nil {
				log.Printf("Error marking item as failed: %v", err)
			}
		} else {
			// Exponential backoff for retries
			delay := time.Duration(1<<uint(item.Attempts)) * time.Second
			log.Printf("Rescheduling item %d for retry in %v", item.ID, delay)
			if err := w.queue.RetryWithDelay(item.ID, delay); err != nil {
				log.Printf("Error rescheduling item: %v", err)
			}
		}
		return
	}

	// Mark the item as completed
	if err := w.queue.Complete(item.ID); err != nil {
		log.Printf("Error marking item as completed: %v", err)
	}
}

// Enqueue adds a new item to the queue
func (w *Worker) Enqueue(payload any) (int64, error) {
	return w.queue.Enqueue(payload)
}

// EnqueueWithDelay adds a new item to the queue with a specified delay
func (w *Worker) EnqueueWithDelay(payload any, delay time.Duration) (int64, error) {
	return w.queue.EnqueueWithDelay(payload, delay)
}

