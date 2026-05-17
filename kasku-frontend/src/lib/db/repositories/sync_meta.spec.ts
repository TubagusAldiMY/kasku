import 'fake-indexeddb/auto';
import { afterEach, describe, expect, it } from 'vitest';
import { _resetConnection } from '../connection';
import { DB_NAME } from '../schema';
import { syncMetaRepo } from './sync_meta';

afterEach(async () => {
	_resetConnection();
	await new Promise<void>((resolve, reject) => {
		const req = indexedDB.deleteDatabase(DB_NAME);
		req.onsuccess = () => resolve();
		req.onerror = () => reject(req.error);
		req.onblocked = () => reject(new Error('deleteDatabase blocked'));
	});
});

describe('sync_meta repository', () => {
	it('returns undefined for missing resource', async () => {
		const v = await syncMetaRepo.get('accounts');
		expect(v).toBeUndefined();
	});

	it('stores and retrieves last_synced_at per resource', async () => {
		await syncMetaRepo.set('accounts', '2026-05-17T10:00:00.000Z');
		await syncMetaRepo.set('transactions', '2026-05-17T11:00:00.000Z');

		expect(await syncMetaRepo.get('accounts')).toBe('2026-05-17T10:00:00.000Z');
		expect(await syncMetaRepo.get('transactions')).toBe('2026-05-17T11:00:00.000Z');
	});

	it('overwrites existing entry on repeated set', async () => {
		await syncMetaRepo.set('accounts', '2026-05-17T10:00:00.000Z');
		await syncMetaRepo.set('accounts', '2026-05-17T12:00:00.000Z');
		expect(await syncMetaRepo.get('accounts')).toBe('2026-05-17T12:00:00.000Z');
	});

	it('getAll returns every meta row', async () => {
		await syncMetaRepo.set('accounts', '2026-05-17T10:00:00.000Z');
		await syncMetaRepo.set('transactions', '2026-05-17T11:00:00.000Z');
		const all = await syncMetaRepo.getAll();
		expect(all).toHaveLength(2);
	});
});
