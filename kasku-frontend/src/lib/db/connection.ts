import { DB_NAME, DB_VERSION } from './schema';
import { runMigrations } from './migrations';

let dbPromise: Promise<IDBDatabase> | null = null;

/**
 * Buka koneksi IndexedDB sebagai singleton lazy.
 *
 * Jangan re-export instance langsung — call `openDB()` agar lazy
 * dan SSR-safe (di server `indexedDB` tidak ada).
 */
export function openDB(): Promise<IDBDatabase> {
	if (typeof indexedDB === 'undefined') {
		return Promise.reject(new Error('IndexedDB tidak tersedia di environment ini'));
	}

	if (dbPromise) return dbPromise;

	dbPromise = new Promise<IDBDatabase>((resolve, reject) => {
		const request = indexedDB.open(DB_NAME, DB_VERSION);

		request.onupgradeneeded = (event) => {
			const db = request.result;
			const tx = request.transaction;
			if (!tx) {
				reject(new Error('Upgrade transaction tidak tersedia'));
				return;
			}
			runMigrations(db, tx, event.oldVersion);
		};

		request.onsuccess = () => {
			const db = request.result;
			db.onversionchange = () => {
				// Tab lain men-trigger upgrade — tutup koneksi agar tidak block.
				db.close();
				dbPromise = null;
			};
			resolve(db);
		};

		request.onerror = () => {
			dbPromise = null;
			reject(request.error ?? new Error('IndexedDB open gagal'));
		};

		request.onblocked = () => {
			reject(new Error('IndexedDB open ter-block (tab lain belum tutup koneksi versi lama)'));
		};
	});

	return dbPromise;
}

/**
 * Tutup koneksi (testing helper). Production code tidak perlu panggil ini.
 */
export function _resetConnection(): void {
	if (dbPromise) {
		dbPromise.then((db) => db.close()).catch(() => undefined);
		dbPromise = null;
	}
}

/**
 * Wrapper Promise untuk `IDBRequest`.
 */
export function requestToPromise<T>(req: IDBRequest<T>): Promise<T> {
	return new Promise<T>((resolve, reject) => {
		req.onsuccess = () => resolve(req.result);
		req.onerror = () => reject(req.error ?? new Error('IDBRequest gagal'));
	});
}

/**
 * Wrapper Promise untuk transaction completion. Resolve saat `complete`,
 * reject saat `error` atau `abort`.
 */
export function transactionToPromise(tx: IDBTransaction): Promise<void> {
	return new Promise<void>((resolve, reject) => {
		tx.oncomplete = () => resolve();
		tx.onerror = () => reject(tx.error ?? new Error('Transaction error'));
		tx.onabort = () => reject(tx.error ?? new Error('Transaction abort'));
	});
}
