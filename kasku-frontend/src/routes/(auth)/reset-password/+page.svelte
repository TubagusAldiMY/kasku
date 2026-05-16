<script lang="ts">
	import { apiFetch } from '$lib/api/client';
	import { page } from '$app/state';
	import { fade, fly } from 'svelte/transition';
	import { goto } from '$app/navigation';

	let newPassword = $state('');
	let confirmPassword = $state('');
	let loading = $state(false);
	let showPassword = $state(false);
	let message = $state<{ type: 'success' | 'error', text: string } | null>(null);

	const token = page.url.searchParams.get('token');

	async function handleResetPassword(e: SubmitEvent) {
		e.preventDefault();
		
		if (!token) {
			message = { type: 'error', text: 'Token tidak valid atau sudah kadaluarsa.' };
			return;
		}

		if (newPassword !== confirmPassword) {
			message = { type: 'error', text: 'Konfirmasi kata sandi tidak cocok.' };
			return;
		}

		loading = true;
		message = null;

		try {
			const response = await apiFetch('/auth/reset-password', {
				method: 'POST',
				body: JSON.stringify({ token, newPassword }),
				skipAuth: true
			});

			const result = await response.json();

			if (result.success) {
				message = { type: 'success', text: 'Kata sandi berhasil diperbarui! Mengalihkan ke halaman login...' };
				setTimeout(() => {
					goto('/login');
				}, 2000);
			} else {
				message = { 
					type: 'error', 
					text: result.error?.message || 'Gagal memperbarui kata sandi.' 
				};
			}
		} catch (err) {
			message = { 
				type: 'error', 
				text: 'Terjadi kesalahan koneksi.' 
			};
			console.error(err);
		} finally {
			loading = false;
		}
	}
</script>

<div class="space-y-10 animate-in fade-in slide-in-from-bottom-4 duration-700">
	<div class="space-y-3 text-center">
		<h2 class="text-3xl font-bold tracking-tight text-[#0a2e31]">Atur Ulang Sandi</h2>
		<p class="text-[15px] text-gray-500 leading-relaxed max-w-[320px] mx-auto">
			Silakan masukkan kata sandi baru untuk akun Anda.
		</p>
	</div>

	{#if !token}
		<div class="p-6 rounded-[2rem] bg-red-50 border border-red-100 text-center space-y-4">
			<svg class="h-12 w-12 text-red-400 mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<p class="text-sm font-bold text-red-800">Tautan tidak valid atau sudah kadaluarsa.</p>
			<a href="/forgot-password" class="inline-block text-xs font-black uppercase tracking-widest text-[#217b84] hover:underline">
				Minta Tautan Baru
			</a>
		</div>
	{:else}
		{#if message}
			<div 
				in:fly={{ y: -10, duration: 400 }}
				class="p-4 rounded-2xl border flex items-center gap-3 {message.type === 'success' ? 'bg-green-50 border-green-100 text-green-800' : 'bg-red-50 border-red-100 text-red-800'}"
			>
				<svg class="h-5 w-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					{#if message.type === 'success'}
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
					{:else}
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					{/if}
				</svg>
				<span class="text-xs font-bold leading-tight">{message.text}</span>
			</div>
		{/if}

		<form class="space-y-6" onsubmit={handleResetPassword}>
			<div class="space-y-5">
				<div>
					<label for="new-password" class="block text-[13px] font-bold text-[#0a2e31] mb-2 px-1">
						Kata Sandi Baru <span class="text-teal-600">*</span>
					</label>
					<div class="relative group">
						<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-gray-300 group-focus-within:text-[#217b84] transition-colors">
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
							</svg>
						</div>
						<input
							id="new-password"
							type={showPassword ? 'text' : 'password'}
							required
							bind:value={newPassword}
							class="block w-full pl-11 pr-11 py-3.5 bg-gray-50/30 border border-gray-200 rounded-2xl text-sm transition-all focus:ring-4 focus:ring-teal-50/50 focus:border-[#217b84] focus:bg-white outline-none placeholder:text-gray-400"
							placeholder="Masukkan sandi baru"
						/>
						<button
							type="button"
							aria-label="Tampilkan/sembunyikan sandi"
							onclick={() => showPassword = !showPassword}
							class="absolute inset-y-0 right-0 pr-4 flex items-center text-gray-300 hover:text-gray-500"
						>
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
							</svg>
						</button>
					</div>
				</div>

				<div>
					<label for="confirm-password" class="block text-[13px] font-bold text-[#0a2e31] mb-2 px-1">
						Konfirmasi Kata Sandi <span class="text-teal-600">*</span>
					</label>
					<div class="relative group">
						<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-gray-300 group-focus-within:text-[#217b84] transition-colors">
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
							</svg>
						</div>
						<input
							id="confirm-password"
							type={showPassword ? 'text' : 'password'}
							required
							bind:value={confirmPassword}
							class="block w-full pl-11 pr-11 py-3.5 bg-gray-50/30 border border-gray-200 rounded-2xl text-sm transition-all focus:ring-4 focus:ring-teal-50/50 focus:border-[#217b84] focus:bg-white outline-none placeholder:text-gray-400"
							placeholder="Ulangi sandi baru"
						/>
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
				{loading ? 'Memproses...' : 'Simpan Kata Sandi Baru'}
			</button>
		</form>
	{/if}

	<div class="text-center">
		<a href="/login" class="text-sm font-bold text-[#0a2e31] hover:text-[#217b84] transition-colors">
			Kembali ke Login
		</a>
	</div>
</div>
