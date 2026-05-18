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

<div class="animate-in fade-in slide-in-from-bottom-4 space-y-10 duration-700">
	<div class="space-y-3 text-center">
		<h2 class="text-3xl font-bold tracking-tight text-[#0a2e31]">Buat Akun Baru</h2>
		<p class="mx-auto max-w-[320px] text-[15px] leading-relaxed text-gray-500">
			Bergabunglah dan mulai kendalikan masa depan finansial Anda hari ini.
		</p>
	</div>

	<!-- Tab Switcher (Center Aligned) -->
	<div class="flex justify-center border-b border-gray-100">
		<a
			href={resolve('/login')}
			class="px-6 py-3 text-sm font-semibold text-gray-400 transition-colors hover:text-gray-600"
		>
			Masuk Akun
		</a>
		<div class="relative px-6 py-3 text-sm font-bold text-[#0a2e31] transition-colors">
			Daftar Baru
			<div class="absolute bottom-0 left-0 h-0.5 w-full bg-[#217b84]"></div>
		</div>
	</div>

	{#if success}
		<div class="animate-in zoom-in-95 space-y-8 py-10 text-center duration-500">
			<div
				class="mx-auto flex h-24 w-24 items-center justify-center rounded-full border-2 border-teal-100 bg-teal-50 shadow-inner"
			>
				<svg class="h-12 w-12 text-teal-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2.5"
						d="M5 13l4 4L19 7"
					/>
				</svg>
			</div>
			<div class="space-y-3">
				<h3 class="text-2xl font-black text-[#0a2e31]">Pendaftaran Berhasil!</h3>
				<p class="px-6 text-[15px] leading-relaxed font-medium text-gray-600">
					Tautan verifikasi telah dikirim ke email Anda. Silakan periksa kotak masuk untuk
					mengaktifkan akun.
				</p>
			</div>
			<a
				href={resolve('/login')}
				class="block w-full rounded-2xl bg-[#217b84] py-4 text-center font-bold text-white shadow-xl transition-all hover:bg-[#1a5f66] active:scale-[0.98]"
			>
				Kembali ke Login
			</a>
		</div>
	{:else}
		<form class="space-y-6" onsubmit={handleRegister}>
			{#if error}
				<div
					class="animate-in zoom-in-95 flex items-center gap-3 rounded-xl border border-red-100 bg-red-50 p-4"
				>
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
			{/if}

			<div class="space-y-5">
				<div>
					<label for="username" class="mb-2 block px-1 text-[13px] font-bold text-[#0a2e31]">
						Nama Pengguna <span class="text-teal-600">*</span>
					</label>
					<div class="group relative">
						<div
							class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4 text-gray-300 transition-colors group-focus-within:text-[#217b84]"
						>
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"
								><path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
								/></svg
							>
						</div>
						<input
							id="username"
							type="text"
							required
							bind:value={username}
							class="block w-full rounded-2xl border border-gray-200 bg-gray-50/30 py-3.5 pr-4 pl-11 text-sm transition-all outline-none placeholder:text-gray-400 focus:border-[#217b84] focus:bg-white focus:ring-4 focus:ring-teal-50/50"
							placeholder="Pilih nama unik"
						/>
					</div>
				</div>

				<div>
					<label for="email" class="mb-2 block px-1 text-[13px] font-bold text-[#0a2e31]">
						Alamat Email <span class="text-teal-600">*</span>
					</label>
					<div class="group relative">
						<div
							class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4 text-gray-300 transition-colors group-focus-within:text-[#217b84]"
						>
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"
								><path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
								/></svg
							>
						</div>
						<input
							id="email"
							type="email"
							required
							bind:value={email}
							class="block w-full rounded-2xl border border-gray-200 bg-gray-50/30 py-3.5 pr-4 pl-11 text-sm transition-all outline-none placeholder:text-gray-400 focus:border-[#217b84] focus:bg-white focus:ring-4 focus:ring-teal-50/50"
							placeholder="nama@email.com"
						/>
					</div>
				</div>

				<div class="space-y-3">
					<div>
						<label for="password" class="mb-2 block px-1 text-[13px] font-bold text-[#0a2e31]">
							Kata Sandi <span class="text-teal-600">*</span>
						</label>
						<div class="group relative">
							<div
								class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4 text-gray-300 transition-colors group-focus-within:text-[#217b84]"
							>
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"
									><path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
									/></svg
								>
							</div>
							<input
								id="password"
								type="password"
								required
								bind:value={password}
								class="block w-full rounded-2xl border border-gray-200 bg-gray-50/30 py-3.5 pr-4 pl-11 text-sm transition-all outline-none placeholder:text-gray-400 focus:border-[#217b84] focus:bg-white focus:ring-4 focus:ring-teal-50/50"
								placeholder="Minimal 6 karakter"
							/>
						</div>
					</div>

					<!-- Password Requirements Indicator -->
					<div class="grid grid-cols-2 gap-2 px-1">
						<div
							class="flex items-center gap-2 {validations.minLength
								? 'text-teal-600'
								: 'text-gray-400'}"
						>
							<div class="h-1.5 w-1.5 rounded-full bg-current"></div>
							<span class="text-[10px] font-bold tracking-tight uppercase">6+ Karakter</span>
						</div>
						<div
							class="flex items-center gap-2 {validations.hasUpper
								? 'text-teal-600'
								: 'text-gray-400'}"
						>
							<div class="h-1.5 w-1.5 rounded-full bg-current"></div>
							<span class="text-[10px] font-bold tracking-tight uppercase">Huruf Besar</span>
						</div>
						<div
							class="flex items-center gap-2 {validations.hasNumber
								? 'text-teal-600'
								: 'text-gray-400'}"
						>
							<div class="h-1.5 w-1.5 rounded-full bg-current"></div>
							<span class="text-[10px] font-bold tracking-tight uppercase">Angka</span>
						</div>
						<div
							class="flex items-center gap-2 {validations.hasSymbol
								? 'text-teal-600'
								: 'text-gray-400'}"
						>
							<div class="h-1.5 w-1.5 rounded-full bg-current"></div>
							<span class="text-[10px] font-bold tracking-tight uppercase">Simbol</span>
						</div>
					</div>

					<div>
						<label
							for="confirm-password"
							class="mb-2 block px-1 text-[13px] font-bold text-[#0a2e31]"
						>
							Konfirmasi Kata Sandi <span class="text-teal-600">*</span>
						</label>
						<div class="group relative">
							<div
								class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4 text-gray-300 transition-colors group-focus-within:text-[#217b84]"
							>
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"
									><path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04c0 4.833 1.277 9.473 3.535 13.502a11.952 11.952 0 0010.166 0c2.258-4.029 3.535-8.669 3.535-13.502z"
									/></svg
								>
							</div>
							<input
								id="confirm-password"
								type="password"
								required
								bind:value={confirmPassword}
								class="block w-full border bg-gray-50/30 py-3.5 pr-4 pl-11 ${confirmPassword !==
									'' && !validations.match
									? 'border-red-300'
									: 'border-gray-200'} rounded-2xl text-sm transition-all focus:ring-4 ${confirmPassword !==
									'' && !validations.match
									? 'focus:border-red-400 focus:ring-red-50'
									: 'focus:border-[#217b84] focus:ring-teal-50/50'} outline-none placeholder:text-gray-400 focus:bg-white"
								placeholder="Ulangi kata sandi"
							/>
						</div>
						{#if confirmPassword !== '' && !validations.match}
							<p class="mt-1.5 px-1 text-[10px] font-bold text-red-500 italic">
								Katasandi tidak cocok.
							</p>
						{/if}
					</div>
				</div>
			</div>

			<button
				type="submit"
				disabled={loading || !isPasswordStrong || !validations.match}
				class="flex w-full items-center justify-center gap-3 rounded-2xl bg-[#217b84] py-4 text-[15px] font-bold text-white shadow-xl shadow-teal-900/10 transition-all hover:bg-[#1a5f66] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-40"
			>
				{#if loading}
					<div
						class="h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white"
					></div>
				{/if}
				{loading ? 'Mendaftarkan...' : 'Daftar Sekarang'}
			</button>
		</form>

		<!-- Social Login -->
		<div class="space-y-6">
			<div class="relative">
				<div class="absolute inset-0 flex items-center">
					<div class="w-full border-t border-gray-100"></div>
				</div>
				<div
					class="relative flex justify-center text-[11px] font-bold tracking-widest text-gray-400 uppercase"
				>
					<span class="bg-white px-4">Atau daftar dengan</span>
				</div>
			</div>

			<button
				class="group flex w-full items-center justify-center gap-3 rounded-2xl border-2 border-gray-50 py-3.5 transition-all hover:border-gray-100 hover:bg-gray-50 active:scale-[0.98]"
			>
				<img
					src="https://www.gstatic.com/firebasejs/ui/2.0.0/images/auth/google.svg"
					alt="Google"
					class="h-5 w-5"
				/>
				<span class="text-[14px] font-bold text-[#0a2e31]">Google</span>
			</button>
		</div>
	{/if}
</div>
