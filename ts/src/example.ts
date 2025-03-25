import { createQueue } from './index';

async function example() {
  // Create a queue instance
  const queue = createQueue('./example.db', 'test-queue');

  // Enqueue some items
  console.log('Enqueueing items...');
  await queue.enqueue({ message: 'Hello' });
  await queue.enqueueWithDelay({ message: 'Delayed hello' }, 5000); // 5 seconds delay

  // Check queue size
  console.log(`Queue size: ${queue.size()}`);

  // Process items
  console.log('Processing items...');
  const item = queue.dequeue();
  if (item) {
    console.log('Dequeued item:', {
      id: item.id,
      payload: JSON.parse(item.payload.toString()),
      attempts: item.attempts
    });

    // Mark as completed
    queue.complete(item.id);
  }

  // Wait for delayed item
  console.log('Waiting for delayed item...');
  await new Promise(resolve => setTimeout(resolve, 5000));

  // Process delayed item
  const delayedItem = queue.dequeue();
  if (delayedItem) {
    console.log('Dequeued delayed item:', {
      id: delayedItem.id,
      payload: JSON.parse(delayedItem.payload.toString()),
      attempts: delayedItem.attempts
    });

    // Simulate failure and retry
    queue.fail(delayedItem.id);
    console.log('Failed item, retrying in 2 seconds...');
    queue.retryWithDelay(delayedItem.id, 2000);

    // Wait for retry
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Process retry
    const retryItem = queue.dequeue();
    if (retryItem) {
      console.log('Dequeued retry item:', {
        id: retryItem.id,
        payload: JSON.parse(retryItem.payload.toString()),
        attempts: retryItem.attempts
      });
      queue.complete(retryItem.id);
    }
  }

  console.log(`Final queue size: ${queue.size()}`);
}

// Run the example
example().catch(console.error); 