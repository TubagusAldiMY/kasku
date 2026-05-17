/// <reference types="@sveltejs/kit" />
/// <reference no-default-lib="true" />
/// <reference lib="esnext" />
/// <reference lib="webworker" />

import { build, files, version } from '$service-worker';

const sw = self as unknown as ServiceWorkerGlobalScope;

const CACHE_PREFIX = 'kasku-shell';
const CACHE_NAME = `${CACHE_PREFIX}-${version}`;
const OFFLINE_URL = '/offline';

const PRECACHE_URLS = [
	...build, // hashed app bundles
	...files, // static/ assets
	OFFLINE_URL
];

sw.addEventListener('install', (event) => {
	event.waitUntil(
		(async () => {
			const cache = await caches.open(CACHE_NAME);
			await cache.addAll(PRECACHE_URLS);
			await sw.skipWaiting();
		})()
	);
});

sw.addEventListener('activate', (event) => {
	event.waitUntil(
		(async () => {
			const names = await caches.keys();
			await Promise.all(
				names
					.filter((n) => n.startsWith(CACHE_PREFIX) && n !== CACHE_NAME)
					.map((n) => caches.delete(n))
			);
			await sw.clients.claim();
		})()
	);
});

sw.addEventListener('fetch', (event) => {
	const request = event.request;
	if (request.method !== 'GET') return;

	const url = new URL(request.url);
	// Cross-origin (mis. api-gateway) → biarkan engine handle.
	if (url.origin !== sw.location.origin) return;

	// Navigation: network-first → fallback offline page.
	if (request.mode === 'navigate') {
		event.respondWith(navigateWithOfflineFallback(request));
		return;
	}

	// Hashed immutable assets → cache-first.
	if (url.pathname.startsWith('/_app/immutable/')) {
		event.respondWith(cacheFirst(request));
		return;
	}

	// Static files (icons, manifest, dll) → stale-while-revalidate.
	if (files.includes(url.pathname)) {
		event.respondWith(staleWhileRevalidate(request));
		return;
	}
});

sw.addEventListener('sync', (event) => {
	if (event.tag === 'kasku-sync') {
		event.waitUntil(notifyClientsToSync());
	}
});

sw.addEventListener('message', (event) => {
	if (event.data?.type === 'SKIP_WAITING') {
		sw.skipWaiting();
	}
});

async function navigateWithOfflineFallback(request: Request): Promise<Response> {
	try {
		const networkResponse = await fetch(request);
		if (networkResponse.ok || networkResponse.status === 304) return networkResponse;
		return (await caches.match(OFFLINE_URL)) ?? networkResponse;
	} catch {
		const fallback = await caches.match(OFFLINE_URL);
		if (fallback) return fallback;
		return new Response('Offline', { status: 503, statusText: 'Offline' });
	}
}

async function cacheFirst(request: Request): Promise<Response> {
	const cached = await caches.match(request);
	if (cached) return cached;
	const response = await fetch(request);
	if (response.ok) {
		const cache = await caches.open(CACHE_NAME);
		cache.put(request, response.clone()).catch(() => undefined);
	}
	return response;
}

async function staleWhileRevalidate(request: Request): Promise<Response> {
	const cache = await caches.open(CACHE_NAME);
	const cached = await cache.match(request);
	const fetchPromise = fetch(request)
		.then((response) => {
			if (response.ok) cache.put(request, response.clone()).catch(() => undefined);
			return response;
		})
		.catch(() => cached ?? Response.error());
	return cached ?? fetchPromise;
}

async function notifyClientsToSync(): Promise<void> {
	const clients = await sw.clients.matchAll({ type: 'window', includeUncontrolled: true });
	for (const client of clients) {
		client.postMessage({ type: 'TRIGGER_SYNC' });
	}
}
