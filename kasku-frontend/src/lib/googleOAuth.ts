/**
 * Mulai redirect-based Google OAuth (authorization code flow).
 * Dipakai di halaman login & register.
 *
 * State CSRF disimpan di sessionStorage, diverifikasi di /google/callback.
 * Client secret tetap di backend — frontend hanya mengirim authorization code.
 */
export function startGoogleLogin(clientId: string): void {
	if (!clientId) return;

	const state = crypto.randomUUID();
	sessionStorage.setItem('google_oauth_state', state);

	const params = new URLSearchParams({
		client_id: clientId,
		redirect_uri: `${window.location.origin}/google/callback`,
		response_type: 'code',
		scope: 'openid email profile',
		state,
		prompt: 'select_account'
	});

	window.location.href = `https://accounts.google.com/o/oauth2/v2/auth?${params}`;
}
