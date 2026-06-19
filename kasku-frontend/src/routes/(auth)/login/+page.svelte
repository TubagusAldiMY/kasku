<script lang="ts">
	import { apiFetch } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { env } from '$env/dynamic/public';

	const googleClientId = env.PUBLIC_GOOGLE_CLIENT_ID ?? '';

	let email = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state<string | null>(null);
	let isBackendDown = $state(false);
	let needsVerification = $state(false);
	let resendLoading = $state(false);
	let resendMessage = $state<string | null>(null);
	let showPassword = $state(false);

	async function handleLogin(e: SubmitEvent) {
		e.preventDefault();
		loading = true;
		error = null;
		isBackendDown = false;
		needsVerification = false;
		resendMessage = null;

		try {
			const response = await apiFetch('/auth/login', {
				method: 'POST',
				body: JSON.stringify({ email, password }),
				skipAuth: true
			});

			const result = await response.json();

			if (result.success) {
				auth.setToken(result.data.access_token);
				auth.setUser({ id: '1', email, username: email.split('@')[0] });
				localStorage.removeItem('kasku_mock_mode');
				goto(resolve('/dashboard'));
			} else {
				const msg = result.error?.message || 'Email atau password salah.';
				error = msg;
				if (msg.toLowerCase().includes('verifikasi') || msg.toLowerCase().includes('verify')) {
					needsVerification = true;
				}
			}
		} catch (err) {
			isBackendDown = true;
			error = 'Backend belum aktif. Gunakan Mode Demo untuk mencoba UI.';
			console.error(err);
		} finally {
			loading = false;
		}
	}

	async function handleResendVerification() {
		resendLoading = true;
		resendMessage = null;
		try {
			const response = await apiFetch('/auth/resend-verification', {
				method: 'POST',
				body: JSON.stringify({ email }),
				skipAuth: true
			});
			const result = await response.json();
			if (result.success) {
				resendMessage = 'Email verifikasi telah dikirim ulang!';
			} else {
				resendMessage = result.error?.message || 'Gagal mengirim ulang email verifikasi.';
			}
		} catch {
			resendMessage = 'Terjadi kesalahan koneksi.';
		} finally {
			resendLoading = false;
		}
	}

	function handleMockLogin() {
		auth.setToken('mock-jwt-token');
		auth.setUser({
			id: 'mock-user-id',
			email: 'demo@kasku.id',
			username: 'Juragan Demo'
		});
		localStorage.setItem('kasku_mock_mode', 'true');
		goto(resolve('/dashboard'));
	}

	// Redirect-based OAuth flow — bekerja di semua browser tanpa popup / GIS library.
	// Client secret tetap aman di backend; frontend hanya mengirim authorization code.
	function handleGoogleLogin() {
		if (!googleClientId) return;

		const state = crypto.randomUUID();
		sessionStorage.setItem('google_oauth_state', state);

		const params = new URLSearchParams({
			client_id: googleClientId,
			redirect_uri: `${window.location.origin}/google/callback`,
			response_type: 'code',
			scope: 'openid email profile',
			state,
			prompt: 'select_account'
		});

		window.location.href = `https://accounts.google.com/o/oauth2/v2/auth?${params}`;
	}
</script>

<div class="animate-in fade-in slide-in-from-bottom-4 space-y-10 duration-700">
	<div class="space-y-3 text-center">
		<h2 class="text-3xl font-bold tracking-tight text-[#0a2e31]">Selamat Datang</h2>
		<p class="mx-auto max-w-[320px] text-[15px] leading-relaxed text-gray-500">
			Kelola dan pantau pertumbuhan aset Anda dalam satu platform terpadu.
		</p>
	</div>

	<!-- Tab Switcher -->
	<div class="flex justify-center border-b border-gray-100">
		<div class="relative px-6 py-3 text-sm font-bold text-[#0a2e31] transition-colors">
			Masuk Akun
			<div class="absolute bottom-0 left-0 h-0.5 w-full bg-[#217b84]"></div>
		</div>
		<a
			href={resolve('/register')}
			class="px-6 py-3 text-sm font-semibold text-gray-400 transition-colors hover:text-gray-600"
		>
			Daftar Baru
		</a>
	</div>

	<form class="space-y-6" onsubmit={handleLogin}>
		{#if error}
			<div
				class="animate-in zoom-in-95 flex flex-col gap-3 rounded-xl border border-red-100 bg-red-50 p-4"
			>
				<div class="flex items-center gap-3">
					<svg
						class="h-5 w-5 flex-shrink-0 text-red-500"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
					<p class="text-xs font-bold text-red-800">{error}</p>
				</div>
				{#if needsVerification}
					<button
						type="button"
						onclick={handleResendVerification}
						disabled={resendLoading}
						class="rounded-lg bg-[#217b84] py-2 text-[11px] font-bold text-white transition-colors hover:bg-[#1a5f66] disabled:opacity-50"
					>
						{resendLoading ? 'Mengirim...' : 'Kirim Ulang Link Verifikasi'}
					</button>
					{#if resendMessage}
						<p class="text-center text-[10px] font-bold text-teal-700">{resendMessage}</p>
					{/if}
				{/if}
				{#if isBackendDown}
					<button
						type="button"
						onclick={handleMockLogin}
						class="rounded-lg border border-red-200 bg-white py-2 text-[11px] font-bold text-red-600 transition-colors hover:bg-red-50"
					>
						Masuk dengan Mode Demo (Mock FE)
					</button>
				{/if}
			</div>
		{/if}

		<div class="space-y-5">
			<div>
				<label for="email" class="mb-2 block px-1 text-[13px] font-bold text-[#0a2e31]">
					Alamat Email <span class="text-teal-600">*</span>
				</label>
				<div class="group relative">
					<div
						class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4 text-gray-300 transition-colors group-focus-within:text-[#217b84]"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
							/>
						</svg>
					</div>
					<input
						id="email"
						type="email"
						required
						bind:value={email}
						class="block w-full rounded-2xl border border-gray-200 bg-gray-50/30 py-3.5 pr-4 pl-11 text-sm transition-all outline-none placeholder:text-gray-400 focus:border-[#217b84] focus:bg-white focus:ring-4 focus:ring-teal-50/50"
						placeholder="contoh@email.com"
					/>
				</div>
			</div>

			<div>
				<div class="mb-2 flex items-center justify-between px-1">
					<label for="password" class="block text-[13px] font-bold text-[#0a2e31]">
						Kata Sandi <span class="text-teal-600">*</span>
					</label>
					<a
						href={resolve('/forgot-password')}
						class="text-[12px] font-bold text-[#217b84] hover:underline"
					>
						Lupa sandi?
					</a>
				</div>
				<div class="group relative">
					<div
						class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4 text-gray-300 transition-colors group-focus-within:text-[#217b84]"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
							/>
						</svg>
					</div>
					<input
						id="password"
						type={showPassword ? 'text' : 'password'}
						required
						bind:value={password}
						class="block w-full rounded-2xl border border-gray-200 bg-gray-50/30 py-3.5 pr-11 pl-11 text-sm transition-all outline-none placeholder:text-gray-400 focus:border-[#217b84] focus:bg-white focus:ring-4 focus:ring-teal-50/50"
						placeholder="Masukkan kata sandi"
					/>
					<button
						type="button"
						aria-label={showPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'}
						onclick={() => (showPassword = !showPassword)}
						class="absolute inset-y-0 right-0 flex items-center pr-4 text-gray-300 hover:text-gray-500"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
							/><path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
							/>
						</svg>
					</button>
				</div>
			</div>
		</div>

		<button
			type="submit"
			disabled={loading}
			class="flex w-full items-center justify-center gap-3 rounded-2xl bg-[#217b84] py-4 text-[15px] font-bold text-white shadow-xl shadow-teal-900/10 transition-all hover:bg-[#1a5f66] active:scale-[0.98] disabled:opacity-70 disabled:active:scale-100"
		>
			{#if loading}
				<div
					class="h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white"
				></div>
			{/if}
			{loading ? 'Memproses...' : 'Masuk ke Dashboard'}
		</button>
	</form>

	<!-- Social Login -->
	<div class="space-y-4">
		<div class="relative">
			<div class="absolute inset-0 flex items-center">
				<div class="w-full border-t border-gray-100"></div>
			</div>
			<div
				class="relative flex justify-center text-[11px] font-bold tracking-widest text-gray-400 uppercase"
			>
				<span class="bg-white px-4">Atau lanjut dengan</span>
			</div>
		</div>

		{#if googleClientId}
			<button
				type="button"
				onclick={handleGoogleLogin}
				disabled={!googleClientId}
				class="group flex w-full items-center justify-center gap-3 rounded-2xl border border-gray-200 bg-white py-3.5 shadow-sm transition-all hover:border-gray-300 hover:shadow-md active:scale-[0.98] disabled:opacity-60 disabled:active:scale-100"
			>
				<svg class="h-5 w-5" viewBox="0 0 24 24">
					<path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
					<path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
					<path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z"/>
					<path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
				</svg>
				<span class="text-[14px] font-bold text-[#0a2e31]">Masuk dengan Google</span>
			</button>
		{/if}

		<button
			type="button"
			onclick={handleMockLogin}
			class="flex w-full items-center justify-center gap-2 rounded-2xl border border-dashed border-gray-200 py-3 text-[13px] font-semibold text-gray-400 transition-all hover:border-gray-300 hover:text-gray-500 active:scale-[0.98]"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17H3a2 2 0 01-2-2V5a2 2 0 012-2h14a2 2 0 012 2v10a2 2 0 01-2 2h-2" />
			</svg>
			Mode Demo (tanpa akun)
		</button>
	</div>
</div>
