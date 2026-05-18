import { browser } from '$app/environment';

/**
 * Admin auth store terpisah dari user `auth` karena admin-service
 * memakai JWT HS256 track yang berbeda dari user RS256.
 */

export type AdminUser = {
	id: string;
	username: string;
	role: string;
	is_active: boolean;
	last_login_at?: string | null;
	created_at: string;
};

type AdminAuthState = {
	accessToken: string | null;
	expiresAt: number | null;
	admin: AdminUser | null;
	loading: boolean;
};

const STORAGE_KEY = 'kasku_admin_auth';

type Persisted = {
	accessToken: string | null;
	expiresAt: number | null;
	admin: AdminUser | null;
};

function readPersisted(): Persisted | null {
	if (!browser) return null;
	try {
		const raw = sessionStorage.getItem(STORAGE_KEY);
		if (!raw) return null;
		return JSON.parse(raw) as Persisted;
	} catch {
		return null;
	}
}

function writePersisted(p: Persisted) {
	if (!browser) return;
	try {
		if (!p.accessToken && !p.admin) {
			sessionStorage.removeItem(STORAGE_KEY);
			return;
		}
		sessionStorage.setItem(STORAGE_KEY, JSON.stringify(p));
	} catch {
		// no-op
	}
}

function createAdminAuthStore() {
	const hydrated = readPersisted();
	const state = $state<AdminAuthState>({
		accessToken: hydrated?.accessToken ?? null,
		expiresAt: hydrated?.expiresAt ?? null,
		admin: hydrated?.admin ?? null,
		loading: false
	});

	function persist() {
		writePersisted({
			accessToken: state.accessToken,
			expiresAt: state.expiresAt,
			admin: state.admin
		});
	}

	return {
		get accessToken() {
			return state.accessToken;
		},
		get expiresAt() {
			return state.expiresAt;
		},
		get admin() {
			return state.admin;
		},
		get isAuthenticated() {
			if (!state.accessToken) return false;
			if (state.expiresAt && state.expiresAt < Date.now()) return false;
			return true;
		},
		get loading() {
			return state.loading;
		},

		setSession(token: string, expiresInSec: number, admin: AdminUser) {
			state.accessToken = token;
			state.expiresAt = Date.now() + expiresInSec * 1000;
			state.admin = admin;
			state.loading = false;
			persist();
		},
		setAdmin(admin: AdminUser | null) {
			state.admin = admin;
			persist();
		},
		setLoading(v: boolean) {
			state.loading = v;
		},
		logout() {
			state.accessToken = null;
			state.expiresAt = null;
			state.admin = null;
			state.loading = false;
			persist();
		}
	};
}

export const adminAuth = createAdminAuthStore();
