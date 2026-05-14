<script lang="ts">
	import { apiFetch } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { goto } from '$app/navigation';

	let email = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state<string | null>(null);
	let isBackendDown = $state(false);
	let showPassword = $state(false);

	async function handleLogin(e: SubmitEvent) {
		e.preventDefault();
		loading = true;
		error = null;
		isBackendDown = false;

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
				goto('/dashboard');
			} else {
				error = result.error?.message || 'Email atau password salah.';
			}
		} catch (err) {
			isBackendDown = true;
			error = 'Backend belum aktif. Gunakan Mode Demo untuk mencoba UI.';
			console.error(err);
		} finally {
			loading = false;
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
		goto('/dashboard');
	}
</script>

<div class="space-y-10 animate-in fade-in slide-in-from-bottom-4 duration-700">
	<div class="space-y-3 text-center">
		<h2 class="text-3xl font-bold tracking-tight text-[#0a2e31]">Selamat Datang</h2>
		<p class="text-[15px] text-gray-500 leading-relaxed max-w-[320px] mx-auto">
			Kelola dan pantau pertumbuhan aset Anda dalam satu platform terpadu.
		</p>
	</div>

	<!-- Tab Switcher -->
	<div class="flex justify-center border-b border-gray-100">
		<div class="relative px-6 py-3 text-sm font-bold text-[#0a2e31] transition-colors">
			Masuk Akun
			<div class="absolute bottom-0 left-0 h-0.5 w-full bg-[#217b84]"></div>
		</div>
		<a href="/register" class="px-6 py-3 text-sm font-semibold text-gray-400 hover:text-gray-600 transition-colors">
			Daftar Baru
		</a>
	</div>

	<form class="space-y-6" onsubmit={handleLogin}>
		{#if error}
			<div class="rounded-xl bg-red-50 p-4 border border-red-100 flex flex-col gap-3 animate-in zoom-in-95">
				<div class="flex gap-3 items-center">
					<svg class="h-5 w-5 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					<p class="text-xs font-bold text-red-800">{error}</p>
				</div>
				{#if isBackendDown}
					<button 
						type="button"
						onclick={handleMockLogin}
						class="text-[11px] bg-white border border-red-200 text-red-600 py-2 rounded-lg font-bold hover:bg-red-50 transition-colors"
					>
						Masuk dengan Mode Demo (Mock FE)
					</button>
				{/if}
			</div>
		{/if}

		<div class="space-y-5">
			<div>
				<label for="email" class="block text-[13px] font-bold text-[#0a2e31] mb-2 px-1">
					Alamat Email <span class="text-teal-600">*</span>
				</label>
				<div class="relative group">
					<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-gray-300 group-focus-within:text-[#217b84] transition-colors">
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
						</svg>
					</div>
					<input
						id="email"
						type="email"
						required
						bind:value={email}
						class="block w-full pl-11 pr-4 py-3.5 bg-gray-50/30 border border-gray-200 rounded-2xl text-sm transition-all focus:ring-4 focus:ring-teal-50/50 focus:border-[#217b84] focus:bg-white outline-none placeholder:text-gray-400"
						placeholder="contoh@email.com"
					/>
				</div>
			</div>

			<div>
				<div class="flex justify-between items-center mb-2 px-1">
					<label for="password" class="block text-[13px] font-bold text-[#0a2e31]">
						Kata Sandi <span class="text-teal-600">*</span>
					</label>
					<a href="/forgot-password" class="text-[12px] font-bold text-[#217b84] hover:underline">
						Lupa sandi?
					</a>
				</div>
				<div class="relative group">
					<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-gray-300 group-focus-within:text-[#217b84] transition-colors">
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
						</svg>
					</div>
					<input
						id="password"
						type={showPassword ? 'text' : 'password'}
						required
						bind:value={password}
						class="block w-full pl-11 pr-11 py-3.5 bg-gray-50/30 border border-gray-200 rounded-2xl text-sm transition-all focus:ring-4 focus:ring-teal-50/50 focus:border-[#217b84] focus:bg-white outline-none placeholder:text-gray-400"
						placeholder="Masukkan kata sandi"
					/>
					<button
						type="button"
						aria-label={showPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'}
						onclick={() => showPassword = !showPassword}
						class="absolute inset-y-0 right-0 pr-4 flex items-center text-gray-300 hover:text-gray-500"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
						</svg>
					</button>
				</div>
			</div>
		</div>

		<button
			type="submit"
			disabled={loading}
			class="w-full py-4 bg-[#217b84] hover:bg-[#1a5f66] text-white text-[15px] font-bold rounded-2xl shadow-xl shadow-teal-900/10 transition-all active:scale-[0.98] disabled:opacity-70 disabled:active:scale-100 flex items-center justify-center gap-3"
		>
			{#if loading}
				<div class="h-5 w-5 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
			{/if}
			{loading ? 'Memproses...' : 'Masuk ke Dashboard'}
		</button>
	</form>

	<!-- Social Login -->
	<div class="space-y-6">
		<div class="relative">
			<div class="absolute inset-0 flex items-center"><div class="w-full border-t border-gray-100"></div></div>
			<div class="relative flex justify-center text-[11px] font-bold uppercase tracking-widest text-gray-400"><span class="bg-white px-4">Atau lanjut dengan</span></div>
		</div>

		<button 
			type="button"
			onclick={handleMockLogin}
			class="w-full flex items-center justify-center gap-3 py-3.5 border-2 border-gray-50 rounded-2xl hover:bg-gray-50 hover:border-gray-100 transition-all active:scale-[0.98] group"
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
