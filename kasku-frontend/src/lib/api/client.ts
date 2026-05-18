import { PUBLIC_API_BASE_URL } from '$env/static/public';
import { auth } from '$lib/stores/auth.svelte';

type FetchOptions = RequestInit & {
	skipAuth?: boolean;
};

let refreshPromise: Promise<boolean> | null = null;

/**
 * apiFetch adalah wrapper fetch yang secara otomatis menangani:
 * 1. Base URL
 * 2. Authorization Header (Bearer Token)
 * 3. Content-Type JSON
 * 4. Silent Refresh jika mendapat 401 Unauthorized
 */
export async function apiFetch(path: string, options: FetchOptions = {}) {
	const { skipAuth = false, ...requestOptions } = options;
	const url = `${PUBLIC_API_BASE_URL}${path}`;
	const headers = new Headers(requestOptions.headers);

	// Sisipkan Bearer token jika tidak eksplisit di-skip
	if (!skipAuth && auth.accessToken) {
		headers.set('Authorization', `Bearer ${auth.accessToken}`);
	}

	// Default Content-Type ke JSON jika body bukan FormData
	if (!(requestOptions.body instanceof FormData) && !headers.has('Content-Type')) {
		headers.set('Content-Type', 'application/json');
	}

	let response = await fetch(url, {
		...requestOptions,
		headers,
		credentials: requestOptions.credentials ?? 'include'
	});

	// Jika 401 dan bukan sedang mencoba refresh, coba silent refresh
	if (response.status === 401 && !skipAuth && path !== '/auth/refresh') {
		const refreshed = await silentRefresh();
		if (refreshed) {
			// Retry request asli dengan token baru
			headers.set('Authorization', `Bearer ${auth.accessToken}`);
			response = await fetch(url, {
				...requestOptions,
				headers,
				credentials: requestOptions.credentials ?? 'include'
			});
		}
	}

	return response;
}

/**
 * silentRefresh memanggil endpoint refresh token backend.
 * Mengandalkan HttpOnly cookie yang dikirim otomatis oleh browser.
 *
 * Hanya force-logout saat refresh endpoint EKSPLISIT tolak (401/403) —
 * artinya refresh token benar-benar invalid. Untuk network error, 404
 * (endpoint belum aktif), atau 5xx, biarkan token yang ada agar user
 * tidak kena logout palsu karena masalah konektivitas/backend transient.
 */
async function silentRefresh(): Promise<boolean> {
	if (refreshPromise) return refreshPromise;

	refreshPromise = doSilentRefresh().finally(() => {
		refreshPromise = null;
	});

	return refreshPromise;
}

async function doSilentRefresh(): Promise<boolean> {
	try {
		const response = await fetch(`${PUBLIC_API_BASE_URL}/auth/refresh`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			credentials: 'include'
		});

		if (response.ok) {
			const result = await response.json();
			if (result.success && result.data?.access_token) {
				auth.setToken(result.data.access_token);
				return true;
			}
		}

		if (response.status === 401 || response.status === 403) {
			// Refresh token dinyatakan invalid oleh server — sesi memang habis.
			auth.logout();
		}
	} catch (e) {
		// Network error / offline — jangan logout, biarkan user retry.
		console.error('[API] Silent refresh failed:', e);
	}

	return false;
}
