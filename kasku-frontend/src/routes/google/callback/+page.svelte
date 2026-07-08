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
			errorMsg =
				oauthError === 'access_denied'
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

<div class="flex min-h-screen items-center justify-center bg-paper px-6">
	<div class="w-full max-w-[380px] space-y-8 text-center">
		<span class="font-serif text-[26px] leading-none tracking-tight text-ink"
			>Kas<em class="text-teal">Ku</em></span
		>

		{#if status === 'loading'}
			<div class="flex flex-col items-center gap-5">
				<span class="h-10 w-10 animate-spin rounded-full border-2 border-teal/25 border-t-teal"
				></span>
				<div class="space-y-1.5">
					<p class="text-sm font-semibold text-ink">Memproses login Google…</p>
					<p class="text-[13px] text-ink/50">Mohon tunggu sebentar</p>
				</div>
			</div>
		{:else}
			<div class="flex flex-col items-center gap-6">
				<div
					class="flex h-16 w-16 items-center justify-center rounded-full border border-clay/25 bg-clay/5"
				>
					<svg
						class="h-8 w-8 text-clay"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="1.8"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
				</div>
				<div class="space-y-1.5">
					<p class="font-serif text-[22px] leading-tight tracking-tight text-ink">Login gagal</p>
					<p class="text-[13px] text-ink/55">{errorMsg}</p>
				</div>
				<a
					href={resolve('/login')}
					class="flex items-center justify-center gap-2.5 rounded-full bg-teal px-6 py-3 text-sm font-semibold text-card transition-colors hover:bg-ink"
				>
					Kembali ke masuk
				</a>
			</div>
		{/if}
	</div>
</div>
