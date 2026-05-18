import { PUBLIC_API_BASE_URL } from '$env/static/public';
import { adminAuth } from '$lib/stores/admin_auth.svelte';

type FetchOptions = RequestInit & {
	skipAuth?: boolean;
};

/**
 * apiFetch khusus admin-service. Admin pakai JWT HS256 yang independen
 * dari user RS256, jadi token dan storage-nya juga terpisah dari `auth`.
 *
 * Path harus dimulai dengan `/admin/...` — base URL sudah include `/v1`.
 * Tidak ada silent refresh — admin token TTL panjang dan re-login dipakai
 * saat expired (HTTP 401 → kick ke /admin/login).
 */
export async function adminApiFetch(path: string, options: FetchOptions = {}): Promise<Response> {
	const url = `${PUBLIC_API_BASE_URL}${path}`;
	const headers = new Headers(options.headers);

	if (!options.skipAuth && adminAuth.accessToken) {
		headers.set('Authorization', `Bearer ${adminAuth.accessToken}`);
	}

	if (!(options.body instanceof FormData) && !headers.has('Content-Type')) {
		headers.set('Content-Type', 'application/json');
	}

	return fetch(url, { ...options, headers });
}
