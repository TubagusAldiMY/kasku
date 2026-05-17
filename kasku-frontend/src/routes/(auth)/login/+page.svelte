<script lang="ts">
	import { apiFetch } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

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
	<div class="space-y-6">
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

		<button
			type="button"
			onclick={handleMockLogin}
			class="group flex w-full items-center justify-center gap-3 rounded-2xl border-2 border-gray-50 py-3.5 transition-all hover:border-gray-100 hover:bg-gray-50 active:scale-[0.98]"
		>
			<img
				src="https://www.gstatic.com/firebasejs/ui/2.0.0/images/auth/google.svg"
				alt="Google"
				class="h-5 w-5"
			/>
			<span class="text-[14px] font-bold text-[#0a2e31]">Google (Mode Demo)</span>
		</button>
	</div>
</div>
