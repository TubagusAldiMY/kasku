import { defineEnvVars } from '@sveltejs/kit/env';

/**
 * Deklarasi environment variable (SvelteKit 3 menggantikan $env/static|dynamic/public).
 * Nama variabel sengaja dipertahankan supaya .env, Dockerfile, dan setelan Vercel tidak berubah.
 */
export const variables = defineEnvVars({
	// URL api-gateway (termasuk /v1). Di-inline saat build: wajib ada di build Docker/Vercel.
	PUBLIC_API_BASE_URL: {
		public: true,
		static: true,
		description: 'URL api-gateway termasuk suffix /v1'
	},
	// Google OAuth Client ID. Dibaca saat runtime; kosong/tidak diset = tombol Google dinonaktifkan.
	PUBLIC_GOOGLE_CLIENT_ID: {
		public: true,
		schema: (value) => value ?? '',
		description: 'Google OAuth Client ID, kosong untuk menonaktifkan login Google'
	}
});
