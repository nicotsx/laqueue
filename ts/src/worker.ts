import { LaQueue } from './queue';
import EventEmitter from 'events';

export interface WorkerOptions {
  pollInterval?: number;  // How often to check for new items (ms)
  maxConcurrent?: number; // Maximum number of concurrent jobs
  maxAttempts?: number;   // Maximum number of retry attempts
  backoffDelay?: number;  // Base delay for retries (ms)
}

export interface WorkerEvents {
  'processing': (item: { id: number; payload: any }) => void;
  'completed': (item: { id: number; payload: any }) => void;
  'failed': (item: { id: number; payload: any; error: Error }) => void;
  'error': (error: Error) => void;
}

export declare interface Worker {
  on<U extends keyof WorkerEvents>(event: U, listener: WorkerEvents[U]): this;
  emit<U extends keyof WorkerEvents>(event: U, ...args: Parameters<WorkerEvents[U]>): boolean;
}

export class Worker extends EventEmitter {
  private queue: LaQueue;
  private handler: (payload: any) => Promise<void>;
  private isRunning: boolean = false;
  private activeJobs: Set<number> = new Set();
  private options: Required<WorkerOptions>;

  constructor(
    queue: LaQueue,
    handler: (payload: any) => Promise<void>,
    options: WorkerOptions = {}
  ) {
    super();
    this.queue = queue;
    this.handler = handler;
    this.options = {
      pollInterval: options.pollInterval ?? 1000,
      maxConcurrent: options.maxConcurrent ?? 1,
      maxAttempts: options.maxAttempts ?? 3,
      backoffDelay: options.backoffDelay ?? 5000,
    };
  }

  start(): void {
    if (this.isRunning) return;
    this.isRunning = true;
    this.poll();
  }

  stop(): void {
    this.isRunning = false;
  }

  private async poll(): Promise<void> {
    while (this.isRunning) {
      try {
        // If we've reached max concurrent jobs, wait before polling again
        if (this.activeJobs.size >= this.options.maxConcurrent) {
          await new Promise(resolve => setTimeout(resolve, this.options.pollInterval));
          continue;
        }

        // Try to get a new item from the queue
        const item = this.queue.dequeue();
        if (!item) {
          await new Promise(resolve => setTimeout(resolve, this.options.pollInterval));
          continue;
        }

        // Process the item without waiting
        this.processItem(item.id, JSON.parse(item.payload.toString()), item.attempts);
      } catch (error) {
        this.emit('error', error as Error);
        await new Promise(resolve => setTimeout(resolve, this.options.pollInterval));
      }
    }
  }

  private async processItem(id: number, payload: any, attempts: number): Promise<void> {
    this.activeJobs.add(id);
    this.emit('processing', { id, payload });

    try {
      await this.handler(payload);
      this.queue.complete(id);
      this.emit('completed', { id, payload });
    } catch (error) {
      const shouldRetry = attempts < this.options.maxAttempts;
      this.queue.fail(id);
      this.emit('failed', { id, payload, error: error as Error });

      if (shouldRetry) {
        // Exponential backoff with jitter
        const delay = Math.min(
          this.options.backoffDelay * Math.pow(2, attempts - 1) * (0.5 + Math.random()),
          30000 // Max delay of 30 seconds
        );
        this.queue.retryWithDelay(id, delay);
      }
    } finally {
      this.activeJobs.delete(id);
    }
  }
} 