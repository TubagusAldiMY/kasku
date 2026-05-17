import { openDB, requestToPromise, transactionToPromise } from '../connection';
import type { SyncQueueRow, SyncOperationStatus } from '../schema';

const STORE = 'sync_queue';

async function enqueue(row: Omit<SyncQueueRow, 'status' | 'attempts'>): Promise<void> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readwrite');
	tx.objectStore(STORE).put({
		...row,
		status: 'PENDING' satisfies SyncOperationStatus,
		attempts: 0
	} satisfies SyncQueueRow);
	await transactionToPromise(tx);
}

async function pending(limit?: number): Promise<SyncQueueRow[]> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readonly');
	const idx = tx.objectStore(STORE).index('by_created_at');
	const rows = await requestToPromise(idx.getAll() as IDBRequest<SyncQueueRow[]>);
	await transactionToPromise(tx);
	const pendingOnly = rows.filter((r) => r.status !== 'IN_FLIGHT');
	return typeof limit === 'number' ? pendingOnly.slice(0, limit) : pendingOnly;
}

async function markInFlight(syncIds: string[]): Promise<void> {
	if (syncIds.length === 0) return;
	const db = await openDB();
	const tx = db.transaction(STORE, 'readwrite');
	const store = tx.objectStore(STORE);
	for (const id of syncIds) {
		const row = await requestToPromise(store.get(id) as IDBRequest<SyncQueueRow | undefined>);
		if (!row) continue;
		store.put({ ...row, status: 'IN_FLIGHT' });
	}
	await transactionToPromise(tx);
}

async function markAcked(syncId: string): Promise<void> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readwrite');
	tx.objectStore(STORE).delete(syncId);
	await transactionToPromise(tx);
}

async function markFailed(syncId: string, error: string): Promise<void> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readwrite');
	const store = tx.objectStore(STORE);
	const row = await requestToPromise(store.get(syncId) as IDBRequest<SyncQueueRow | undefined>);
	if (!row) {
		await transactionToPromise(tx);
		return;
	}
	store.put({
		...row,
		status: 'FAILED' satisfies SyncOperationStatus,
		attempts: row.attempts + 1,
		last_error: error
	});
	await transactionToPromise(tx);
}

async function count(): Promise<number> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readonly');
	const n = await requestToPromise(tx.objectStore(STORE).count());
	await transactionToPromise(tx);
	return n;
}

async function clear(): Promise<void> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readwrite');
	tx.objectStore(STORE).clear();
	await transactionToPromise(tx);
}

export const syncQueueRepo = {
	enqueue,
	pending,
	markInFlight,
	markAcked,
	markFailed,
	count,
	clear
};
