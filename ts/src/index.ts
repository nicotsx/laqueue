import Database from 'better-sqlite3';
import path from 'path';
import { LaQueue } from './queue';

const DEFAULT_DB_PATH = './laqueue.db';

function initDB(db: Database.Database): void {
  db.exec(`
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
  `);
}

export function createQueue(dbPath: string = DEFAULT_DB_PATH, queueName: string): LaQueue {
  const db = new Database(dbPath);
  initDB(db);
  return new LaQueue(db, queueName);
}

// If this file is run directly
if (require.main === module) {
  const dbPath = process.argv[2] || DEFAULT_DB_PATH;
  const dbDir = path.dirname(dbPath);
  
  try {
    const db = new Database(dbPath);
    initDB(db);
    console.log('LaQueue initialized successfully!');
    db.close();
  } catch (err) {
    console.error('Failed to initialize LaQueue:', err);
    process.exit(1);
  }
} 