import type { SyncableEntity } from '$lib/db';

/**
 * Server Wins conflict resolution:
 * - Jika tidak ada salinan lokal → terima server (pull baru).
 * - Jika server.updated_at >= local.updated_at → terima server.
 * - Jika local.updated_at > server.updated_at → tetap lokal,
 *   tapi tandai conflict agar engine bisa retry push lokal.
 *
 * Tombstone (`_deleted=true`) dari server menghapus lokal apa pun.
 */

export type ConflictDecision = 'apply_server' | 'keep_local';

export type ConflictResolution<T extends SyncableEntity> = {
	decision: ConflictDecision;
	winner: T;
	loserHadLocalChanges: boolean;
};

export function applyServerWins<T extends SyncableEntity>(
	local: T | undefined,
	server: T
): ConflictResolution<T> {
	if (!local) {
		return { decision: 'apply_server', winner: server, loserHadLocalChanges: false };
	}

	if (server._deleted) {
		return {
			decision: 'apply_server',
			winner: server,
			loserHadLocalChanges: !!local._local_dirty
		};
	}

	const localT = Date.parse(local.updated_at);
	const serverT = Date.parse(server.updated_at);

	if (Number.isNaN(serverT)) {
		return { decision: 'keep_local', winner: local, loserHadLocalChanges: false };
	}
	if (Number.isNaN(localT)) {
		return { decision: 'apply_server', winner: server, loserHadLocalChanges: false };
	}

	if (serverT >= localT) {
		return {
			decision: 'apply_server',
			winner: server,
			loserHadLocalChanges: !!local._local_dirty
		};
	}

	return { decision: 'keep_local', winner: local, loserHadLocalChanges: false };
}
