interface AuthState {
	accessToken: string | null;
	user: {
		id: string;
		email: string;
		username: string;
	} | null;
	isAuthenticated: boolean;
	loading: boolean;
}

const createAuthStore = () => {
	let state = $state<AuthState>({
		accessToken: null,
		user: null,
		isAuthenticated: false,
		loading: true
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
		},
		setUser: (user: AuthState['user']) => {
			state.user = user;
		},
		setLoading: (loading: boolean) => {
			state.loading = loading;
		},
		logout: () => {
			state.accessToken = null;
			state.user = null;
			state.isAuthenticated = false;
			state.loading = false;
		}
	};
};

export const auth = createAuthStore();
