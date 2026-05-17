/**
 * Reactive sync status untuk dipantau UI (badge, banner offline, dll).
 *
 * Bukan tempat menyimpan data sync — hanya snapshot ringan.
 */

type SyncStatus = {
	running: boolean;
	lastSyncAt: string | null;
	queuedCount: number;
	error: string | null;
	online: boolean;
	dataVersion: number;
};

const initialOnline = typeof navigator !== 'undefined' ? navigator.onLine : true;

const state = $state<SyncStatus>({
	running: false,
	lastSyncAt: null,
	queuedCount: 0,
	error: null,
	online: initialOnline,
	dataVersion: 0
});

export const syncStatus = {
	get running() {
		return state.running;
	},
	get lastSyncAt() {
		return state.lastSyncAt;
	},
	get queuedCount() {
		return state.queuedCount;
	},
	get error() {
		return state.error;
	},
	get online() {
		return state.online;
	},
	get dataVersion() {
		return state.dataVersion;
	},

	setRunning(v: boolean) {
		state.running = v;
	},
	setLastSyncAt(v: string | null) {
		state.lastSyncAt = v;
	},
	setQueuedCount(v: number) {
		state.queuedCount = v;
	},
	setError(v: string | null) {
		state.error = v;
	},
	setOnline(v: boolean) {
		state.online = v;
	},
	bumpDataVersion() {
		state.dataVersion += 1;
	}
};
