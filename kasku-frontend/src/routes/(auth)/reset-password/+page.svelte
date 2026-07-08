<script lang="ts">
	import { apiFetch } from '$lib/api/client';
	import { page } from '$app/state';
	import { fly } from 'svelte/transition';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	let newPassword = $state('');
	let confirmPassword = $state('');
	let loading = $state(false);
	let showPassword = $state(false);
	let message = $state<{ type: 'success' | 'error'; text: string } | null>(null);

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
				body: JSON.stringify({ token, new_password: newPassword }),
				skipAuth: true
			});

			const result = await response.json();

			if (result.success) {
				message = {
					type: 'success',
					text: 'Kata sandi berhasil diperbarui! Mengalihkan ke halaman login...'
				};
				setTimeout(() => {
					goto(resolve('/login'));
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

<div class="space-y-8">
	<div>
		<h1 class="font-serif text-[34px] leading-tight tracking-tight text-ink">Atur ulang sandi</h1>
		<p class="mt-2.5 text-sm text-ink/60">Silakan masukkan kata sandi baru untuk akunmu.</p>
	</div>

	{#if !token}
		<div class="space-y-4 rounded-xl border border-clay/25 bg-clay/5 p-6 text-center">
			<svg
				class="mx-auto h-10 w-10 text-clay"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				stroke-width="1.8"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
				/>
			</svg>
			<p class="text-[13px] font-medium text-clay">Tautan tidak valid atau sudah kadaluarsa.</p>
			<a
				href={resolve('/forgot-password')}
				class="inline-block text-[12px] font-semibold tracking-[0.12em] text-teal uppercase hover:text-ink"
			>
				Minta tautan baru
			</a>
		</div>
	{:else}
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

		<form class="space-y-5" onsubmit={handleResetPassword}>
			<div>
				<label
					for="new-password"
					class="mb-2 block text-[11px] font-semibold tracking-[0.12em] text-ink/50 uppercase"
				>
					Kata sandi baru
				</label>
				<div class="relative">
					<input
						id="new-password"
						type={showPassword ? 'text' : 'password'}
						required
						bind:value={newPassword}
						placeholder="Masukkan sandi baru"
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
						type={showPassword ? 'text' : 'password'}
						required
						bind:value={confirmPassword}
						placeholder="Ulangi sandi baru"
						class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-teal"
					/>
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
				{loading ? 'Memproses…' : 'Simpan kata sandi baru'}
			</button>
		</form>
	{/if}

	<p class="text-center text-[13px] text-ink/55">
		<a href={resolve('/login')} class="font-semibold text-teal hover:text-ink">Kembali ke masuk</a>
	</p>
</div>
