<script lang="ts">
	import { apiFetch } from '$lib/api/client';
	import { fly } from 'svelte/transition';
	import { resolve } from '$app/paths';

	let email = $state('');
	let loading = $state(false);
	let message = $state<{ type: 'success' | 'error'; text: string } | null>(null);

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

<div class="animate-in fade-in slide-in-from-bottom-4 space-y-10 duration-700">
	<div class="space-y-3 text-center">
		<h2 class="text-3xl font-bold tracking-tight text-[#0a2e31]">Lupa Kata Sandi?</h2>
		<p class="mx-auto max-w-[320px] text-[15px] leading-relaxed text-gray-500">
			Masukkan email Anda untuk menerima tautan pemulihan kata sandi.
		</p>
	</div>

	{#if message}
		<div
			in:fly={{ y: -10, duration: 400 }}
			class="flex items-center gap-3 rounded-2xl border p-4 {message.type === 'success'
				? 'border-green-100 bg-green-50 text-green-800'
				: 'border-red-100 bg-red-50 text-red-800'}"
		>
			<svg class="h-5 w-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				{#if message.type === 'success'}
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2.5"
						d="M5 13l4 4L19 7"
					/>
				{:else}
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2.5"
						d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				{/if}
			</svg>
			<span class="text-xs leading-tight font-bold">{message.text}</span>
		</div>
	{/if}

	<form class="space-y-6" onsubmit={handleForgotPassword}>
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
			{loading ? 'Memproses...' : 'Kirim Tautan Pemulihan'}
		</button>
	</form>

	<div class="text-center">
		<a
			href={resolve('/login')}
			class="inline-flex items-center gap-2 text-sm font-bold text-[#0a2e31] transition-colors hover:text-[#217b84]"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2.5"
					d="M10 19l-7-7m0 0l7-7m-7 7h18"
				/>
			</svg>
			Kembali ke Login
		</a>
	</div>
</div>
