import {
	accountsRepo,
	transactionsRepo,
	investmentsRepo,
	syncQueueRepo,
	type AccountRow,
	type TransactionRow,
	type InvestmentRow,
	type SyncableEntity
} from '$lib/db';
import { syncStatus } from './store.svelte';
import { triggerManualSync } from './trigger';
import type { SyncableResource } from './types';

/**
 * Helper mutation untuk UI: tulis ke IDB sebagai source-of-truth lokal,
 * enqueue ke sync_queue agar engine push ke server, lalu trigger sync
 * di background.
 *
 * Pattern ini menggantikan `apiFetch('/accounts', { method: 'POST' })`
 * langsung di komponen. UI tidak perlu await network — optimistic update.
 */

function nowIso(): string {
	return new Date().toISOString();
}

function newSyncId(): string {
	return crypto.randomUUID();
}

function newEntityId(): string {
	return crypto.randomUUID();
}

async function fireAndForgetSync(): Promise<void> {
	if (typeof navigator !== 'undefined' && !navigator.onLine) return;
	void triggerManualSync();
}

type AnySyncableRow = AccountRow | TransactionRow | InvestmentRow;

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

async function putLocal(resource: SyncableResource, row: AnySyncableRow): Promise<void> {
	switch (resource) {
		case 'accounts':
			await accountsRepo.put(row as AccountRow);
			return;
		case 'transactions':
			await transactionsRepo.put(row as TransactionRow);
			return;
		case 'investments':
			await investmentsRepo.put(row as InvestmentRow);
			return;
	}
}

export async function enqueueCreate<T extends AnySyncableRow>(
	resource: SyncableResource,
	input: Omit<T, 'id' | 'sync_id' | 'updated_at' | '_local_dirty' | '_deleted'> &
		Partial<SyncableEntity>
): Promise<T> {
	const syncId = newSyncId();
	const id = input.id ?? newEntityId();
	const updated_at = nowIso();
	const row = {
		...input,
		id,
		sync_id: syncId,
		updated_at,
		_local_dirty: true,
		_deleted: false
	} as T;

	await putLocal(resource, row);
	await syncQueueRepo.enqueue({
		sync_id: syncId,
		resource,
		operation: 'CREATE',
		entity_id: id,
		payload: row,
		created_at: updated_at
	});

	syncStatus.bumpDataVersion();
	syncStatus.setQueuedCount(await syncQueueRepo.count());
	void fireAndForgetSync();
	return row;
}

export async function enqueueUpdate<T extends AnySyncableRow>(
	resource: SyncableResource,
	id: string,
	patch: Partial<T>
): Promise<T | undefined> {
	const repo = repoFor(resource);
	const existing = (await repo.getById(id)) as T | undefined;
	if (!existing) return undefined;

	const syncId = newSyncId();
	const updated_at = nowIso();
	const row = {
		...existing,
		...patch,
		id,
		sync_id: syncId,
		updated_at,
		_local_dirty: true,
		_deleted: false
	} as T;

	await putLocal(resource, row);
	await syncQueueRepo.enqueue({
		sync_id: syncId,
		resource,
		operation: 'UPDATE',
		entity_id: id,
		payload: row,
		created_at: updated_at
	});

	syncStatus.bumpDataVersion();
	syncStatus.setQueuedCount(await syncQueueRepo.count());
	void fireAndForgetSync();
	return row;
}

export async function enqueueDelete(resource: SyncableResource, id: string): Promise<void> {
	const repo = repoFor(resource);
	const existing = await repo.getById(id);
	if (!existing) return;

	const syncId = newSyncId();
	const updated_at = nowIso();
	await repo.softDelete(id, syncId, updated_at);

	await syncQueueRepo.enqueue({
		sync_id: syncId,
		resource,
		operation: 'DELETE',
		entity_id: id,
		payload: { id },
		created_at: updated_at
	});

	syncStatus.bumpDataVersion();
	syncStatus.setQueuedCount(await syncQueueRepo.count());
	void fireAndForgetSync();
}
