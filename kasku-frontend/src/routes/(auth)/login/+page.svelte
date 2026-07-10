<script lang="ts">
	import { apiFetch } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { env } from '$env/dynamic/public';
	import { startGoogleLogin } from '$lib/googleOAuth';

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

</script>

<div class="space-y-8">
	<div>
		<h1 class="font-serif text-[34px] leading-tight tracking-tight text-ink">
			Selamat datang kembali
		</h1>
		<p class="mt-2.5 text-sm text-ink/60">Masuk untuk melanjutkan pencatatan.</p>
	</div>

	<form class="space-y-5" onsubmit={handleLogin}>
		{#if error}
			<div class="flex flex-col gap-3 rounded-xl border border-clay/25 bg-clay/5 p-4">
				<div class="flex items-start gap-2.5">
					<svg
						class="mt-px h-4 w-4 shrink-0 text-clay"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
					<p class="text-[13px] font-medium text-clay">{error}</p>
				</div>
				{#if needsVerification}
					<button
						type="button"
						onclick={handleResendVerification}
						disabled={resendLoading}
						class="rounded-full bg-teal py-2 text-[12px] font-semibold text-card transition-colors hover:bg-ink disabled:opacity-50"
					>
						{resendLoading ? 'Mengirim…' : 'Kirim ulang link verifikasi'}
					</button>
					{#if resendMessage}
						<p class="text-center text-[11px] font-medium text-teal">{resendMessage}</p>
					{/if}
				{/if}
				{#if isBackendDown}
					<button
						type="button"
						onclick={handleMockLogin}
						class="rounded-full border border-clay/30 bg-field py-2 text-[12px] font-semibold text-clay transition-colors hover:bg-clay/10"
					>
						Masuk dengan Mode Demo
					</button>
				{/if}
			</div>
		{/if}

		<div>
			<label
				for="email"
				class="mb-2 block text-[11px] font-semibold tracking-[0.12em] text-ink/50 uppercase"
			>
				Email
			</label>
			<input
				id="email"
				type="email"
				required
				bind:value={email}
				placeholder="contoh@email.com"
				class="w-full rounded-[10px] border border-ink/15 bg-field px-4 py-3 text-sm text-ink shadow-sm transition outline-none placeholder:text-ink/35 focus:border-teal focus:ring-2 focus:ring-teal/15"
			/>
		</div>

		<div>
			<div class="mb-2 flex items-baseline justify-between">
				<label
					for="password"
					class="text-[11px] font-semibold tracking-[0.12em] text-ink/50 uppercase"
				>
					Kata sandi
				</label>
				<a
					href={resolve('/forgot-password')}
					class="text-[12px] font-semibold text-teal hover:text-ink"
				>
					Lupa sandi?
				</a>
			</div>
			<div class="relative">
				<input
					id="password"
					type={showPassword ? 'text' : 'password'}
					required
					bind:value={password}
					placeholder="Masukkan kata sandi"
					class="w-full rounded-[10px] border border-ink/15 bg-field px-4 py-3 pr-11 text-sm text-ink shadow-sm transition outline-none placeholder:text-ink/35 focus:border-teal focus:ring-2 focus:ring-teal/15"
				/>
				<button
					type="button"
					aria-label={showPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'}
					onclick={() => (showPassword = !showPassword)}
					class="absolute inset-y-0 right-0 flex items-center pr-4 text-ink/35 hover:text-ink"
				>
					{#if showPassword}
						<svg
							class="h-5 w-5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.8"
							><path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.542 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
							/></svg
						>
					{:else}
						<svg
							class="h-5 w-5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.8"
							><path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
							/><path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
							/></svg
						>
					{/if}
				</button>
			</div>
		</div>

		<button
			type="submit"
			disabled={loading}
			class="flex w-full items-center justify-center gap-2.5 rounded-full bg-teal py-3.5 text-sm font-semibold text-card transition-colors hover:bg-ink disabled:opacity-70"
		>
			{#if loading}
				<span class="h-4 w-4 animate-spin rounded-full border-2 border-card/40 border-t-card"
				></span>
			{/if}
			{loading ? 'Memproses…' : 'Masuk'}
		</button>

		{#if googleClientId}
			<button
				type="button"
				onclick={() => startGoogleLogin(googleClientId)}
				class="flex w-full items-center justify-center gap-2.5 rounded-full border border-ink/25 bg-field py-3 text-sm font-semibold text-ink transition-colors hover:border-ink/40"
			>
				<svg class="h-4 w-4" viewBox="0 0 24 24">
					<path
						fill="#4285F4"
						d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
					/>
					<path
						fill="#34A853"
						d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
					/>
					<path
						fill="#FBBC05"
						d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z"
					/>
					<path
						fill="#EA4335"
						d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
					/>
				</svg>
				Masuk dengan Google
			</button>
		{/if}

		<button
			type="button"
			onclick={handleMockLogin}
			class="flex w-full items-center justify-center gap-2 rounded-full border border-dashed border-ink/20 py-2.5 text-[13px] font-medium text-ink/45 transition-colors hover:border-ink/35 hover:text-ink/70"
		>
			Mode demo (tanpa akun)
		</button>
	</form>

	<p class="text-center text-[13px] text-ink/55">
		Belum punya akun?
		<a href={resolve('/register')} class="font-semibold text-teal hover:text-ink">Daftar gratis</a>
	</p>
</div>
