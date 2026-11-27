import { randomUUID } from 'expo-crypto';
import type { SQLiteDatabase } from 'expo-sqlite';
import type { Feed } from './feed-format.ts';

export type Target = { desktopOrigin: string; siteId: string; siteName: string };
export type Draft = {
  id: string;
  title: string;
  content: string;
  updatedAt: number;
  desktopOrigin: string | null;
  siteId: string | null;
  siteName: string | null;
  cid: string | null;
  publishedAt: number | null;
};
export type DraftSummary = Omit<Draft, 'content'> & { preview: string };
export type Subscription = { id: string; title: string; url: string; refreshedAt: number };
export type EntryKey = { subscriptionId: string; entryId: string };
export type EntrySummary = EntryKey & {
  sourceTitle: string;
  title: string;
  preview: string;
  publishedAt: number | null;
  readAt: number | null;
  starred: 0 | 1;
};
export type Entry = EntrySummary & { content: string; format: 'text' | 'markdown'; url: string };
export type ReadingFilter = 'all' | 'unread' | 'starred';

export async function initializeStorage(db: SQLiteDatabase) {
  db.execSync('PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON;');
  const version = db.getFirstSync<{ user_version: number }>('PRAGMA user_version')?.user_version ?? 0;
  if (version === 1) return;
  if (version !== 0) throw new Error('Unsupported Kinpage database');
  db.withTransactionSync(() => {
    db.execSync(`
      CREATE TABLE drafts (
        id TEXT PRIMARY KEY NOT NULL,
        title TEXT NOT NULL DEFAULT '',
        content TEXT NOT NULL DEFAULT '',
        updatedAt INTEGER NOT NULL,
        desktopOrigin TEXT,
        siteId TEXT,
        siteName TEXT,
        cid TEXT,
        publishedAt INTEGER
      );
      CREATE TABLE subscriptions (
        id TEXT PRIMARY KEY NOT NULL,
        title TEXT NOT NULL,
        url TEXT UNIQUE NOT NULL,
        refreshedAt INTEGER NOT NULL
      );
      CREATE TABLE entries (
        subscriptionId TEXT NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
        entryId TEXT NOT NULL,
        title TEXT NOT NULL,
        content TEXT NOT NULL,
        format TEXT NOT NULL,
        url TEXT NOT NULL,
        publishedAt INTEGER,
        sortAt INTEGER NOT NULL,
        readAt INTEGER,
        starred INTEGER NOT NULL DEFAULT 0,
        PRIMARY KEY (subscriptionId, entryId)
      );
      CREATE INDEX entries_by_date ON entries(sortAt DESC);
      CREATE INDEX entries_by_source_date ON entries(subscriptionId, sortAt DESC);
      PRAGMA user_version = 1;
    `);
  });
}

export function listDrafts(db: SQLiteDatabase): DraftSummary[] {
  return db.getAllSync(`
    SELECT id, title, substr(content, 1, 180) AS preview, updatedAt, desktopOrigin, siteId, siteName, cid, publishedAt
    FROM drafts ORDER BY updatedAt DESC, id
  `);
}

export function createDraft(db: SQLiteDatabase, title = '', content = ''): string {
  const id = randomUUID();
  db.runSync('INSERT INTO drafts (id, title, content, updatedAt) VALUES (?, ?, ?, ?)', id, title, content, Date.now());
  return id;
}

export function getDraft(db: SQLiteDatabase, id: string): Draft | null {
  return db.getFirstSync<Draft>('SELECT * FROM drafts WHERE id = ?', id);
}

export function editDraft(db: SQLiteDatabase, id: string, title: string, content: string): number {
  const updatedAt = Date.now();
  const result = db.runSync(
    'UPDATE drafts SET title = ?, content = ?, updatedAt = ? WHERE id = ? AND siteId IS NULL',
    title, content, updatedAt, id,
  );
  if (result.changes !== 1) throw new Error('Draft is unavailable for editing');
  return updatedAt;
}

export function deleteDraft(db: SQLiteDatabase, id: string) {
  db.runSync('DELETE FROM drafts WHERE id = ?', id);
}

export function beginPublication(db: SQLiteDatabase, id: string, target: Target): boolean {
  return db.runSync(
    'UPDATE drafts SET desktopOrigin = ?, siteId = ?, siteName = ?, updatedAt = ? WHERE id = ? AND siteId IS NULL',
    target.desktopOrigin, target.siteId, target.siteName, Date.now(), id,
  ).changes === 1;
}

export function endPublication(db: SQLiteDatabase, id: string, cid: string) {
  const now = Date.now();
  const saved = db.runSync(
    'UPDATE drafts SET cid = ?, publishedAt = ?, updatedAt = ? WHERE id = ? AND siteId IS NOT NULL AND cid IS NULL',
    cid, now, now, id,
  );
  if (saved.changes !== 1) throw new Error('Publication receipt could not be recorded');
}

export function listSubscriptions(db: SQLiteDatabase): Subscription[] {
  return db.getAllSync('SELECT * FROM subscriptions ORDER BY refreshedAt DESC, title');
}

export function saveFeed(db: SQLiteDatabase, feed: Feed & { url: string }, refreshId?: string) {
  const now = Date.now();
  db.withTransactionSync(() => {
    let id = refreshId;
    if (id) {
      const changed = db.runSync('UPDATE subscriptions SET title = ?, refreshedAt = ? WHERE id = ?', feed.title, now, id);
      if (changed.changes !== 1) return;
    } else {
      id = db.getFirstSync<{ id: string }>('SELECT id FROM subscriptions WHERE url = ?', feed.url)?.id ?? randomUUID();
      db.runSync(`
        INSERT INTO subscriptions(id, title, url, refreshedAt) VALUES (?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET title = excluded.title, refreshedAt = excluded.refreshedAt
      `, id, feed.title, feed.url, now);
    }

    const statement = db.prepareSync(`
      INSERT INTO entries (subscriptionId, entryId, title, content, format, url, publishedAt, sortAt)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?)
      ON CONFLICT(subscriptionId, entryId) DO UPDATE SET
        title = excluded.title, content = excluded.content, format = excluded.format, url = excluded.url,
        publishedAt = coalesce(excluded.publishedAt, entries.publishedAt),
        sortAt = coalesce(excluded.publishedAt, entries.sortAt)
    `);
    try {
      for (const [index, entry] of feed.entries.entries()) {
        statement.executeSync(id, entry.id, entry.title, entry.content, entry.format, entry.url,
          entry.publishedAt, entry.publishedAt ?? now - index);
      }
    } finally {
      statement.finalizeSync();
    }
    db.runSync(`
      DELETE FROM entries WHERE subscriptionId = ? AND starred = 0 AND entryId IN (
        SELECT entryId FROM entries WHERE subscriptionId = ? AND starred = 0
        ORDER BY sortAt DESC, entryId LIMIT -1 OFFSET 150
      )
    `, id, id);
  });
}

export function deleteSubscription(db: SQLiteDatabase, id: string) {
  db.runSync('DELETE FROM subscriptions WHERE id = ?', id);
}

const entryFields = `
  entries.subscriptionId, entryId, subscriptions.title AS sourceTitle, entries.title,
  substr(content, 1, 160) AS preview, publishedAt, readAt, starred
`;

export function listEntries(db: SQLiteDatabase, filter: ReadingFilter, source: string | null): EntrySummary[] {
  const where = { all: '', unread: 'AND readAt IS NULL', starred: 'AND starred = 1' };
  return db.getAllSync(`
    SELECT ${entryFields} FROM entries
    JOIN subscriptions ON subscriptions.id = entries.subscriptionId
    WHERE (? IS NULL OR entries.subscriptionId = ?) ${where[filter]}
    ORDER BY sortAt DESC, entryId, entries.subscriptionId
  `, source, source);
}

export function getEntry(db: SQLiteDatabase, key: EntryKey): Entry | null {
  return db.getFirstSync(`
    SELECT ${entryFields}, content, format, entries.url FROM entries
    JOIN subscriptions ON subscriptions.id = entries.subscriptionId
    WHERE entries.subscriptionId = ? AND entryId = ?
  `, key.subscriptionId, key.entryId);
}

export function markRead(db: SQLiteDatabase, key: EntryKey, read: boolean) {
  db.runSync('UPDATE entries SET readAt = ? WHERE subscriptionId = ? AND entryId = ?',
    read ? Date.now() : null, key.subscriptionId, key.entryId);
}

export function setStarred(db: SQLiteDatabase, key: EntryKey, starred: boolean) {
  db.runSync('UPDATE entries SET starred = ? WHERE subscriptionId = ? AND entryId = ?',
    starred ? 1 : 0, key.subscriptionId, key.entryId);
}
