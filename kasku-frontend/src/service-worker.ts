/// <reference types="@sveltejs/kit" />
/// <reference lib="webworker" />

// Tombstone service worker.
//
// KasKu web is now FULL ONLINE — the offline layer (IndexedDB + sync engine +
// the old caching service worker) was removed. This file exists ONLY to evict
// the previously-installed caching SW from users' browsers: when their browser
// update-checks `/service-worker.js` it receives this script, which unregisters
// itself and purges every cache. Without it the stale offline app (with the
// `/sync` route + 429 sync errors) would keep being served from cache forever,
// because a 404 on the SW script does NOT auto-unregister an installed worker.
//
// ponytail: delete this file once existing installs have cycled through it (a
// few weeks after deploy). New visitors never register a SW — nothing calls
// register() — so this only ever cleans up old installs.

const sw = self as unknown as ServiceWorkerGlobalScope;

sw.addEventListener('install', () => {
	sw.skipWaiting();
});

sw.addEventListener('activate', (event) => {
	event.waitUntil(
		(async () => {
			for (const key of await caches.keys()) {
				await caches.delete(key);
			}
			await sw.registration.unregister();
			// Reload any open tabs so they fetch the fresh, SW-less app.
			const clients = await sw.clients.matchAll({ type: 'window' });
			for (const client of clients) {
				(client as WindowClient).navigate((client as WindowClient).url);
			}
		})()
	);
});
