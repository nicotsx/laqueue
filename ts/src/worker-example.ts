import { createQueue } from './index';
import { Worker } from './worker';

async function workerExample() {
  // Create a queue instance
  const queue = createQueue(':memory:', 'test-queue');

  // Create a worker with a handler function
  const worker = new Worker(
    queue,
    async (payload) => {
      console.log('Processing payload:', payload);
      
      // Simulate some work
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      // Randomly fail some items to demonstrate retry behavior
      if (Math.random() < 0.3) {
        throw new Error('Random processing failure');
      }
    },
    {
      pollInterval: 500,      // Check for new items every 500ms
      maxConcurrent: 2,       // Process up to 2 items simultaneously
      maxAttempts: 3,         // Retry failed items up to 3 times
      backoffDelay: 1000,     // Start with 1 second delay for retries
    }
  );

  // Set up event handlers
  worker.on('processing', ({ id, payload }) => {
    console.log(`[${new Date().toISOString()}] Processing item ${id}:`, payload);
  });

  worker.on('completed', ({ id, payload }) => {
    console.log(`[${new Date().toISOString()}] Completed item ${id}:`, payload);
  });

  worker.on('failed', ({ id, payload, error }) => {
    console.log(`[${new Date().toISOString()}] Failed item ${id}:`, payload);
    console.log('Error:', error.message);
  });

  worker.on('error', (error) => {
    console.error('Worker error:', error);
  });

  // Start the worker
  console.log('Starting worker...');
  worker.start();

  // Enqueue some test items
  console.log('Enqueueing test items...');
  for (let i = 1; i <= 5; i++) {
    await queue.enqueue({ message: `Test message ${i}` });
  }

  // Keep the process running for a while to see the worker in action
  await new Promise(resolve => setTimeout(resolve, 20000));

  // Stop the worker
  console.log('Stopping worker...');
  worker.stop();
}

// Run the example
workerExample().catch(console.error); 