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

<div class="animate-in fade-in slide-in-from-bottom-4 space-y-10 duration-700">
	<div class="space-y-3 text-center">
		<h2 class="text-3xl font-bold tracking-tight text-[#0a2e31]">Atur Ulang Sandi</h2>
		<p class="mx-auto max-w-[320px] text-[15px] leading-relaxed text-gray-500">
			Silakan masukkan kata sandi baru untuk akun Anda.
		</p>
	</div>

	{#if !token}
		<div class="space-y-4 rounded-[2rem] border border-red-100 bg-red-50 p-6 text-center">
			<svg
				class="mx-auto h-12 w-12 text-red-400"
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
			<p class="text-sm font-bold text-red-800">Tautan tidak valid atau sudah kadaluarsa.</p>
			<a
				href={resolve('/forgot-password')}
				class="inline-block text-xs font-black tracking-widest text-[#217b84] uppercase hover:underline"
			>
				Minta Tautan Baru
			</a>
		</div>
	{:else}
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

		<form class="space-y-6" onsubmit={handleResetPassword}>
			<div class="space-y-5">
				<div>
					<label for="new-password" class="mb-2 block px-1 text-[13px] font-bold text-[#0a2e31]">
						Kata Sandi Baru <span class="text-teal-600">*</span>
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
									d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
								/>
							</svg>
						</div>
						<input
							id="new-password"
							type={showPassword ? 'text' : 'password'}
							required
							bind:value={newPassword}
							class="block w-full rounded-2xl border border-gray-200 bg-gray-50/30 py-3.5 pr-11 pl-11 text-sm transition-all outline-none placeholder:text-gray-400 focus:border-[#217b84] focus:bg-white focus:ring-4 focus:ring-teal-50/50"
							placeholder="Masukkan sandi baru"
						/>
						<button
							type="button"
							aria-label="Tampilkan/sembunyikan sandi"
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

				<div>
					<label
						for="confirm-password"
						class="mb-2 block px-1 text-[13px] font-bold text-[#0a2e31]"
					>
						Konfirmasi Kata Sandi <span class="text-teal-600">*</span>
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
									d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
								/>
							</svg>
						</div>
						<input
							id="confirm-password"
							type={showPassword ? 'text' : 'password'}
							required
							bind:value={confirmPassword}
							class="block w-full rounded-2xl border border-gray-200 bg-gray-50/30 py-3.5 pr-11 pl-11 text-sm transition-all outline-none placeholder:text-gray-400 focus:border-[#217b84] focus:bg-white focus:ring-4 focus:ring-teal-50/50"
							placeholder="Ulangi sandi baru"
						/>
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
				{loading ? 'Memproses...' : 'Simpan Kata Sandi Baru'}
			</button>
		</form>
	{/if}

	<div class="text-center">
		<a
			href={resolve('/login')}
			class="text-sm font-bold text-[#0a2e31] transition-colors hover:text-[#217b84]"
		>
			Kembali ke Login
		</a>
	</div>
</div>
