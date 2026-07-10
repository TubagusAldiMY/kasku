import { openDB, requestToPromise, transactionToPromise } from '../connection';
import type { SyncConflictRow } from '../schema';

const STORE = 'sync_conflicts';

async function record(row: SyncConflictRow): Promise<void> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readwrite');
	tx.objectStore(STORE).put(row);
	await transactionToPromise(tx);
}

/** Terbaru dulu. */
async function list(): Promise<SyncConflictRow[]> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readonly');
	const rows = await requestToPromise(
		tx.objectStore(STORE).getAll() as IDBRequest<SyncConflictRow[]>
	);
	await transactionToPromise(tx);
	return rows.sort((a, b) => b.detected_at.localeCompare(a.detected_at));
}

/** Tandai selesai — hapus dari daftar. */
async function resolve(id: string): Promise<void> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readwrite');
	tx.objectStore(STORE).delete(id);
	await transactionToPromise(tx);
}

async function count(): Promise<number> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readonly');
	const n = await requestToPromise(tx.objectStore(STORE).count());
	await transactionToPromise(tx);
	return n;
}

export const conflictsRepo = { record, list, resolve, count };
