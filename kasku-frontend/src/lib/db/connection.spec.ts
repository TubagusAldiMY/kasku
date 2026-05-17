import 'fake-indexeddb/auto';
import { afterEach, describe, expect, it } from 'vitest';
import { openDB, _resetConnection } from './connection';
import { DB_NAME, DB_VERSION, STORE_NAMES } from './schema';

afterEach(async () => {
	_resetConnection();
	await new Promise<void>((resolve, reject) => {
		const req = indexedDB.deleteDatabase(DB_NAME);
		req.onsuccess = () => resolve();
		req.onerror = () => reject(req.error);
		req.onblocked = () => reject(new Error('deleteDatabase blocked'));
	});
});

describe('IndexedDB connection', () => {
	it('opens database at expected version', async () => {
		const db = await openDB();
		expect(db.name).toBe(DB_NAME);
		expect(db.version).toBe(DB_VERSION);
	});

	it('creates all required object stores', async () => {
		const db = await openDB();
		for (const name of STORE_NAMES) {
			expect(db.objectStoreNames.contains(name)).toBe(true);
		}
	});

	it('creates expected indexes on transactions store', async () => {
		const db = await openDB();
		const tx = db.transaction('transactions', 'readonly');
		const store = tx.objectStore('transactions');
		expect(store.indexNames.contains('by_account_id')).toBe(true);
		expect(store.indexNames.contains('by_transaction_date')).toBe(true);
		expect(store.indexNames.contains('by_updated_at')).toBe(true);
	});

	it('uses sync_id as keyPath for sync_queue', async () => {
		const db = await openDB();
		const tx = db.transaction('sync_queue', 'readonly');
		const store = tx.objectStore('sync_queue');
		expect(store.keyPath).toBe('sync_id');
	});

	it('returns the same instance on repeated openDB calls', async () => {
		const db1 = await openDB();
		const db2 = await openDB();
		expect(db1).toBe(db2);
	});
});
