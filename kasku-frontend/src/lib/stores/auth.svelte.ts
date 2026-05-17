import { browser } from '$app/environment';

interface AuthUser {
	id: string;
	email: string;
	username: string;
}

interface AuthState {
	accessToken: string | null;
	user: AuthUser | null;
	isAuthenticated: boolean;
	loading: boolean;
}

const STORAGE_KEY = 'kasku_auth';

type Persisted = {
	accessToken: string | null;
	user: AuthUser | null;
};

function readPersisted(): Persisted | null {
	if (!browser) return null;
	try {
		const raw = sessionStorage.getItem(STORAGE_KEY);
		if (!raw) return null;
		const parsed = JSON.parse(raw) as Persisted;
		if (typeof parsed?.accessToken !== 'string' && parsed?.accessToken !== null) return null;
		return parsed;
	} catch {
		return null;
	}
}

function writePersisted(token: string | null, user: AuthUser | null) {
	if (!browser) return;
	try {
		if (!token && !user) {
			sessionStorage.removeItem(STORAGE_KEY);
			return;
		}
		sessionStorage.setItem(STORAGE_KEY, JSON.stringify({ accessToken: token, user }));
	} catch {
		// sessionStorage may be unavailable (private mode quota, etc.) — fail open
	}
}

const createAuthStore = () => {
	const hydrated = readPersisted();
	const initialToken = hydrated?.accessToken ?? null;
	const initialUser = hydrated?.user ?? null;

	const state = $state<AuthState>({
		accessToken: initialToken,
		user: initialUser,
		isAuthenticated: !!initialToken,
		loading: !initialToken
	});

	return {
		get accessToken() {
			return state.accessToken;
		},
		get user() {
			return state.user;
		},
		get isAuthenticated() {
			return state.isAuthenticated;
		},
		get loading() {
			return state.loading;
		},

		setToken: (token: string | null) => {
			state.accessToken = token;
			state.isAuthenticated = !!token;
			state.loading = false;
			writePersisted(token, state.user);
		},
		setUser: (user: AuthUser | null) => {
			state.user = user;
			writePersisted(state.accessToken, user);
		},
		setLoading: (loading: boolean) => {
			state.loading = loading;
		},
		logout: () => {
			state.accessToken = null;
			state.user = null;
			state.isAuthenticated = false;
			state.loading = false;
			writePersisted(null, null);
		}
	};
};

export const auth = createAuthStore();
