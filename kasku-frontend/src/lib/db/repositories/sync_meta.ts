import { openDB, requestToPromise, transactionToPromise } from '../connection';
import type { SyncMetaRow } from '../schema';

const STORE = 'sync_meta';

async function get(resource: string): Promise<string | undefined> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readonly');
	const row = await requestToPromise(
		tx.objectStore(STORE).get(resource) as IDBRequest<SyncMetaRow | undefined>
	);
	await transactionToPromise(tx);
	return row?.last_synced_at;
}

async function set(resource: string, lastSyncedAt: string): Promise<void> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readwrite');
	tx.objectStore(STORE).put({ resource, last_synced_at: lastSyncedAt } satisfies SyncMetaRow);
	await transactionToPromise(tx);
}

async function getAll(): Promise<SyncMetaRow[]> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readonly');
	const rows = await requestToPromise(tx.objectStore(STORE).getAll() as IDBRequest<SyncMetaRow[]>);
	await transactionToPromise(tx);
	return rows;
}

async function clear(): Promise<void> {
	const db = await openDB();
	const tx = db.transaction(STORE, 'readwrite');
	tx.objectStore(STORE).clear();
	await transactionToPromise(tx);
}

export const syncMetaRepo = {
	get,
	set,
	getAll,
	clear
};
