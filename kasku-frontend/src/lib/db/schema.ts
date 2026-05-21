/**
 * IndexedDB schema untuk KasKu offline-first storage.
 *
 * Konvensi entity:
 * - `id`: UUID stabil (dari server, atau client-generated `pending-<uuid>` untuk create offline yang belum di-sync).
 * - `sync_id`: UUID unik per mutasi untuk idempotency di server.
 * - `updated_at`: ISO 8601 string — server menang via `applyServerWins` di sync engine.
 * - `_local_dirty`: true jika belum di-ack server.
 * - `_deleted`: tombstone untuk delete operation yang menunggu sync.
 */

export const DB_NAME = 'kasku';
export const DB_VERSION = 2;

export type StoreName =
	| 'accounts'
	| 'transactions'
	| 'categories'
	| 'investments'
	| 'budgets'
	| 'sync_queue'
	| 'sync_meta';

export type SyncableEntity = {
	id: string;
	sync_id?: string;
	updated_at: string;
	_local_dirty?: boolean;
	_deleted?: boolean;
};

export type AccountRow = SyncableEntity & {
	name: string;
	account_type: string;
	balance: number;
	currency: string;
	color?: string;
};

export type TransactionRow = SyncableEntity & {
	account_id: string;
	category_id: string;
	transaction_type: 'INCOME' | 'EXPENSE' | 'TRANSFER';
	amount_idr: number;
	transaction_date: string;
	notes?: string;
	to_account_id?: string;
};

export type CategoryRow = SyncableEntity & {
	name: string;
	category_type: 'INCOME' | 'EXPENSE' | 'BOTH';
	color?: string;
};

export type InvestmentRow = SyncableEntity & {
	name: string;
	asset_type: 'CRYPTO' | 'GOLD' | 'STOCK' | 'MUTUAL_FUND';
	symbol?: string;
	units: number;
	avg_buy_price_idr: number;
};

export type BudgetRow = {
	id: string;
	name: string;
	limit_idr: number;
	category_id?: string;
	category_name?: string;
	period_type: 'MONTHLY' | 'WEEKLY' | 'CUSTOM';
	start_date?: string;
	end_date?: string | null;
	alert_threshold: number;
	spent_idr: number;
	remaining_idr: number;
	progress_percent: number;
	is_over_budget: boolean;
	updated_at: string;
};

export type SyncOperationType = 'CREATE' | 'UPDATE' | 'DELETE';
export type SyncOperationStatus = 'PENDING' | 'IN_FLIGHT' | 'FAILED';

export type SyncQueueRow = {
	sync_id: string;
	resource: Exclude<StoreName, 'sync_queue' | 'sync_meta'>;
	operation: SyncOperationType;
	entity_id: string;
	payload: unknown;
	status: SyncOperationStatus;
	attempts: number;
	last_error?: string;
	created_at: string;
};

export type SyncMetaRow = {
	resource: string;
	last_synced_at: string;
};

type StoreSpec = {
	keyPath: string;
	indexes: { name: string; keyPath: string; options?: IDBIndexParameters }[];
};

export const STORES: Record<StoreName, StoreSpec> = {
	accounts: {
		keyPath: 'id',
		indexes: [{ name: 'by_updated_at', keyPath: 'updated_at' }]
	},
	transactions: {
		keyPath: 'id',
		indexes: [
			{ name: 'by_account_id', keyPath: 'account_id' },
			{ name: 'by_transaction_date', keyPath: 'transaction_date' },
			{ name: 'by_updated_at', keyPath: 'updated_at' }
		]
	},
	categories: {
		keyPath: 'id',
		indexes: [{ name: 'by_updated_at', keyPath: 'updated_at' }]
	},
	investments: {
		keyPath: 'id',
		indexes: [{ name: 'by_updated_at', keyPath: 'updated_at' }]
	},
	budgets: {
		keyPath: 'id',
		indexes: [{ name: 'by_updated_at', keyPath: 'updated_at' }]
	},
	sync_queue: {
		keyPath: 'sync_id',
		indexes: [
			{ name: 'by_created_at', keyPath: 'created_at' },
			{ name: 'by_status', keyPath: 'status' }
		]
	},
	sync_meta: {
		keyPath: 'resource',
		indexes: []
	}
};

export const STORE_NAMES: StoreName[] = Object.keys(STORES) as StoreName[];
