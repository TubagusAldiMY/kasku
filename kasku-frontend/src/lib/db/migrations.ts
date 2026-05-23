import { STORES, STORE_NAMES } from './schema';

/**
 * Migration runner untuk IndexedDB.
 *
 * Setiap entry di `MIGRATIONS` adalah function yang dipanggil pada
 * `onupgradeneeded` ketika `oldVersion < key`. Function ini menerima
 * `IDBDatabase` (untuk create/delete object store) dan `IDBTransaction`
 * (untuk create indexes pada store baru).
 *
 * Strict rule: jangan lakukan I/O lain di sini. Hanya perubahan struktural.
 * Jika butuh transform data, lakukan di seed/migration code yang berjalan
 * setelah open berhasil — dalam transaction terpisah.
 */
export type MigrationFn = (db: IDBDatabase, tx: IDBTransaction) => void;

export function createMissingStores(db: IDBDatabase): void {
	for (const name of STORE_NAMES) {
		if (db.objectStoreNames.contains(name)) continue;
		const spec = STORES[name];
		const store = db.createObjectStore(name, { keyPath: spec.keyPath });
		for (const idx of spec.indexes) {
			store.createIndex(idx.name, idx.keyPath, idx.options);
		}
	}
}

export const MIGRATIONS: Record<number, MigrationFn> = {
	1: (db) => {
		createMissingStores(db);
	},
	// v3: buat store yang belum ada (menangkap user di v1/v2 yang belum punya 'budgets').
	3: (db) => {
		createMissingStores(db);
	},
	// v4: recovery untuk browser yang sudah terlanjur punya DB v3 tanpa store baru.
	4: (db) => {
		createMissingStores(db);
	}
};

export function runMigrations(db: IDBDatabase, tx: IDBTransaction, oldVersion: number) {
	const targetVersions = Object.keys(MIGRATIONS)
		.map(Number)
		.sort((a, b) => a - b);

	for (const version of targetVersions) {
		if (oldVersion < version) {
			MIGRATIONS[version](db, tx);
		}
	}
}
