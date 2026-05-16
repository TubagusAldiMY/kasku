<script lang="ts">
	import { apiFetch } from '$lib/api/client';
	import { fade, fly } from 'svelte/transition';

	let email = $state('');
	let loading = $state(false);
	let message = $state<{ type: 'success' | 'error', text: string } | null>(null);

	async function handleForgotPassword(e: SubmitEvent) {
		e.preventDefault();
		loading = true;
		message = null;

		try {
			const response = await apiFetch('/auth/forgot-password', {
				method: 'POST',
				body: JSON.stringify({ email }),
				skipAuth: true
			});

			const result = await response.json();

			if (result.success) {
				message = { 
					type: 'success', 
					text: 'Tautan pemulihan telah dikirim ke email Anda. Silakan periksa kotak masuk (dan folder spam).' 
				};
				email = '';
			} else {
				message = { 
					type: 'error', 
					text: result.error?.message || 'Terjadi kesalahan saat memproses permintaan.' 
				};
			}
		} catch (err) {
			message = { 
				type: 'error', 
				text: 'Gagal menghubungi server. Pastikan koneksi internet Anda aktif.' 
			};
			console.error(err);
		} finally {
			loading = false;
		}
	}
</script>

<div class="space-y-10 animate-in fade-in slide-in-from-bottom-4 duration-700">
	<div class="space-y-3 text-center">
		<h2 class="text-3xl font-bold tracking-tight text-[#0a2e31]">Lupa Kata Sandi?</h2>
		<p class="text-[15px] text-gray-500 leading-relaxed max-w-[320px] mx-auto">
			Masukkan email Anda untuk menerima tautan pemulihan kata sandi.
		</p>
	</div>

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

	<form class="space-y-6" onsubmit={handleForgotPassword}>
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

		<button
			type="submit"
			disabled={loading}
			class="w-full py-4 bg-[#217b84] hover:bg-[#1a5f66] text-white text-[15px] font-bold rounded-2xl shadow-xl shadow-teal-900/10 transition-all active:scale-[0.98] disabled:opacity-70 disabled:active:scale-100 flex items-center justify-center gap-3"
		>
			{#if loading}
				<div class="h-5 w-5 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
			{/if}
			{loading ? 'Memproses...' : 'Kirim Tautan Pemulihan'}
		</button>
	</form>

	<div class="text-center">
		<a href="/login" class="text-sm font-bold text-[#0a2e31] hover:text-[#217b84] transition-colors inline-flex items-center gap-2">
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
			</svg>
			Kembali ke Login
		</a>
	</div>
</div>
