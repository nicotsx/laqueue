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

### Running the Example

```bash
npm run start
```

Or run the example file directly:

```bash
npx ts-node src/example.ts
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

## License

ISC 