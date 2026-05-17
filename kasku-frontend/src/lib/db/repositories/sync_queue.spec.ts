import 'fake-indexeddb/auto';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { _resetConnection } from '../connection';
import { DB_NAME } from '../schema';
import { syncQueueRepo } from './sync_queue';
import type { SyncQueueRow } from '../schema';

function makeEntry(
	createdAt: string,
	overrides: Partial<SyncQueueRow> = {}
): Omit<SyncQueueRow, 'status' | 'attempts'> {
	return {
		sync_id: overrides.sync_id ?? crypto.randomUUID(),
		resource: overrides.resource ?? 'accounts',
		operation: overrides.operation ?? 'CREATE',
		entity_id: overrides.entity_id ?? crypto.randomUUID(),
		payload: overrides.payload ?? { foo: 'bar' },
		created_at: createdAt
	};
}

beforeEach(async () => {
	await syncQueueRepo.clear().catch(() => undefined);
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

describe('sync_queue repository', () => {
	it('enqueues with status PENDING and attempts 0', async () => {
		const entry = makeEntry('2026-05-17T00:00:00.000Z');
		await syncQueueRepo.enqueue(entry);
		const pending = await syncQueueRepo.pending();
		expect(pending).toHaveLength(1);
		expect(pending[0].status).toBe('PENDING');
		expect(pending[0].attempts).toBe(0);
	});

	it('returns pending in FIFO order via by_created_at', async () => {
		await syncQueueRepo.enqueue(makeEntry('2026-05-17T01:00:00.000Z', { entity_id: 'B' }));
		await syncQueueRepo.enqueue(makeEntry('2026-05-17T00:00:00.000Z', { entity_id: 'A' }));
		await syncQueueRepo.enqueue(makeEntry('2026-05-17T02:00:00.000Z', { entity_id: 'C' }));

		const order = (await syncQueueRepo.pending()).map((r) => r.entity_id);
		expect(order).toEqual(['A', 'B', 'C']);
	});

	it('markAcked removes entry from queue', async () => {
		const entry = makeEntry('2026-05-17T00:00:00.000Z');
		await syncQueueRepo.enqueue(entry);
		await syncQueueRepo.markAcked(entry.sync_id);
		expect(await syncQueueRepo.count()).toBe(0);
	});

	it('markFailed increments attempts and stores last_error', async () => {
		const entry = makeEntry('2026-05-17T00:00:00.000Z');
		await syncQueueRepo.enqueue(entry);
		await syncQueueRepo.markFailed(entry.sync_id, 'network error');
		await syncQueueRepo.markFailed(entry.sync_id, 'timeout');

		const pending = await syncQueueRepo.pending();
		expect(pending[0].attempts).toBe(2);
		expect(pending[0].last_error).toBe('timeout');
		expect(pending[0].status).toBe('FAILED');
	});

	it('markInFlight hides entries from pending() default behaviour', async () => {
		const a = makeEntry('2026-05-17T00:00:00.000Z', { entity_id: 'A' });
		const b = makeEntry('2026-05-17T01:00:00.000Z', { entity_id: 'B' });
		await syncQueueRepo.enqueue(a);
		await syncQueueRepo.enqueue(b);

		await syncQueueRepo.markInFlight([a.sync_id]);
		const visible = await syncQueueRepo.pending();
		expect(visible.map((r) => r.entity_id)).toEqual(['B']);
	});

	it('respects pending() limit', async () => {
		for (let i = 0; i < 5; i++) {
			await syncQueueRepo.enqueue(makeEntry(`2026-05-17T0${i}:00:00.000Z`, { entity_id: `E${i}` }));
		}
		const limited = await syncQueueRepo.pending(3);
		expect(limited).toHaveLength(3);
	});
});
