import { syncAll } from './engine';
import { syncStatus } from './store.svelte';
import { syncQueueRepo } from '$lib/db';

const VISIBILITY_THROTTLE_MS = 30_000;
const PERIODIC_INTERVAL_MS = 5 * 60_000;

let lastSyncAttemptAt = 0;
let periodicHandle: ReturnType<typeof setInterval> | null = null;
let listenersAttached = false;

async function safeSync(): Promise<void> {
	if (typeof navigator !== 'undefined' && !navigator.onLine) return;
	lastSyncAttemptAt = Date.now();
	try {
		await syncAll();
	} catch {
		// Error sudah disimpan ke syncStatus oleh engine; jangan rethrow di trigger.
	}
}

function handleOnline() {
	syncStatus.setOnline(true);
	void safeSync();
}

function handleOffline() {
	syncStatus.setOnline(false);
}

function handleVisibility() {
	if (typeof document === 'undefined') return;
	if (document.visibilityState !== 'visible') return;
	if (typeof navigator !== 'undefined' && !navigator.onLine) return;
	if (Date.now() - lastSyncAttemptAt < VISIBILITY_THROTTLE_MS) return;
	void safeSync();
}

/**
 * Attach event listeners + start periodic sync. Idempotent — safe to call
 * multiple times (mis. dari root layout onMount).
 */
export function initSyncTriggers(): void {
	if (listenersAttached) return;
	if (typeof window === 'undefined') return;

	window.addEventListener('online', handleOnline);
	window.addEventListener('offline', handleOffline);
	document.addEventListener('visibilitychange', handleVisibility);
	periodicHandle = setInterval(() => void safeSync(), PERIODIC_INTERVAL_MS);
	listenersAttached = true;

	void refreshQueueCount();
}

export function teardownSyncTriggers(): void {
	if (!listenersAttached) return;
	if (typeof window === 'undefined') return;

	window.removeEventListener('online', handleOnline);
	window.removeEventListener('offline', handleOffline);
	document.removeEventListener('visibilitychange', handleVisibility);
	if (periodicHandle) clearInterval(periodicHandle);
	periodicHandle = null;
	listenersAttached = false;
}

/** Manual button — bypass throttle. */
export async function triggerManualSync(): Promise<void> {
	await safeSync();
	await refreshQueueCount();
}

async function refreshQueueCount(): Promise<void> {
	try {
		syncStatus.setQueuedCount(await syncQueueRepo.count());
	} catch {
		// IDB belum siap di SSR atau ada error transient — diamkan.
	}
}
