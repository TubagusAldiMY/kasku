/**
 * Kontrak HTTP sync-service (mirror dari kasku-backend/sync-service/src/domain/entity.rs).
 *
 * Sumber kebenaran tunggal ada di Rust struct. Setiap perubahan di server
 * harus disinkron manual di sini. Pertimbangkan codegen dari OpenAPI/proto
 * di milestone berikutnya jika gap makin sering muncul.
 */

import type { StoreName } from '$lib/db';

export type ServerEntityType = 'financial_account' | 'transaction' | 'investment_asset';
export type ServerOperation = 'create' | 'update' | 'delete';

/** Resource lokal (object store IDB) yang ikut dalam protokol sync. */
export type SyncableResource = Extract<StoreName, 'accounts' | 'transactions' | 'investments'>;

export const RESOURCE_TO_SERVER_ENTITY: Record<SyncableResource, ServerEntityType> = {
	accounts: 'financial_account',
	transactions: 'transaction',
	investments: 'investment_asset'
};

export const SERVER_ENTITY_TO_RESOURCE: Record<ServerEntityType, SyncableResource> = {
	financial_account: 'accounts',
	transaction: 'transactions',
	investment_asset: 'investments'
};

export const SYNCABLE_RESOURCES: SyncableResource[] = ['accounts', 'transactions', 'investments'];

export type SyncOperationPayload = {
	sync_id: string;
	entity_type: ServerEntityType;
	entity_id: string;
	operation: ServerOperation;
	payload: unknown;
	client_timestamp: string;
};

export type PushRequest = {
	operations: SyncOperationPayload[];
};

export type SyncResultStatus = 'applied' | 'skipped' | 'conflict' | 'error';

export type SyncResult = {
	sync_id: string;
	entity_type: ServerEntityType;
	entity_id: string;
	status: SyncResultStatus;
	server_data?: unknown;
};

export type PushResponseData = {
	processed: number;
	conflicts: number;
	skipped: number;
	results: SyncResult[];
	server_timestamp: string;
};

export type EntityChange = {
	entity_type: ServerEntityType;
	entity_id: string;
	operation: ServerOperation;
	data: Record<string, unknown>;
	updated_at: string;
};

export type PullResponseData = {
	changes: EntityChange[];
	server_timestamp: string;
};

export type ApiEnvelope<T> = {
	success: boolean;
	data?: T;
	error?: { code: string; message: string };
};
