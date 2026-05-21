<script lang="ts">
	import { auth } from '$lib/stores/auth.svelte';
	import { fly } from 'svelte/transition';
	import { apiFetch } from '$lib/api/client';
	import { onMount } from 'svelte';

	let loading = $state(false);
	let message = $state<{ type: 'success' | 'error'; text: string } | null>(null);

	// User info state
	let profile = $state({
		username: auth.user?.username || '',
		email: auth.user?.email || '',
		displayName: ''
	});

	// Password change state
	let passwordData = $state({
		current: '',
		new: '',
		confirm: ''
	});

	// Notification preferences state
	let notificationPrefs = $state({
		email_enabled: true,
		payment_alerts_enabled: true,
		expiry_alerts_enabled: true
	});

	let prefLoading = $state(false);

	async function fetchPreferences() {
		try {
			const res = await apiFetch('/notifications/preferences');
			const result = await res.json();
			if (result.success && result.data) {
				notificationPrefs = {
					email_enabled: result.data.email_enabled,
					payment_alerts_enabled: result.data.payment_alerts_enabled,
					expiry_alerts_enabled: result.data.expiry_alerts_enabled
				};
			}
		} catch (err) {
			console.error('Gagal memuat preferensi:', err);
		}
	}

	async function fetchUserProfile() {
		try {
			const res = await apiFetch('/users/profile');
			const result = await res.json();
			if (result.success && result.data) {
				profile.username = result.data.username || auth.user?.username || '';
				profile.email = result.data.email || auth.user?.email || '';
				profile.displayName = result.data.display_name || '';

				// Update store
				auth.setUser({
					...auth.user!,
					username: profile.username,
					email: profile.email
				});
			}
		} catch (err) {
			console.error('Gagal memuat profil user:', err);
		}
	}

	async function updatePreferences() {
		prefLoading = true;
		message = null;
		try {
			const res = await apiFetch('/notifications/preferences', {
				method: 'PUT',
				body: JSON.stringify(notificationPrefs)
			});
			const result = await res.json();
			if (result.success) {
				message = { type: 'success', text: 'Preferensi notifikasi berhasil diperbarui!' };
			} else {
				message = { type: 'error', text: result.error?.message || 'Gagal memperbarui preferensi.' };
			}
		} catch {
			message = { type: 'error', text: 'Gagal menghubungi server.' };
		} finally {
			prefLoading = false;
		}
	}

	onMount(() => {
		fetchPreferences();
		fetchUserProfile();
	});

	async function updateProfile(e: SubmitEvent) {
		e.preventDefault();

		const trimmedUsername = profile.username.trim();
		if (trimmedUsername && !/^[a-zA-Z0-9_]{3,50}$/.test(trimmedUsername)) {
			message = { type: 'error', text: 'Username hanya boleh huruf, angka, dan underscore (3–50 karakter).' };
			return;
		}

		loading = true;
		message = null;

		try {
			const res = await apiFetch('/users/profile', {
				method: 'PUT',
				body: JSON.stringify({
					username: trimmedUsername,
					display_name: profile.displayName.trim()
				})
			});
			const result = await res.json();

			if (result.success) {
				auth.setUser({
					...auth.user!,
					username: profile.username,
					email: profile.email
				});
				message = { type: 'success', text: 'Profil berhasil diperbarui!' };
			} else {
				message = { type: 'error', text: result.error?.message || 'Gagal memperbarui profil.' };
			}
		} catch {
			message = { type: 'error', text: 'Gagal memperbarui profil.' };
		} finally {
			loading = false;
		}
	}

	async function changePassword(e: SubmitEvent) {
		e.preventDefault();
		if (passwordData.new !== passwordData.confirm) {
			message = { type: 'error', text: 'Konfirmasi katasandi tidak cocok.' };
			return;
		}
		if (passwordData.new.length < 8) {
			message = { type: 'error', text: 'Katasandi baru minimal 8 karakter.' };
			return;
		}

		loading = true;
		message = null;

		try {
			const res = await apiFetch('/auth/change-password', {
				method: 'PUT',
				body: JSON.stringify({
					current_password: passwordData.current,
					new_password: passwordData.new
				})
			});
			const result = await res.json();

			if (result.success) {
				message = { type: 'success', text: 'Katasandi berhasil diubah!' };
				passwordData = { current: '', new: '', confirm: '' };
			} else {
				message = { type: 'error', text: result.error?.message || 'Gagal mengubah katasandi.' };
			}
		} catch {
			message = { type: 'error', text: 'Gagal menghubungi server.' };
		} finally {
			loading = false;
		}
	}
</script>

<div class="animate-in fade-in mx-auto max-w-4xl space-y-10 pb-20 duration-700">
	<div class="space-y-1">
		<h1 class="text-3xl font-black text-[#0a2e31]">Pengaturan Profil</h1>
		<p class="font-medium text-gray-500">Kelola informasi akun dan keamanan Anda.</p>
	</div>

	{#if message}
		<div
			in:fly={{ y: -10, duration: 400 }}
			class="flex items-center gap-3 rounded-2xl border p-4 {message.type === 'success'
				? 'border-green-100 bg-green-50 text-green-800'
				: 'border-red-100 bg-red-50 text-red-800'}"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
			<span class="text-sm font-bold">{message.text}</span>
		</div>
	{/if}

	<div class="grid grid-cols-1 gap-10 lg:grid-cols-3">
		<!-- Sidebar Info -->
		<div class="space-y-6">
			<div class="rounded-[2.5rem] border border-gray-100 bg-white p-8 text-center shadow-sm">
				<div
					class="mx-auto mb-4 flex h-24 w-24 items-center justify-center rounded-full border-4 border-white bg-teal-500/10 text-4xl font-black text-teal-600 shadow-xl"
				>
					{auth.user?.username?.charAt(0).toUpperCase()}
				</div>
				<h2 class="text-xl font-black text-[#0a2e31]">{auth.user?.username}</h2>
				<p class="mt-1 text-xs font-bold tracking-widest text-gray-400 uppercase">
					{auth.user?.email}
				</p>

				<div class="mt-8 flex flex-col gap-2 border-t border-gray-50 pt-6">
					<div class="flex items-center justify-between px-2 text-xs">
						<span class="font-bold text-gray-400 uppercase">Tier</span>
						<span class="rounded-full bg-teal-600 px-3 py-1 font-black text-white">FREE</span>
					</div>
					<div class="flex items-center justify-between px-2 text-xs">
						<span class="font-bold text-gray-400 uppercase">Bergabung</span>
						<span class="font-bold text-[#0a2e31]">Mei 2026</span>
					</div>
				</div>
			</div>

			<div
				class="group relative overflow-hidden rounded-[2.5rem] bg-[#0a2e31] p-8 text-white shadow-xl"
			>
				<div
					class="absolute -right-4 -bottom-4 h-24 w-24 rounded-full bg-white/5 transition-transform duration-700 group-hover:scale-150"
				></div>
				<h3 class="relative z-10 mb-2 text-lg font-bold">Upgrade ke Pro</h3>
				<p class="relative z-10 mb-6 text-xs text-white/60">
					Dapatkan kuota transaksi tak terbatas dan laporan PDF premium.
				</p>
				<button
					class="relative z-10 w-full rounded-xl bg-[#217b84] py-3 text-xs font-black tracking-widest uppercase transition-all hover:bg-[#1a5f66]"
					>Lihat Paket</button
				>
			</div>
		</div>

		<!-- Forms -->
		<div class="space-y-8 lg:col-span-2">
			<!-- Account Form -->
			<div class="space-y-8 rounded-[2.5rem] border border-gray-100 bg-white p-10 shadow-sm">
				<h3 class="text-xl font-black text-[#0a2e31]">Informasi Akun</h3>

				<form onsubmit={updateProfile} class="space-y-6">
					<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
						<div class="space-y-2">
							<label
								for="username"
								class="block px-1 text-[11px] font-bold tracking-widest text-gray-400 uppercase"
								>Username</label
							>
							<input
								id="username"
								type="text"
								bind:value={profile.username}
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-medium text-[#0a2e31] transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
							/>
						</div>
						<div class="space-y-2">
							<label
								for="email"
								class="block px-1 text-[11px] font-bold tracking-widest text-gray-400 uppercase"
								>Email</label
							>
							<input
								id="email"
								type="email"
								bind:value={profile.email}
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-medium text-[#0a2e31] transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
							/>
						</div>
					</div>

					<div class="pt-4">
						<button
							type="submit"
							disabled={loading}
							class="rounded-2xl bg-[#0a2e31] px-8 py-3.5 text-xs font-black tracking-widest text-white uppercase shadow-lg transition-all hover:bg-black active:scale-[0.98] disabled:opacity-50"
						>
							{loading ? 'Menyimpan...' : 'Simpan Perubahan'}
						</button>
					</div>
				</form>
			</div>

			<!-- Password Form -->
			<div class="space-y-8 rounded-[2.5rem] border border-gray-100 bg-white p-10 shadow-sm">
				<h3 class="text-xl font-black text-[#0a2e31]">Keamanan</h3>

				<form onsubmit={changePassword} class="space-y-6">
					<div class="space-y-2">
						<label
							for="current-pass"
							class="block px-1 text-[11px] font-bold tracking-widest text-gray-400 uppercase"
							>Katasandi Saat Ini</label
						>
						<input
							id="current-pass"
							type="password"
							bind:value={passwordData.current}
							class="w-full rounded-2xl border border-gray-200 bg-gray-50 px-5 py-3.5 transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
							placeholder="••••••••"
						/>
					</div>

					<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
						<div class="space-y-2">
							<label
								for="new-pass"
								class="block px-1 text-[11px] font-bold tracking-widest text-gray-400 uppercase"
								>Katasandi Baru</label
							>
							<input
								id="new-pass"
								type="password"
								bind:value={passwordData.new}
								class="w-full rounded-2xl border border-gray-200 bg-gray-50 px-5 py-3.5 transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
								placeholder="••••••••"
							/>
						</div>
						<div class="space-y-2">
							<label
								for="confirm-pass"
								class="block px-1 text-[11px] font-bold tracking-widest text-gray-400 uppercase"
								>Konfirmasi Baru</label
							>
							<input
								id="confirm-pass"
								type="password"
								bind:value={passwordData.confirm}
								class="w-full rounded-2xl border border-gray-200 bg-gray-50 px-5 py-3.5 transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
								placeholder="••••••••"
							/>
						</div>
					</div>

					<div class="pt-4 text-right">
						<button
							type="submit"
							disabled={loading || !passwordData.new}
							class="rounded-2xl border-2 border-gray-100 bg-white px-8 py-3.5 text-xs font-black tracking-widest text-[#0a2e31] uppercase transition-all hover:border-teal-500 hover:text-teal-600 active:scale-[0.98] disabled:opacity-50"
						>
							Ganti Katasandi
						</button>
					</div>
				</form>
			</div>

			<!-- Notification Preferences Form -->
			<div class="space-y-8 rounded-[2.5rem] border border-gray-100 bg-white p-10 shadow-sm">
				<div class="flex items-center justify-between">
					<h3 class="text-xl font-black text-[#0a2e31]">Pengaturan Notifikasi</h3>
					<div
						class="flex h-10 w-10 items-center justify-center rounded-2xl bg-teal-50 text-teal-600"
					>
						<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
							/>
						</svg>
					</div>
				</div>

				<div class="space-y-6">
					<div class="flex items-center justify-between rounded-2xl bg-gray-50 p-4">
						<div class="space-y-0.5">
							<p class="text-sm font-black text-[#0a2e31]">Notifikasi Email</p>
							<p class="text-[11px] font-bold tracking-tight text-gray-400 uppercase">
								Kirim ringkasan dan berita ke email Anda
							</p>
						</div>
						<label class="relative inline-flex cursor-pointer items-center">
							<input
								type="checkbox"
								bind:checked={notificationPrefs.email_enabled}
								class="peer sr-only"
							/>
							<div
								class="peer h-6 w-11 rounded-full bg-gray-200 peer-checked:bg-teal-600 peer-focus:outline-none after:absolute after:top-[2px] after:left-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:after:translate-x-full peer-checked:after:border-white"
							></div>
						</label>
					</div>

					<div class="flex items-center justify-between rounded-2xl bg-gray-50 p-4">
						<div class="space-y-0.5">
							<p class="text-sm font-black text-[#0a2e31]">Peringatan Pembayaran</p>
							<p class="text-[11px] font-bold tracking-tight text-gray-400 uppercase">
								Beritahu saat tagihan paket berhasil dibayar
							</p>
						</div>
						<label class="relative inline-flex cursor-pointer items-center">
							<input
								type="checkbox"
								bind:checked={notificationPrefs.payment_alerts_enabled}
								class="peer sr-only"
							/>
							<div
								class="peer h-6 w-11 rounded-full bg-gray-200 peer-checked:bg-teal-600 peer-focus:outline-none after:absolute after:top-[2px] after:left-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:after:translate-x-full peer-checked:after:border-white"
							></div>
						</label>
					</div>

					<div class="flex items-center justify-between rounded-2xl bg-gray-50 p-4">
						<div class="space-y-0.5">
							<p class="text-sm font-black text-[#0a2e31]">Peringatan Kedaluwarsa</p>
							<p class="text-[11px] font-bold tracking-tight text-gray-400 uppercase">
								Ingatkan sebelum paket langganan habis
							</p>
						</div>
						<label class="relative inline-flex cursor-pointer items-center">
							<input
								type="checkbox"
								bind:checked={notificationPrefs.expiry_alerts_enabled}
								class="peer sr-only"
							/>
							<div
								class="peer h-6 w-11 rounded-full bg-gray-200 peer-checked:bg-teal-600 peer-focus:outline-none after:absolute after:top-[2px] after:left-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:after:translate-x-full peer-checked:after:border-white"
							></div>
						</label>
					</div>

					<div class="pt-4">
						<button
							onclick={updatePreferences}
							disabled={prefLoading}
							class="rounded-2xl bg-[#0a2e31] px-8 py-3.5 text-xs font-black tracking-widest text-white uppercase shadow-lg transition-all hover:bg-black active:scale-[0.98] disabled:opacity-50"
						>
							{prefLoading ? 'Menyimpan...' : 'Simpan Preferensi'}
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
