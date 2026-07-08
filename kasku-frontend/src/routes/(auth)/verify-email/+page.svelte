<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { apiFetch } from '$lib/api/client';
	import { resolve } from '$app/paths';

	let status = $state<'loading' | 'success' | 'error'>('loading');
	let message = $state('Sedang memverifikasi email Anda...');
	let email = $state('');
	let resendLoading = $state(false);
	let resendMessage = $state<string | null>(null);

	async function handleResendVerification() {
		if (!email) {
			resendMessage = 'Silakan masukkan email Anda.';
			return;
		}

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

	onMount(async () => {
		const token = page.url.searchParams.get('token');

		if (!token) {
			status = 'error';
			message = 'Token verifikasi tidak ditemukan.';
			return;
		}

		try {
			const response = await apiFetch(`/auth/verify-email?token=${token}`, {
				method: 'POST',
				skipAuth: true
			});

			const result = await response.json();

			if (result.success) {
				status = 'success';
				message = 'Email Anda berhasil diverifikasi! Silakan login.';
			} else {
				status = 'error';
				message = result.error?.message || 'Gagal memverifikasi email.';
			}
		} catch (err) {
			status = 'error';
			message = 'Terjadi kesalahan koneksi.';
			console.error(err);
		}
	});
</script>

<div class="space-y-8">
	<div>
		<h1 class="font-serif text-[34px] leading-tight tracking-tight text-ink">Verifikasi email</h1>
	</div>

	{#if status === 'loading'}
		<div class="flex flex-col items-center gap-4 py-6 text-center">
			<span class="h-10 w-10 animate-spin rounded-full border-2 border-teal/25 border-t-teal"
			></span>
			<p class="text-sm text-ink/60">{message}</p>
		</div>
	{:else if status === 'success'}
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
			<p class="text-sm text-ink/60">{message}</p>
			<a
				href={resolve('/login')}
				class="flex w-full items-center justify-center gap-2.5 rounded-full bg-teal py-3.5 text-sm font-semibold text-card transition-colors hover:bg-ink"
			>
				Lanjut ke masuk
			</a>
		</div>
	{:else}
		<div class="space-y-6">
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
				<p class="text-[13px] font-medium text-clay">{message}</p>
			</div>

			<div class="space-y-4 border-t border-ink/10 pt-6">
				<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/50 uppercase">
					Kirim ulang tautan?
				</p>
				<input
					type="email"
					bind:value={email}
					placeholder="Masukkan email Anda"
					class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-teal"
				/>
				<button
					onclick={handleResendVerification}
					disabled={resendLoading}
					class="flex w-full items-center justify-center gap-2.5 rounded-full bg-teal py-3.5 text-sm font-semibold text-card transition-colors hover:bg-ink disabled:opacity-70"
				>
					{#if resendLoading}
						<span class="h-4 w-4 animate-spin rounded-full border-2 border-card/40 border-t-card"
						></span>
					{/if}
					{resendLoading ? 'Mengirim…' : 'Kirim ulang verifikasi'}
				</button>
				{#if resendMessage}
					<p
						class="text-center text-[12px] font-medium {resendMessage.includes('berhasil')
							? 'text-teal'
							: 'text-clay'}"
					>
						{resendMessage}
					</p>
				{/if}
			</div>

			<p class="text-center text-[13px] text-ink/55">
				<a href={resolve('/login')} class="font-semibold text-teal hover:text-ink"
					>Ke halaman masuk</a
				>
				<span class="mx-2 text-ink/25">·</span>
				<a href={resolve('/register')} class="font-semibold text-teal hover:text-ink"
					>Kembali ke daftar</a
				>
			</p>
		</div>
	{/if}
</div>
