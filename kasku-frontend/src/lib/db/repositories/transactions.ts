import { createCrudRepository } from './crud';
import { openDB, requestToPromise, transactionToPromise } from '../connection';
import type { TransactionRow } from '../schema';

const base = createCrudRepository<TransactionRow>('transactions');

async function listByAccount(accountId: string): Promise<TransactionRow[]> {
	const db = await openDB();
	const tx = db.transaction('transactions', 'readonly');
	const idx = tx.objectStore('transactions').index('by_account_id');
	const rows = await requestToPromise(idx.getAll(accountId) as IDBRequest<TransactionRow[]>);
	await transactionToPromise(tx);
	return rows.filter((r) => !r._deleted);
}

async function listByDateRange(startIso: string, endIso: string): Promise<TransactionRow[]> {
	const db = await openDB();
	const tx = db.transaction('transactions', 'readonly');
	const idx = tx.objectStore('transactions').index('by_transaction_date');
	const range = IDBKeyRange.bound(startIso, endIso, false, false);
	const rows = await requestToPromise(idx.getAll(range) as IDBRequest<TransactionRow[]>);
	await transactionToPromise(tx);
	return rows.filter((r) => !r._deleted);
}

export const transactionsRepo = {
	...base,
	listByAccount,
	listByDateRange
};
