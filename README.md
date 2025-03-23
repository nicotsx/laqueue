# LaQueue

LaQueue is a simple, lightweight queue system for Go applications that runs on top of SQLite without requiring a dedicated server. It allows for reliable task processing with features like delayed execution, retries with exponential backoff, and persistent storage.

## Features

- **Serverless**: Works entirely with SQLite, no dedicated server required
- **Simple API**: Easy to use with minimal setup
- **Delayed Execution**: Schedule tasks to run in the future
- **Retry Mechanism**: Automatic retries with exponential backoff
- **Persistent Storage**: Queue items persist across application restarts
- **Multiple Queues**: Create separate queues for different types of tasks

## Installation

```bash
go get github.com/nicotsx/laqueue
```

You'll also need to install the SQLite driver:

```bash
go get github.com/mattn/go-sqlite3
```

## Usage

### Basic Usage

```go
package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nicotsx/laqueue/worker"
)

func main() {
	// Open a connection to SQLite
	db, err := sql.Open("sqlite3", "./queue.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create a worker with a queue named "emails"
	w := worker.New(db, worker.Config{
		QueueName:  "emails",
		Interval:   5 * time.Second,
		MaxRetries: 3,
	}, func(payload []byte) error {
		// Process the payload
		log.Printf("Processing: %s", string(payload))
		return nil
	})

	// Start the worker in a background goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Start(ctx)

	// Add a job to the queue
	id, err := w.Enqueue(map[string]string{
		"to":      "user@example.com",
		"subject": "Hello World",
		"body":    "This is a test email",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Enqueued job with ID: %d", id)

	// Add a job with a delay
	id, err = w.EnqueueWithDelay(map[string]string{
		"to":      "delayed@example.com",
		"subject": "Delayed Email",
		"body":    "This email was delayed by 30 seconds",
	}, 30*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Enqueued delayed job with ID: %d", id)

	// Keep the process running
	select {}
}
```

### Advanced Usage

See the `examples/` directory for more complex examples, including:

- Multiple queues
- Error handling and retries
- Custom job processing

## License

MIT
