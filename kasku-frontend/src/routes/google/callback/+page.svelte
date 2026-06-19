<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { apiFetch } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';

	let status = $state<'loading' | 'error'>('loading');
	let errorMsg = $state('');

	onMount(async () => {
		const params = new URLSearchParams(window.location.search);
		const code = params.get('code');
		const state = params.get('state');
		const oauthError = params.get('error');

		if (oauthError) {
			status = 'error';
			errorMsg = oauthError === 'access_denied'
				? 'Akses ditolak. Anda membatalkan login Google.'
				: `Google error: ${oauthError}`;
			return;
		}

		if (!code) {
			status = 'error';
			errorMsg = 'Kode otorisasi tidak ditemukan dalam URL.';
			return;
		}

		const storedState = sessionStorage.getItem('google_oauth_state');
		if (!state || state !== storedState) {
			status = 'error';
			errorMsg = 'State tidak valid. Kemungkinan CSRF attack — coba login ulang.';
			return;
		}
		sessionStorage.removeItem('google_oauth_state');

		try {
			const response = await apiFetch('/auth/google/code', {
				method: 'POST',
				body: JSON.stringify({
					code,
					redirect_uri: `${window.location.origin}/google/callback`
				}),
				skipAuth: true
			});

			const result = await response.json();

			if (result.success) {
				auth.setToken(result.data.access_token);
				auth.setUser({ id: '', email: '', username: '' });
				localStorage.removeItem('kasku_mock_mode');
				goto(resolve('/dashboard'));
			} else {
				status = 'error';
				errorMsg = result.error?.message || 'Login Google gagal.';
			}
		} catch {
			status = 'error';
			errorMsg = 'Gagal terhubung ke server. Periksa koneksi Anda lalu coba lagi.';
		}
	});
</script>

<div class="flex min-h-screen items-center justify-center bg-gradient-to-br from-[#f0fafa] to-[#e6f7f8]">
	<div class="w-full max-w-sm rounded-3xl bg-white p-10 shadow-xl shadow-teal-900/5 text-center space-y-6">
		{#if status === 'loading'}
			<div class="flex flex-col items-center gap-4">
				<div class="h-12 w-12 animate-spin rounded-full border-4 border-teal-100 border-t-[#217b84]"></div>
				<div class="space-y-1">
					<p class="text-[15px] font-bold text-[#0a2e31]">Memproses login Google...</p>
					<p class="text-[13px] text-gray-400">Mohon tunggu sebentar</p>
				</div>
			</div>
		{:else}
			<div class="flex flex-col items-center gap-4">
				<div class="flex h-12 w-12 items-center justify-center rounded-full bg-red-50">
					<svg class="h-6 w-6 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
							d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
				</div>
				<div class="space-y-1">
					<p class="text-[15px] font-bold text-[#0a2e31]">Login Gagal</p>
					<p class="text-[13px] text-gray-500">{errorMsg}</p>
				</div>
				<a
					href={resolve('/login')}
					class="rounded-2xl bg-[#217b84] px-6 py-2.5 text-[13px] font-bold text-white transition-colors hover:bg-[#1a5f66]"
				>
					Kembali ke Login
				</a>
			</div>
		{/if}
	</div>
</div>
