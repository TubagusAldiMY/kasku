<script lang="ts">
	import { apiFetch } from '$lib/api/client';
	import { goto } from '$app/navigation';

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

	function ValidationItem({ met, label }: { met: boolean; label: string }) {
		return `
			<div class="flex items-center gap-1.5 ${met ? 'text-teal-600' : 'text-gray-400'} transition-colors duration-300">
				<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					${met 
						? '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />' 
						: '<circle cx="12" cy="12" r="10" stroke-width="2" />'
					}
				</svg>
				<span class="text-[10px] font-bold uppercase tracking-wider">${label}</span>
			</div>
		`;
	}
</script>

<div class="space-y-10 animate-in fade-in slide-in-from-bottom-4 duration-700">
	<div class="space-y-3 text-center">
		<h2 class="text-3xl font-bold tracking-tight text-[#0a2e31]">Buat Akun Baru</h2>
		<p class="text-[15px] text-gray-500 leading-relaxed max-w-[320px] mx-auto">
			Bergabunglah dan mulai kendalikan masa depan finansial Anda hari ini.
		</p>
	</div>

	<!-- Tab Switcher (Center Aligned) -->
	<div class="flex justify-center border-b border-gray-100">
		<a href="/login" class="px-6 py-3 text-sm font-semibold text-gray-400 hover:text-gray-600 transition-colors">
			Masuk Akun
		</a>
		<div class="relative px-6 py-3 text-sm font-bold text-[#0a2e31] transition-colors">
			Daftar Baru
			<div class="absolute bottom-0 left-0 h-0.5 w-full bg-[#217b84]"></div>
		</div>
	</div>

	{#if success}
		<div class="text-center py-10 space-y-8 animate-in zoom-in-95 duration-500">
			<div class="mx-auto flex items-center justify-center h-24 w-24 rounded-full bg-teal-50 border-2 border-teal-100 shadow-inner">
				<svg class="h-12 w-12 text-teal-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
				</svg>
			</div>
			<div class="space-y-3">
				<h3 class="text-2xl font-black text-[#0a2e31]">Pendaftaran Berhasil!</h3>
				<p class="text-[15px] text-gray-600 px-6 leading-relaxed font-medium">
					Tautan verifikasi telah dikirim ke email Anda. Silakan periksa kotak masuk untuk mengaktifkan akun.
				</p>
			</div>
			<a href="/login" class="block w-full py-4 bg-[#217b84] hover:bg-[#1a5f66] text-white font-bold rounded-2xl shadow-xl transition-all active:scale-[0.98] text-center">
				Kembali ke Login
			</a>
		</div>
	{:else}
		<form class="space-y-6" onsubmit={handleRegister}>
			{#if error}
				<div class="rounded-xl bg-red-50 p-4 border border-red-100 flex gap-3 items-center animate-in zoom-in-95">
					<svg class="h-5 w-5 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					<p class="text-xs font-bold text-red-800">{error}</p>
				</div>
			{/if}

			<div class="space-y-5">
				<div>
					<label for="username" class="block text-[13px] font-bold text-[#0a2e31] mb-2 px-1">
						Nama Pengguna <span class="text-teal-600">*</span>
					</label>
					<div class="relative group">
						<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-gray-300 group-focus-within:text-[#217b84] transition-colors">
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" /></svg>
						</div>
						<input
							id="username"
							type="text"
							required
							bind:value={username}
							class="block w-full pl-11 pr-4 py-3.5 bg-gray-50/30 border border-gray-200 rounded-2xl text-sm transition-all focus:ring-4 focus:ring-teal-50/50 focus:border-[#217b84] focus:bg-white outline-none placeholder:text-gray-400"
							placeholder="Pilih nama unik"
						/>
					</div>
				</div>

				<div>
					<label for="email" class="block text-[13px] font-bold text-[#0a2e31] mb-2 px-1">
						Alamat Email <span class="text-teal-600">*</span>
					</label>
					<div class="relative group">
						<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-gray-300 group-focus-within:text-[#217b84] transition-colors">
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" /></svg>
						</div>
						<input
							id="email"
							type="email"
							required
							bind:value={email}
							class="block w-full pl-11 pr-4 py-3.5 bg-gray-50/30 border border-gray-200 rounded-2xl text-sm transition-all focus:ring-4 focus:ring-teal-50/50 focus:border-[#217b84] focus:bg-white outline-none placeholder:text-gray-400"
							placeholder="nama@email.com"
						/>
					</div>
				</div>

				<div class="space-y-3">
					<div>
						<label for="password" class="block text-[13px] font-bold text-[#0a2e31] mb-2 px-1">
							Kata Sandi <span class="text-teal-600">*</span>
						</label>
						<div class="relative group">
							<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-gray-300 group-focus-within:text-[#217b84] transition-colors">
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" /></svg>
							</div>
							<input
								id="password"
								type="password"
								required
								bind:value={password}
								class="block w-full pl-11 pr-4 py-3.5 bg-gray-50/30 border border-gray-200 rounded-2xl text-sm transition-all focus:ring-4 focus:ring-teal-50/50 focus:border-[#217b84] focus:bg-white outline-none placeholder:text-gray-400"
								placeholder="Minimal 6 karakter"
							/>
						</div>
					</div>

					<!-- Password Requirements Indicator -->
					<div class="grid grid-cols-2 gap-2 px-1">
						<div class="flex items-center gap-2 {validations.minLength ? 'text-teal-600' : 'text-gray-400'}">
							<div class="h-1.5 w-1.5 rounded-full bg-current"></div>
							<span class="text-[10px] font-bold uppercase tracking-tight">6+ Karakter</span>
						</div>
						<div class="flex items-center gap-2 {validations.hasUpper ? 'text-teal-600' : 'text-gray-400'}">
							<div class="h-1.5 w-1.5 rounded-full bg-current"></div>
							<span class="text-[10px] font-bold uppercase tracking-tight">Huruf Besar</span>
						</div>
						<div class="flex items-center gap-2 {validations.hasNumber ? 'text-teal-600' : 'text-gray-400'}">
							<div class="h-1.5 w-1.5 rounded-full bg-current"></div>
							<span class="text-[10px] font-bold uppercase tracking-tight">Angka</span>
						</div>
						<div class="flex items-center gap-2 {validations.hasSymbol ? 'text-teal-600' : 'text-gray-400'}">
							<div class="h-1.5 w-1.5 rounded-full bg-current"></div>
							<span class="text-[10px] font-bold uppercase tracking-tight">Simbol</span>
						</div>
					</div>

					<div>
						<label for="confirm-password" class="block text-[13px] font-bold text-[#0a2e31] mb-2 px-1">
							Konfirmasi Kata Sandi <span class="text-teal-600">*</span>
						</label>
						<div class="relative group">
							<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-gray-300 group-focus-within:text-[#217b84] transition-colors">
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04c0 4.833 1.277 9.473 3.535 13.502a11.952 11.952 0 0010.166 0c2.258-4.029 3.535-8.669 3.535-13.502z" /></svg>
							</div>
							<input
								id="confirm-password"
								type="password"
								required
								bind:value={confirmPassword}
								class="block w-full pl-11 pr-4 py-3.5 bg-gray-50/30 border ${confirmPassword !== '' && !validations.match ? 'border-red-300' : 'border-gray-200'} rounded-2xl text-sm transition-all focus:ring-4 ${confirmPassword !== '' && !validations.match ? 'focus:ring-red-50 focus:border-red-400' : 'focus:ring-teal-50/50 focus:border-[#217b84]'} focus:bg-white outline-none placeholder:text-gray-400"
								placeholder="Ulangi kata sandi"
							/>
						</div>
						{#if confirmPassword !== '' && !validations.match}
							<p class="mt-1.5 text-[10px] font-bold text-red-500 px-1 italic">Katasandi tidak cocok.</p>
						{/if}
					</div>
				</div>
			</div>

			<button
				type="submit"
				disabled={loading || !isPasswordStrong || !validations.match}
				class="w-full py-4 bg-[#217b84] hover:bg-[#1a5f66] text-white text-[15px] font-bold rounded-2xl shadow-xl shadow-teal-900/10 transition-all active:scale-[0.98] disabled:opacity-40 disabled:cursor-not-allowed flex items-center justify-center gap-3"
			>
				{#if loading}
					<div class="h-5 w-5 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
				{/if}
				{loading ? 'Mendaftarkan...' : 'Daftar Sekarang'}
			</button>
		</form>

		<!-- Social Login -->
		<div class="space-y-6">
			<div class="relative">
				<div class="absolute inset-0 flex items-center"><div class="w-full border-t border-gray-100"></div></div>
				<div class="relative flex justify-center text-[11px] font-bold uppercase tracking-widest text-gray-400"><span class="bg-white px-4">Atau daftar dengan</span></div>
			</div>

			<button class="w-full flex items-center justify-center gap-3 py-3.5 border-2 border-gray-50 rounded-2xl hover:bg-gray-50 hover:border-gray-100 transition-all active:scale-[0.98] group">
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
