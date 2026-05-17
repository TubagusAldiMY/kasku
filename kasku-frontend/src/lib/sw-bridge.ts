import { browser, dev } from '$app/environment';
import { triggerManualSync } from '$lib/sync';

let registration: ServiceWorkerRegistration | null = null;
let updateListenerAttached = false;
let onUpdateAvailable: (() => void) | null = null;

/**
 * Register service worker (idempotent). Tidak melakukan apa-apa di SSR
 * atau saat dev mode (Vite tidak menyediakan SW bundling di dev).
 */
export async function registerServiceWorker(opts?: {
	onUpdateAvailable?: () => void;
}): Promise<ServiceWorkerRegistration | null> {
	if (!browser) return null;
	if (dev) return null;
	if (!('serviceWorker' in navigator)) return null;

	onUpdateAvailable = opts?.onUpdateAvailable ?? null;

	try {
		registration = await navigator.serviceWorker.register('/service-worker.js', { type: 'module' });
	} catch {
		registration = null;
		return null;
	}

	if (registration.waiting) onUpdateAvailable?.();

	registration.addEventListener('updatefound', () => {
		const installing = registration?.installing;
		if (!installing) return;
		installing.addEventListener('statechange', () => {
			if (installing.state === 'installed' && navigator.serviceWorker.controller) {
				onUpdateAvailable?.();
			}
		});
	});

	if (!updateListenerAttached) {
		navigator.serviceWorker.addEventListener('message', (event) => {
			if (event.data?.type === 'TRIGGER_SYNC') {
				void triggerManualSync();
			}
		});
		updateListenerAttached = true;
	}

	return registration;
}

/**
 * Daftarkan Background Sync agar OS bangunkan SW saat connectivity kembali.
 * No-op jika API tidak didukung (Safari/Firefox).
 */
export async function requestBackgroundSync(tag = 'kasku-sync'): Promise<void> {
	if (!browser) return;
	if (!('serviceWorker' in navigator)) return;

	const reg = registration ?? (await navigator.serviceWorker.ready.catch(() => null));
	const syncManager = (
		reg as ServiceWorkerRegistration & { sync?: { register: (t: string) => Promise<void> } }
	)?.sync;
	if (!syncManager) return;

	try {
		await syncManager.register(tag);
	} catch {
		// Background Sync gagal — diamkan, manual trigger tetap bekerja.
	}
}

/**
 * Perintahkan SW yang sedang menunggu untuk segera aktif.
 */
export function skipWaiting(): void {
	registration?.waiting?.postMessage({ type: 'SKIP_WAITING' });
}
