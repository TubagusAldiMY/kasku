import { PUBLIC_API_BASE_URL } from '$env/static/public';
import { auth } from '$lib/stores/auth.svelte';

type FetchOptions = RequestInit & {
	skipAuth?: boolean;
};

/**
 * apiFetch adalah wrapper fetch yang secara otomatis menangani:
 * 1. Base URL
 * 2. Authorization Header (Bearer Token)
 * 3. Content-Type JSON
 * 4. Silent Refresh jika mendapat 401 Unauthorized
 */
export async function apiFetch(path: string, options: FetchOptions = {}) {
	const url = `${PUBLIC_API_BASE_URL}${path}`;
	const headers = new Headers(options.headers);

	// Sisipkan Bearer token jika tidak eksplisit di-skip
	if (!options.skipAuth && auth.accessToken) {
		headers.set('Authorization', `Bearer ${auth.accessToken}`);
	}

	// Default Content-Type ke JSON jika body bukan FormData
	if (!(options.body instanceof FormData) && !headers.has('Content-Type')) {
		headers.set('Content-Type', 'application/json');
	}

	let response = await fetch(url, {
		...options,
		headers
	});

	// Jika 401 dan bukan sedang mencoba refresh, coba silent refresh
	if (response.status === 401 && !options.skipAuth && path !== '/auth/refresh') {
		const refreshed = await silentRefresh();
		if (refreshed) {
			// Retry request asli dengan token baru
			headers.set('Authorization', `Bearer ${auth.accessToken}`);
			response = await fetch(url, { ...options, headers });
		}
	}

	return response;
}

/**
 * silentRefresh memanggil endpoint refresh token backend.
 * Mengandalkan HttpOnly cookie yang dikirim otomatis oleh browser.
 */
async function silentRefresh(): Promise<boolean> {
	try {
		const response = await fetch(`${PUBLIC_API_BASE_URL}/auth/refresh`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' }
		});

		if (response.ok) {
			const result = await response.json();
			if (result.success && result.data.access_token) {
				auth.setToken(result.data.access_token);
				return true;
			}
		}
	} catch (e) {
		console.error('[API] Silent refresh failed:', e);
	}

	// Jika refresh gagal, paksa logout (bersihkan state)
	auth.logout();
	return false;
}
