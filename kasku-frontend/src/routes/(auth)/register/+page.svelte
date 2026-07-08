<script lang="ts">
	import { apiFetch } from '$lib/api/client';
	import { resolve } from '$app/paths';

	let email = $state('');
	let username = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let loading = $state(false);
	let error = $state<string | null>(null);
	let success = $state(false);
	let showPassword = $state(false);
	let showConfirmPassword = $state(false);

	// Password Validation Logic
	let validations = $derived({
		minLength: password.length >= 6,
		hasUpper: /[A-Z]/.test(password),
		hasNumber: /[0-9]/.test(password),
		hasSymbol: /[^A-Za-z0-9]/.test(password),
		match: password === confirmPassword && confirmPassword !== ''
	});

	let isPasswordStrong = $derived(
		validations.minLength && validations.hasUpper && validations.hasNumber && validations.hasSymbol
	);

	const requirements = $derived([
		{ ok: validations.minLength, label: '6+ Karakter' },
		{ ok: validations.hasUpper, label: 'Huruf Besar' },
		{ ok: validations.hasNumber, label: 'Angka' },
		{ ok: validations.hasSymbol, label: 'Simbol' }
	]);

	async function handleRegister(e: SubmitEvent) {
		e.preventDefault();

		if (!isPasswordStrong) {
			error = 'Katasandi belum memenuhi kriteria keamanan.';
			return;
		}

		if (!validations.match) {
			error = 'Konfirmasi katasandi tidak cocok.';
			return;
		}

		loading = true;
		error = null;

		try {
			const response = await apiFetch('/auth/register', {
				method: 'POST',
				body: JSON.stringify({ email, username, password }),
				skipAuth: true
			});

			const result = await response.json();

			if (result.success) {
				success = true;
			} else {
				error = result.error?.message || 'Registrasi gagal. Silakan coba lagi.';
			}
		} catch (err) {
			error = 'Terjadi kesalahan koneksi.';
			console.error(err);
		} finally {
			loading = false;
		}
	}
</script>

<div class="space-y-8">
	<div>
		<h1 class="font-serif text-[34px] leading-tight tracking-tight text-ink">Buat akun baru</h1>
		<p class="mt-2.5 text-sm text-ink/60">Bergabung dan mulai kendalikan masa depan finansialmu.</p>
	</div>

	{#if success}
		<div class="space-y-6 text-center">
			<div
				class="mx-auto flex h-16 w-16 items-center justify-center rounded-full border border-teal/25 bg-teal/5"
			>
				<svg
					class="h-8 w-8 text-teal"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
				</svg>
			</div>
			<div>
				<h2 class="font-serif text-[26px] leading-tight tracking-tight text-ink">
					Pendaftaran berhasil
				</h2>
				<p class="mt-2.5 text-sm text-ink/60">
					Tautan verifikasi telah dikirim ke emailmu. Silakan periksa kotak masuk untuk mengaktifkan
					akun.
				</p>
			</div>
			<a
				href={resolve('/login')}
				class="flex w-full items-center justify-center gap-2.5 rounded-full bg-teal py-3.5 text-sm font-semibold text-card transition-colors hover:bg-ink"
			>
				Kembali ke masuk
			</a>
		</div>
	{:else}
		<form class="space-y-5" onsubmit={handleRegister}>
			{#if error}
				<div class="flex items-start gap-2.5 rounded-xl border border-clay/25 bg-clay/5 p-4">
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
			{/if}

			<div>
				<label
					for="username"
					class="mb-2 block text-[11px] font-semibold tracking-[0.12em] text-ink/50 uppercase"
				>
					Nama pengguna
				</label>
				<input
					id="username"
					type="text"
					required
					bind:value={username}
					placeholder="Pilih nama unik"
					class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-teal"
				/>
			</div>

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
					placeholder="nama@email.com"
					class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-teal"
				/>
			</div>

			<div>
				<label
					for="password"
					class="mb-2 block text-[11px] font-semibold tracking-[0.12em] text-ink/50 uppercase"
				>
					Kata sandi
				</label>
				<div class="relative">
					<input
						id="password"
						type={showPassword ? 'text' : 'password'}
						required
						bind:value={password}
						placeholder="Minimal 6 karakter"
						class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 pr-11 text-sm text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-teal"
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

			<!-- Password Requirements Indicator -->
			<div class="grid grid-cols-2 gap-2">
				{#each requirements as req (req.label)}
					<div class="flex items-center gap-2 {req.ok ? 'text-teal' : 'text-ink/35'}">
						<div class="h-1.5 w-1.5 rounded-full bg-current"></div>
						<span class="text-[10px] font-semibold tracking-[0.08em] uppercase">{req.label}</span>
					</div>
				{/each}
			</div>

			<div>
				<label
					for="confirm-password"
					class="mb-2 block text-[11px] font-semibold tracking-[0.12em] text-ink/50 uppercase"
				>
					Konfirmasi kata sandi
				</label>
				<div class="relative">
					<input
						id="confirm-password"
						type={showConfirmPassword ? 'text' : 'password'}
						required
						bind:value={confirmPassword}
						placeholder="Ulangi kata sandi"
						class="w-full rounded-[10px] border bg-field px-4 py-3 pr-11 text-sm text-ink transition-colors outline-none placeholder:text-ink/30 {confirmPassword !==
							'' && !validations.match
							? 'border-clay/50 focus:border-clay'
							: 'border-ink/25 focus:border-teal'}"
					/>
					<button
						type="button"
						aria-label={showConfirmPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'}
						onclick={() => (showConfirmPassword = !showConfirmPassword)}
						class="absolute inset-y-0 right-0 flex items-center pr-4 text-ink/35 hover:text-ink"
					>
						{#if showConfirmPassword}
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
				{#if confirmPassword !== '' && !validations.match}
					<p class="mt-2 text-[12px] font-medium text-clay">Katasandi tidak cocok.</p>
				{/if}
			</div>

			<button
				type="submit"
				disabled={loading || !isPasswordStrong || !validations.match}
				class="flex w-full items-center justify-center gap-2.5 rounded-full bg-teal py-3.5 text-sm font-semibold text-card transition-colors hover:bg-ink disabled:opacity-70"
			>
				{#if loading}
					<span class="h-4 w-4 animate-spin rounded-full border-2 border-card/40 border-t-card"
					></span>
				{/if}
				{loading ? 'Mendaftarkan…' : 'Daftar sekarang'}
			</button>

			<button
				type="button"
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
				Daftar dengan Google
			</button>
		</form>

		<p class="text-center text-[13px] text-ink/55">
			Sudah punya akun?
			<a href={resolve('/login')} class="font-semibold text-teal hover:text-ink">Masuk</a>
		</p>
	{/if}
</div>
