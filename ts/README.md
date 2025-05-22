# LaQueue TypeScript Implementation

A simple SQLite-backed queue system implemented in TypeScript.

## Installation

```bash
npm install
```

## Building

```bash
npm run build
```

## Usage

### Basic Usage

```typescript
import { createQueue } from './index';

// Create a queue instance
const queue = createQueue('./queue.db', 'my-queue');

// Enqueue an item
await queue.enqueue({ message: 'Hello' });

// Enqueue with delay (5 seconds)
await queue.enqueueWithDelay({ message: 'Delayed hello' }, 5000);

// Process items
const item = queue.dequeue();
if (item) {
  const payload = JSON.parse(item.payload.toString());
  console.log('Processing:', payload);
  
  // Mark as completed
  queue.complete(item.id);
}

// Check queue size
console.log(`Queue size: ${queue.size()}`);
```

### Worker Pattern

For automated processing of queue items, you can use the Worker class:

```typescript
import { createQueue } from './index';
import { Worker } from './worker';

// Create a queue instance
const queue = createQueue('./queue.db', 'my-queue');

// Create a worker with a handler function
const worker = new Worker(
  queue,
  async (payload) => {
    // Process the payload
    console.log('Processing:', payload);
    await someAsyncWork(payload);
  },
  {
    pollInterval: 1000,    // Check for new items every second
    maxConcurrent: 2,      // Process up to 2 items simultaneously
    maxAttempts: 3,        // Retry failed items up to 3 times
    backoffDelay: 5000,    // Start with 5 second delay for retries
  }
);

// Listen for events
worker.on('processing', ({ id, payload }) => {
  console.log(`Processing item ${id}`);
});

worker.on('completed', ({ id, payload }) => {
  console.log(`Completed item ${id}`);
});

worker.on('failed', ({ id, payload, error }) => {
  console.log(`Failed item ${id}:`, error.message);
});

// Start the worker
worker.start();

// Stop the worker when done
worker.stop();
```

### Error Handling and Retries

```typescript
const item = queue.dequeue();
if (item) {
  try {
    // Process item
    throw new Error('Processing failed');
  } catch (err) {
    // Mark as failed
    queue.fail(item.id);
    
    // Retry after 5 seconds
    queue.retryWithDelay(item.id, 5000);
  }
}
```

### Running the Examples

Basic queue example:
```bash
npx ts-node src/example.ts
```

Worker pattern example:
```bash
npx ts-node src/worker-example.ts
```

## API Reference

### `createQueue(dbPath?: string, queueName: string): LaQueue`

Creates a new queue instance.

- `dbPath`: Path to SQLite database file (default: './laqueue.db')
- `queueName`: Name of the queue

### `LaQueue` Methods

- `enqueue(payload: any): Promise<number>` - Add an item to the queue
- `enqueueWithDelay(payload: any, delay: number): Promise<number>` - Add an item with delay
- `dequeue(): QueueItem | null` - Get and claim the next available item
- `complete(id: number): boolean` - Mark an item as completed
- `fail(id: number): boolean` - Mark an item as failed
- `retryWithDelay(id: number, delay: number): boolean` - Reschedule a failed item
- `size(): number` - Get number of pending items in queue

### `Worker` Class

#### Constructor Options

```typescript
interface WorkerOptions {
  pollInterval?: number;  // How often to check for new items (ms)
  maxConcurrent?: number; // Maximum number of concurrent jobs
  maxAttempts?: number;   // Maximum number of retry attempts
  backoffDelay?: number;  // Base delay for retries (ms)
}
```

#### Events

- `processing` - Emitted when an item starts processing
- `completed` - Emitted when an item is successfully processed
- `failed` - Emitted when an item processing fails
- `error` - Emitted when the worker encounters an error

## License

ISC 