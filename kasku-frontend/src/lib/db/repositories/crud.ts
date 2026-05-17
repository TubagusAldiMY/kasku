import { openDB, requestToPromise, transactionToPromise } from '../connection';
import type { StoreName, SyncableEntity } from '../schema';

/**
 * Factory CRUD repository untuk store yang menyimpan SyncableEntity.
 *
 * Setiap method open transaction-nya sendiri agar caller bisa
 * compose call tanpa blocking. Jika butuh batch atomik, pakai
 * `runInTx` dari module ini.
 */
export function createCrudRepository<T extends SyncableEntity>(storeName: StoreName) {
	async function getAll(): Promise<T[]> {
		const db = await openDB();
		const tx = db.transaction(storeName, 'readonly');
		const store = tx.objectStore(storeName);
		const result = await requestToPromise(store.getAll() as IDBRequest<T[]>);
		await transactionToPromise(tx);
		return result.filter((item) => !item._deleted);
	}

	async function getById(id: string): Promise<T | undefined> {
		const db = await openDB();
		const tx = db.transaction(storeName, 'readonly');
		const store = tx.objectStore(storeName);
		const result = await requestToPromise(store.get(id) as IDBRequest<T | undefined>);
		await transactionToPromise(tx);
		if (result?._deleted) return undefined;
		return result;
	}

	async function put(item: T): Promise<void> {
		const db = await openDB();
		const tx = db.transaction(storeName, 'readwrite');
		tx.objectStore(storeName).put(item);
		await transactionToPromise(tx);
	}

	async function putMany(items: T[]): Promise<void> {
		if (items.length === 0) return;
		const db = await openDB();
		const tx = db.transaction(storeName, 'readwrite');
		const store = tx.objectStore(storeName);
		for (const item of items) {
			store.put(item);
		}
		await transactionToPromise(tx);
	}

	async function softDelete(id: string, syncId: string, nowIso: string): Promise<void> {
		const db = await openDB();
		const tx = db.transaction(storeName, 'readwrite');
		const store = tx.objectStore(storeName);
		const existing = (await requestToPromise(store.get(id) as IDBRequest<T | undefined>)) ?? null;
		if (!existing) {
			await transactionToPromise(tx);
			return;
		}
		const tombstone: T = {
			...existing,
			sync_id: syncId,
			updated_at: nowIso,
			_local_dirty: true,
			_deleted: true
		};
		store.put(tombstone);
		await transactionToPromise(tx);
	}

	async function hardDelete(id: string): Promise<void> {
		const db = await openDB();
		const tx = db.transaction(storeName, 'readwrite');
		tx.objectStore(storeName).delete(id);
		await transactionToPromise(tx);
	}

	async function getModifiedSince(sinceIso: string): Promise<T[]> {
		const db = await openDB();
		const tx = db.transaction(storeName, 'readonly');
		const store = tx.objectStore(storeName);
		const idx = store.index('by_updated_at');
		const range = IDBKeyRange.lowerBound(sinceIso, true);
		const result = await requestToPromise(idx.getAll(range) as IDBRequest<T[]>);
		await transactionToPromise(tx);
		return result;
	}

	async function clear(): Promise<void> {
		const db = await openDB();
		const tx = db.transaction(storeName, 'readwrite');
		tx.objectStore(storeName).clear();
		await transactionToPromise(tx);
	}

	return {
		getAll,
		getById,
		put,
		putMany,
		softDelete,
		hardDelete,
		getModifiedSince,
		clear
	};
}
