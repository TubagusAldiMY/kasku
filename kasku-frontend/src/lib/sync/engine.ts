import { apiFetch } from '$lib/api/client';
import {
	accountsRepo,
	transactionsRepo,
	investmentsRepo,
	syncQueueRepo,
	syncMetaRepo,
	type SyncableEntity,
	type SyncQueueRow,
	type AccountRow,
	type TransactionRow,
	type InvestmentRow
} from '$lib/db';
import { applyServerWins } from './conflict';
import {
	RESOURCE_TO_SERVER_ENTITY,
	SERVER_ENTITY_TO_RESOURCE,
	SYNCABLE_RESOURCES,
	type ApiEnvelope,
	type EntityChange,
	type PullResponseData,
	type PushRequest,
	type PushResponseData,
	type SyncableResource,
	type SyncOperationPayload
} from './types';
import { syncStatus } from './store.svelte';

const PUSH_BATCH_SIZE = 50;
/** Default lookback bila belum pernah pull (sentinel epoch zero). */
const PULL_DEFAULT_SINCE = '1970-01-01T00:00:00.000Z';

type FetchLike = typeof apiFetch;

export type EngineDeps = {
	fetcher?: FetchLike;
	now?: () => string;
};

function repoFor(resource: SyncableResource) {
	switch (resource) {
		case 'accounts':
			return accountsRepo;
		case 'transactions':
			return transactionsRepo;
		case 'investments':
			return investmentsRepo;
	}
}

function toEntity(change: EntityChange): SyncableEntity {
	const data = (change.data ?? {}) as Record<string, unknown>;
	const id = (data.id as string | undefined) ?? change.entity_id;
	return {
		...data,
		id,
		updated_at: change.updated_at,
		_deleted:
			change.operation === 'delete' ? true : ((data._deleted as boolean | undefined) ?? false),
		_local_dirty: false
	} as SyncableEntity;
}

async function pullDelta(
	resource: SyncableResource,
	fetcher: FetchLike,
	now: () => string
): Promise<{ applied: number; conflicts: number }> {
	const since = (await syncMetaRepo.get(resource)) ?? PULL_DEFAULT_SINCE;
	const url = `/sync/pull?since=${encodeURIComponent(since)}`;
	const response = await fetcher(url);

	if (!response.ok) {
		throw new Error(`pull ${resource} HTTP ${response.status}`);
	}

	const envelope = (await response.json()) as ApiEnvelope<PullResponseData>;
	if (!envelope.success || !envelope.data) {
		throw new Error(envelope.error?.message ?? `pull ${resource} response invalid`);
	}

	let applied = 0;
	let conflicts = 0;

	const repo = repoFor(resource);
	const targetEntity = RESOURCE_TO_SERVER_ENTITY[resource];

	for (const change of envelope.data.changes) {
		if (change.entity_type !== targetEntity) continue;

		const incoming = toEntity(change);
		const existing = await repo.getById(incoming.id);
		const resolution = applyServerWins(existing as SyncableEntity | undefined, incoming);

		if (resolution.decision === 'apply_server') {
			if (incoming._deleted) {
				await repo.hardDelete(incoming.id);
			} else {
				await typedPut(resource, incoming);
			}
			applied += 1;
			if (resolution.loserHadLocalChanges) conflicts += 1;
		}
	}

	await syncMetaRepo.set(resource, envelope.data.server_timestamp || now());
	return { applied, conflicts };
}

async function typedPut(resource: SyncableResource, item: SyncableEntity): Promise<void> {
	switch (resource) {
		case 'accounts':
			await accountsRepo.put(item as AccountRow);
			return;
		case 'transactions':
			await transactionsRepo.put(item as TransactionRow);
			return;
		case 'investments':
			await investmentsRepo.put(item as InvestmentRow);
			return;
	}
}

async function pushPending(
	fetcher: FetchLike,
	now: () => string
): Promise<{ processed: number; conflicts: number; skipped: number }> {
	const pending = await syncQueueRepo.pending(PUSH_BATCH_SIZE);
	if (pending.length === 0) return { processed: 0, conflicts: 0, skipped: 0 };

	const operations: SyncOperationPayload[] = pending.map((row) => queueRowToOperation(row, now()));
	const syncIds = pending.map((r) => r.sync_id);

	await syncQueueRepo.markInFlight(syncIds);

	const body: PushRequest = { operations };
	let response: Response;
	try {
		response = await fetcher('/sync/push', {
			method: 'POST',
			body: JSON.stringify(body)
		});
	} catch (err) {
		const msg = err instanceof Error ? err.message : 'network error';
		for (const id of syncIds) await syncQueueRepo.markFailed(id, msg);
		throw err;
	}

	if (!response.ok) {
		const msg = `push HTTP ${response.status}`;
		for (const id of syncIds) await syncQueueRepo.markFailed(id, msg);
		throw new Error(msg);
	}

	const envelope = (await response.json()) as ApiEnvelope<PushResponseData>;
	if (!envelope.success || !envelope.data) {
		const msg = envelope.error?.message ?? 'push response invalid';
		for (const id of syncIds) await syncQueueRepo.markFailed(id, msg);
		throw new Error(msg);
	}

	for (const result of envelope.data.results) {
		if (result.status === 'applied' || result.status === 'skipped') {
			await syncQueueRepo.markAcked(result.sync_id);
		} else if (result.status === 'conflict') {
			await syncQueueRepo.markAcked(result.sync_id);
			if (result.server_data) {
				const resource = SERVER_ENTITY_TO_RESOURCE[result.entity_type];
				const entity = toEntity({
					entity_type: result.entity_type,
					entity_id: result.entity_id,
					operation: 'update',
					data: result.server_data as Record<string, unknown>,
					updated_at: envelope.data.server_timestamp
				});
				await typedPut(resource, entity);
			}
		} else {
			await syncQueueRepo.markFailed(result.sync_id, 'server reported error status');
		}
	}

	return {
		processed: envelope.data.processed,
		conflicts: envelope.data.conflicts,
		skipped: envelope.data.skipped
	};
}

function queueRowToOperation(row: SyncQueueRow, nowIso: string): SyncOperationPayload {
	const entityType = RESOURCE_TO_SERVER_ENTITY[row.resource as SyncableResource];
	return {
		sync_id: row.sync_id,
		entity_type: entityType,
		entity_id: row.entity_id,
		operation: row.operation.toLowerCase() as 'create' | 'update' | 'delete',
		payload: row.payload,
		client_timestamp: row.created_at || nowIso
	};
}

export async function syncAll(deps: EngineDeps = {}): Promise<void> {
	const fetcher = deps.fetcher ?? apiFetch;
	const now = deps.now ?? (() => new Date().toISOString());

	if (syncStatus.running) return;
	syncStatus.setRunning(true);
	syncStatus.setError(null);

	try {
		// 1. Pull dulu untuk dapat baseline server terbaru (Server Wins).
		for (const resource of SYNCABLE_RESOURCES) {
			await pullDelta(resource, fetcher, now);
		}

		// 2. Push lokal yang belum di-ack.
		await pushPending(fetcher, now);

		// 3. Pull lagi untuk dapat hasil baru setelah push (mis. server set field tambahan).
		for (const resource of SYNCABLE_RESOURCES) {
			await pullDelta(resource, fetcher, now);
		}

		syncStatus.setQueuedCount(await syncQueueRepo.count());
		syncStatus.setLastSyncAt(now());
	} catch (err) {
		const msg = err instanceof Error ? err.message : String(err);
		syncStatus.setError(msg);
		throw err;
	} finally {
		syncStatus.setRunning(false);
	}
}

export const _internals = {
	pullDelta,
	pushPending,
	queueRowToOperation,
	toEntity,
	PUSH_BATCH_SIZE,
	PULL_DEFAULT_SINCE
};
