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

<div class="space-y-8">
	<div>
		<h1 class="font-serif text-[34px] leading-tight tracking-tight text-ink">Lupa kata sandi?</h1>
		<p class="mt-2.5 text-sm text-ink/60">
			Masukkan emailmu untuk menerima tautan pemulihan kata sandi.
		</p>
	</div>

	{#if message}
		<div
			in:fly={{ y: -10, duration: 400 }}
			class="flex items-start gap-2.5 rounded-xl border p-4 {message.type === 'success'
				? 'border-teal/25 bg-teal/5'
				: 'border-clay/25 bg-clay/5'}"
		>
			<svg
				class="mt-px h-4 w-4 shrink-0 {message.type === 'success' ? 'text-teal' : 'text-clay'}"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				stroke-width="2"
			>
				{#if message.type === 'success'}
					<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
				{:else}
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				{/if}
			</svg>
			<span
				class="text-[13px] font-medium {message.type === 'success' ? 'text-teal' : 'text-clay'}"
			>
				{message.text}
			</span>
		</div>
	{/if}

	<form class="space-y-5" onsubmit={handleForgotPassword}>
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

		<button
			type="submit"
			disabled={loading}
			class="flex w-full items-center justify-center gap-2.5 rounded-full bg-teal py-3.5 text-sm font-semibold text-card transition-colors hover:bg-ink disabled:opacity-70"
		>
			{#if loading}
				<span class="h-4 w-4 animate-spin rounded-full border-2 border-card/40 border-t-card"
				></span>
			{/if}
			{loading ? 'Memproses…' : 'Kirim tautan pemulihan'}
		</button>
	</form>

	<p class="text-center text-[13px] text-ink/55">
		Ingat kata sandimu?
		<a href={resolve('/login')} class="font-semibold text-teal hover:text-ink">Kembali ke masuk</a>
	</p>
</div>
