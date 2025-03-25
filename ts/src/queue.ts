import Database from 'better-sqlite3';

export interface QueueItem {
  id: number;
  queueName: string;
  payload: Buffer;
  createdAt: Date;
  scheduledAt: Date;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  attempts: number;
  lastAttemptAt: Date | null;
}

interface DBQueueItem {
  id: number;
  queue_name: string;
  payload: Buffer;
  created_at: string;
  scheduled_at: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  attempts: number;
  last_attempt_at: string | null;
}

export class LaQueue {
  private db: Database.Database;
  private queueName: string;

  constructor(db: Database.Database, queueName: string) {
    this.db = db;
    this.queueName = queueName;
  }

  async enqueue(payload: any): Promise<number> {
    const payloadBuffer = Buffer.from(JSON.stringify(payload));
    const stmt = this.db.prepare(
      'INSERT INTO queue_items (queue_name, payload) VALUES (?, ?)'
    );
    const result = stmt.run(this.queueName, payloadBuffer);
    return result.lastInsertRowid as number;
  }

  async enqueueWithDelay(payload: any, delay: number): Promise<number> {
    const payloadBuffer = Buffer.from(JSON.stringify(payload));
    const scheduledAt = new Date(Date.now() + delay);
    const stmt = this.db.prepare(
      'INSERT INTO queue_items (queue_name, payload, scheduled_at) VALUES (?, ?, ?)'
    );
    const result = stmt.run(this.queueName, payloadBuffer, scheduledAt.toISOString());
    return result.lastInsertRowid as number;
  }

  dequeue(): QueueItem | null {
    const now = new Date().toISOString();

    return this.db.transaction(() => {
      const stmt = this.db.prepare(`
        SELECT id, queue_name, payload, created_at, scheduled_at, status, attempts, last_attempt_at
        FROM queue_items
        WHERE queue_name = ? AND status = 'pending' AND scheduled_at <= ?
        ORDER BY scheduled_at ASC
        LIMIT 1
      `);

      const item = stmt.get(this.queueName, now) as DBQueueItem;

      if (!item) {
        return null;
      }

      const updateStmt = this.db.prepare(`
        UPDATE queue_items
        SET status = 'processing', attempts = attempts + 1, last_attempt_at = ?
        WHERE id = ? AND queue_name = ?
      `);

      updateStmt.run(now, item.id, this.queueName);

      return {
        id: item.id,
        queueName: item.queue_name,
        payload: item.payload,
        createdAt: new Date(item.created_at),
        scheduledAt: new Date(item.scheduled_at),
        status: 'processing' as const,
        attempts: item.attempts + 1,
        lastAttemptAt: item.last_attempt_at ? new Date(item.last_attempt_at) : null
      };
    })();
  }

  complete(id: number): boolean {
    const stmt = this.db.prepare(`
      UPDATE queue_items
      SET status = 'completed'
      WHERE id = ? AND queue_name = ?
    `);
    const result = stmt.run(id, this.queueName);
    return result.changes > 0;
  }

  fail(id: number): boolean {
    const stmt = this.db.prepare(`
      UPDATE queue_items
      SET status = 'failed'
      WHERE id = ? AND queue_name = ?
    `);
    const result = stmt.run(id, this.queueName);
    return result.changes > 0;
  }

  retryWithDelay(id: number, delay: number): boolean {
    const scheduledAt = new Date(Date.now() + delay).toISOString();
    const stmt = this.db.prepare(`
      UPDATE queue_items
      SET status = 'pending', scheduled_at = ?
      WHERE id = ? AND queue_name = ?
    `);
    const result = stmt.run(scheduledAt, id, this.queueName);
    return result.changes > 0;
  }

  size(): number {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      SELECT COUNT(*) as count FROM queue_items
      WHERE queue_name = ? AND status = 'pending' AND scheduled_at <= ?
    `);
    const result = stmt.get(this.queueName, now) as { count: number };
    return result.count;
  }
} 