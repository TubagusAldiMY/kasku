import 'fake-indexeddb/auto';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { _resetConnection } from '../connection';
import { DB_NAME } from '../schema';
import { accountsRepo } from './accounts';
import type { AccountRow } from '../schema';

function makeAccount(overrides: Partial<AccountRow> = {}): AccountRow {
	return {
		id: overrides.id ?? crypto.randomUUID(),
		sync_id: overrides.sync_id ?? crypto.randomUUID(),
		name: overrides.name ?? 'BCA Utama',
		account_type: overrides.account_type ?? 'BANK',
		balance: overrides.balance ?? 1_000_000,
		currency: overrides.currency ?? 'IDR',
		updated_at: overrides.updated_at ?? new Date().toISOString(),
		...overrides
	};
}

beforeEach(async () => {
	await accountsRepo.clear().catch(() => undefined);
});

afterEach(async () => {
	_resetConnection();
	await new Promise<void>((resolve, reject) => {
		const req = indexedDB.deleteDatabase(DB_NAME);
		req.onsuccess = () => resolve();
		req.onerror = () => reject(req.error);
		req.onblocked = () => reject(new Error('deleteDatabase blocked'));
	});
});

describe('crud repository (accounts)', () => {
	it('round-trips put -> getById', async () => {
		const acc = makeAccount({ name: 'Mandiri' });
		await accountsRepo.put(acc);
		const fetched = await accountsRepo.getById(acc.id);
		expect(fetched?.name).toBe('Mandiri');
	});

	it('returns all non-deleted via getAll', async () => {
		await accountsRepo.putMany([makeAccount({ name: 'A' }), makeAccount({ name: 'B' })]);
		const all = await accountsRepo.getAll();
		expect(all).toHaveLength(2);
	});

	it('hides soft-deleted entries from getAll and getById', async () => {
		const acc = makeAccount();
		await accountsRepo.put(acc);
		await accountsRepo.softDelete(acc.id, crypto.randomUUID(), new Date().toISOString());

		const all = await accountsRepo.getAll();
		expect(all).toHaveLength(0);
		const byId = await accountsRepo.getById(acc.id);
		expect(byId).toBeUndefined();
	});

	it('keeps tombstone reachable via getModifiedSince for sync push', async () => {
		const old = makeAccount({ updated_at: '2020-01-01T00:00:00.000Z' });
		await accountsRepo.put(old);
		await accountsRepo.softDelete(old.id, crypto.randomUUID(), '2026-05-17T00:00:00.000Z');

		const modified = await accountsRepo.getModifiedSince('2026-01-01T00:00:00.000Z');
		expect(modified).toHaveLength(1);
		expect(modified[0]._deleted).toBe(true);
	});

	it('filters by_updated_at with exclusive lower bound', async () => {
		await accountsRepo.put(makeAccount({ name: 'old', updated_at: '2020-01-01T00:00:00.000Z' }));
		await accountsRepo.put(makeAccount({ name: 'mid', updated_at: '2026-05-10T00:00:00.000Z' }));
		await accountsRepo.put(makeAccount({ name: 'new', updated_at: '2026-05-17T00:00:00.000Z' }));

		const after = await accountsRepo.getModifiedSince('2026-05-10T00:00:00.000Z');
		const names = after.map((a) => a.name).sort();
		expect(names).toEqual(['new']);
	});

	it('hardDelete removes the row entirely', async () => {
		const acc = makeAccount();
		await accountsRepo.put(acc);
		await accountsRepo.hardDelete(acc.id);
		const byId = await accountsRepo.getById(acc.id);
		expect(byId).toBeUndefined();
	});
});
