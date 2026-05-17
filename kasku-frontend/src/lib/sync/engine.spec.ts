import 'fake-indexeddb/auto';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { _resetConnection } from '$lib/db/connection';
import { DB_NAME, accountsRepo, syncQueueRepo, syncMetaRepo } from '$lib/db';
import type { AccountRow, SyncQueueRow } from '$lib/db';
import { _internals, syncAll } from './engine';
import type {
	ApiEnvelope,
	PullResponseData,
	PushResponseData,
	SyncOperationPayload
} from './types';

const { pullDelta, pushPending } = _internals;

function envelope<T>(data: T): Response {
	return new Response(JSON.stringify({ success: true, data } satisfies ApiEnvelope<T>), {
		status: 200,
		headers: { 'Content-Type': 'application/json' }
	});
}

function makeAccount(overrides: Partial<AccountRow> = {}): AccountRow {
	return {
		id: overrides.id ?? crypto.randomUUID(),
		name: overrides.name ?? 'BCA Utama',
		account_type: 'BANK',
		balance: 0,
		currency: 'IDR',
		updated_at: overrides.updated_at ?? '2026-05-17T00:00:00.000Z',
		...overrides
	};
}

beforeEach(async () => {
	await accountsRepo.clear().catch(() => undefined);
	await syncQueueRepo.clear().catch(() => undefined);
	await syncMetaRepo.clear().catch(() => undefined);
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

describe('engine.pullDelta', () => {
	it('applies new entities and updates sync_meta', async () => {
		const fetcher = vi.fn().mockResolvedValue(
			envelope<PullResponseData>({
				changes: [
					{
						entity_type: 'financial_account',
						entity_id: 'srv-1',
						operation: 'create',
						data: {
							id: 'srv-1',
							name: 'Mandiri',
							account_type: 'BANK',
							balance: 5_000_000,
							currency: 'IDR',
							updated_at: '2026-05-17T10:00:00.000Z'
						},
						updated_at: '2026-05-17T10:00:00.000Z'
					}
				],
				server_timestamp: '2026-05-17T10:00:00.000Z'
			})
		);

		const result = await pullDelta('accounts', fetcher, () => '2026-05-17T10:00:00.000Z');
		expect(result.applied).toBe(1);

		const all = await accountsRepo.getAll();
		expect(all).toHaveLength(1);
		expect(all[0].name).toBe('Mandiri');

		expect(await syncMetaRepo.get('accounts')).toBe('2026-05-17T10:00:00.000Z');
	});

	it('uses Server Wins to overwrite older local', async () => {
		await accountsRepo.put(
			makeAccount({ id: 'a-1', name: 'old', updated_at: '2026-05-17T08:00:00.000Z' })
		);

		const fetcher = vi.fn().mockResolvedValue(
			envelope<PullResponseData>({
				changes: [
					{
						entity_type: 'financial_account',
						entity_id: 'a-1',
						operation: 'update',
						data: {
							id: 'a-1',
							name: 'new',
							account_type: 'BANK',
							balance: 0,
							currency: 'IDR',
							updated_at: '2026-05-17T12:00:00.000Z'
						},
						updated_at: '2026-05-17T12:00:00.000Z'
					}
				],
				server_timestamp: '2026-05-17T12:00:00.000Z'
			})
		);

		await pullDelta('accounts', fetcher, () => '2026-05-17T12:00:00.000Z');
		const fetched = await accountsRepo.getById('a-1');
		expect(fetched?.name).toBe('new');
	});

	it('keeps local when local is newer than server', async () => {
		await accountsRepo.put(
			makeAccount({ id: 'a-1', name: 'newer-local', updated_at: '2026-05-17T15:00:00.000Z' })
		);

		const fetcher = vi.fn().mockResolvedValue(
			envelope<PullResponseData>({
				changes: [
					{
						entity_type: 'financial_account',
						entity_id: 'a-1',
						operation: 'update',
						data: {
							id: 'a-1',
							name: 'older-server',
							account_type: 'BANK',
							balance: 0,
							currency: 'IDR',
							updated_at: '2026-05-17T10:00:00.000Z'
						},
						updated_at: '2026-05-17T10:00:00.000Z'
					}
				],
				server_timestamp: '2026-05-17T16:00:00.000Z'
			})
		);

		await pullDelta('accounts', fetcher, () => '2026-05-17T16:00:00.000Z');
		const fetched = await accountsRepo.getById('a-1');
		expect(fetched?.name).toBe('newer-local');
	});

	it('hard-deletes locally when server sends delete operation', async () => {
		await accountsRepo.put(makeAccount({ id: 'gone' }));
		const fetcher = vi.fn().mockResolvedValue(
			envelope<PullResponseData>({
				changes: [
					{
						entity_type: 'financial_account',
						entity_id: 'gone',
						operation: 'delete',
						data: { id: 'gone' },
						updated_at: '2026-05-17T10:00:00.000Z'
					}
				],
				server_timestamp: '2026-05-17T10:00:00.000Z'
			})
		);

		await pullDelta('accounts', fetcher, () => '2026-05-17T10:00:00.000Z');
		const remaining = await accountsRepo.getAll();
		expect(remaining).toHaveLength(0);
	});

	it('ignores changes for other entity types in the same response', async () => {
		const fetcher = vi.fn().mockResolvedValue(
			envelope<PullResponseData>({
				changes: [
					{
						entity_type: 'transaction',
						entity_id: 'tx-1',
						operation: 'create',
						data: { id: 'tx-1' },
						updated_at: '2026-05-17T10:00:00.000Z'
					}
				],
				server_timestamp: '2026-05-17T10:00:00.000Z'
			})
		);

		const result = await pullDelta('accounts', fetcher, () => '2026-05-17T10:00:00.000Z');
		expect(result.applied).toBe(0);
		expect(await accountsRepo.getAll()).toHaveLength(0);
	});
});

describe('engine.pushPending', () => {
	async function seedQueue(): Promise<SyncQueueRow> {
		const entry = {
			sync_id: 'sync-1',
			resource: 'accounts' as const,
			operation: 'CREATE' as const,
			entity_id: 'acc-new',
			payload: { id: 'acc-new', name: 'Created' },
			created_at: '2026-05-17T00:00:00.000Z'
		};
		await syncQueueRepo.enqueue(entry);
		return { ...entry, status: 'PENDING', attempts: 0 };
	}

	it('sends batch and acks applied results', async () => {
		await seedQueue();

		const fetcher = vi.fn().mockResolvedValue(
			envelope<PushResponseData>({
				processed: 1,
				conflicts: 0,
				skipped: 0,
				results: [
					{
						sync_id: 'sync-1',
						entity_type: 'financial_account',
						entity_id: 'acc-new',
						status: 'applied'
					}
				],
				server_timestamp: '2026-05-17T10:00:00.000Z'
			})
		);

		const result = await pushPending(fetcher, () => '2026-05-17T10:00:00.000Z');
		expect(result.processed).toBe(1);
		expect(await syncQueueRepo.count()).toBe(0);

		const call = fetcher.mock.calls[0];
		expect(call[0]).toBe('/sync/push');
		const body = JSON.parse((call[1] as RequestInit).body as string) as {
			operations: SyncOperationPayload[];
		};
		expect(body.operations).toHaveLength(1);
		expect(body.operations[0].entity_type).toBe('financial_account');
		expect(body.operations[0].operation).toBe('create');
	});

	it('marks failed when server returns non-OK', async () => {
		await seedQueue();

		const fetcher = vi.fn().mockResolvedValue(new Response('boom', { status: 500 }));

		await expect(pushPending(fetcher, () => '2026-05-17T10:00:00.000Z')).rejects.toThrow();
		const pending = await syncQueueRepo.pending();
		expect(pending).toHaveLength(1);
		expect(pending[0].status).toBe('FAILED');
		expect(pending[0].attempts).toBe(1);
	});

	it('applies server_data to IDB on conflict result', async () => {
		await seedQueue();
		await accountsRepo.put(
			makeAccount({ id: 'acc-new', name: 'local-stale', updated_at: '2026-05-17T08:00:00.000Z' })
		);

		const fetcher = vi.fn().mockResolvedValue(
			envelope<PushResponseData>({
				processed: 0,
				conflicts: 1,
				skipped: 0,
				results: [
					{
						sync_id: 'sync-1',
						entity_type: 'financial_account',
						entity_id: 'acc-new',
						status: 'conflict',
						server_data: {
							id: 'acc-new',
							name: 'server-truth',
							account_type: 'BANK',
							balance: 9_000_000,
							currency: 'IDR',
							updated_at: '2026-05-17T11:00:00.000Z'
						}
					}
				],
				server_timestamp: '2026-05-17T11:00:00.000Z'
			})
		);

		await pushPending(fetcher, () => '2026-05-17T11:00:00.000Z');
		expect(await syncQueueRepo.count()).toBe(0);
		const fetched = await accountsRepo.getById('acc-new');
		expect(fetched?.name).toBe('server-truth');
	});

	it('returns early when queue is empty', async () => {
		const fetcher = vi.fn();
		const result = await pushPending(fetcher, () => '2026-05-17T10:00:00.000Z');
		expect(result).toEqual({ processed: 0, conflicts: 0, skipped: 0 });
		expect(fetcher).not.toHaveBeenCalled();
	});
});

describe('engine.syncAll', () => {
	it('runs pull -> push -> pull and updates syncStatus', async () => {
		// Fresh Response per call — body stream cannot be read twice.
		const fetcher = vi.fn(() =>
			Promise.resolve(
				envelope<PullResponseData>({
					changes: [],
					server_timestamp: '2026-05-17T10:00:00.000Z'
				})
			)
		);

		await syncAll({ fetcher, now: () => '2026-05-17T10:00:00.000Z' });

		// 3 resources × 2 pull passes = 6 calls minimum.
		expect(fetcher.mock.calls.length).toBeGreaterThanOrEqual(6);
		// All calls were to pull (queue empty).
		const urls = fetcher.mock.calls.map((c) => String((c as unknown[])[0]));
		for (const url of urls) {
			expect(url).toMatch(/\/sync\/pull/);
		}
	});
});
