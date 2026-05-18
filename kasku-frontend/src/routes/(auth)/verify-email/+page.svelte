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

<div class="flex min-h-screen items-center justify-center bg-gray-50 px-4 py-12 sm:px-6 lg:px-8">
	<div class="w-full max-w-md space-y-8 text-center">
		<div>
			<h2 class="mt-6 text-3xl font-extrabold text-gray-900">Verifikasi Email</h2>
		</div>

		<div class="mt-8">
			{#if status === 'loading'}
				<div class="flex flex-col items-center">
					<div
						class="h-12 w-12 animate-spin rounded-full border-t-2 border-b-2 border-indigo-600"
					></div>
					<p class="mt-4 text-gray-600">{message}</p>
				</div>
			{:else if status === 'success'}
				<div class="rounded-md bg-green-50 p-4">
					<div class="flex flex-col items-center">
						<svg class="h-12 w-12 text-green-400" viewBox="0 0 20 20" fill="currentColor">
							<path
								fill-rule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
								clip-rule="evenodd"
							/>
						</svg>
						<p class="mt-4 text-sm font-medium text-green-800">{message}</p>
						<div class="mt-6">
							<a
								href={resolve('/login')}
								class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
							>
								Lanjut ke Login
							</a>
						</div>
					</div>
				</div>
			{:else}
				<div class="rounded-md bg-red-50 p-4">
					<div class="flex flex-col items-center gap-4">
						<svg class="h-12 w-12 text-red-400" viewBox="0 0 20 20" fill="currentColor">
							<path
								fill-rule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
								clip-rule="evenodd"
							/>
						</svg>
						<p class="text-sm font-medium text-red-800">{message}</p>

						<div class="w-full space-y-3 border-t border-red-100 pt-4">
							<p class="text-xs font-bold tracking-wider text-[#0a2e31] uppercase">
								Kirim ulang tautan?
							</p>
							<input
								type="email"
								bind:value={email}
								placeholder="Masukkan email Anda"
								class="w-full rounded-xl border border-gray-200 px-4 py-2 text-sm outline-none focus:ring-2 focus:ring-indigo-500"
							/>
							<button
								onclick={handleResendVerification}
								disabled={resendLoading}
								class="w-full rounded-xl bg-indigo-600 py-2 text-sm font-bold text-white transition-all hover:bg-indigo-700 disabled:opacity-50"
							>
								{resendLoading ? 'Mengirim...' : 'Kirim Ulang Verifikasi'}
							</button>
							{#if resendMessage}
								<p
									class="text-[11px] font-bold {resendMessage.includes('berhasil')
										? 'text-green-600'
										: 'text-red-600'}"
								>
									{resendMessage}
								</p>
							{/if}
						</div>

						<div class="mt-2 flex gap-4">
							<a href={resolve('/login')} class="text-sm font-bold text-indigo-600 hover:underline">
								Ke Halaman Login
							</a>
							<a
								href={resolve('/register')}
								class="text-sm font-bold text-indigo-600 hover:underline"
							>
								Kembali ke Daftar
							</a>
						</div>
					</div>
				</div>
			{/if}
		</div>
	</div>
</div>
