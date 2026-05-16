<script lang="ts">
	import { auth } from '$lib/stores/auth.svelte';
	import { fade, fly } from 'svelte/transition';
	import { apiFetch } from '$lib/api/client';

	let loading = $state(false);
	let message = $state<{ type: 'success' | 'error', text: string } | null>(null);

	// User info state
	let profile = $state({
		username: auth.user?.username || '',
		email: auth.user?.email || ''
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
		} catch (err) {
			message = { type: 'error', text: 'Gagal menghubungi server.' };
		} finally {
			prefLoading = false;
		}
	}

	import { onMount } from 'svelte';
	onMount(fetchPreferences);

	async function updateProfile(e: SubmitEvent) {
		e.preventDefault();
		loading = true;
		message = null;

		try {
			// Mock update for now
			setTimeout(() => {
				auth.setUser({
					...auth.user!,
					username: profile.username,
					email: profile.email
				});
				message = { type: 'success', text: 'Profil berhasil diperbarui!' };
				loading = false;
			}, 1000);
		} catch (err) {
			message = { type: 'error', text: 'Gagal memperbarui profil.' };
			loading = false;
		}
	}

	async function changePassword(e: SubmitEvent) {
		e.preventDefault();
		if (passwordData.new !== passwordData.confirm) {
			message = { type: 'error', text: 'Konfirmasi katasandi tidak cocok.' };
			return;
		}

		loading = true;
		message = null;

		try {
			// Mock change password
			setTimeout(() => {
				message = { type: 'success', text: 'Katasandi berhasil diubah!' };
				passwordData = { current: '', new: '', confirm: '' };
				loading = false;
			}, 1000);
		} catch (err) {
			message = { type: 'error', text: 'Gagal mengubah katasandi.' };
			loading = false;
		}
	}
</script>

<div class="max-w-4xl mx-auto space-y-10 animate-in fade-in duration-700 pb-20">
	<div class="space-y-1">
		<h1 class="text-3xl font-black text-[#0a2e31]">Pengaturan Profil</h1>
		<p class="text-gray-500 font-medium">Kelola informasi akun dan keamanan Anda.</p>
	</div>

	{#if message}
		<div 
			in:fly={{ y: -10, duration: 400 }}
			class="p-4 rounded-2xl border flex items-center gap-3 {message.type === 'success' ? 'bg-green-50 border-green-100 text-green-800' : 'bg-red-50 border-red-100 text-red-800'}"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				{#if message.type === 'success'}
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
				{:else}
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
				{/if}
			</svg>
			<span class="text-sm font-bold">{message.text}</span>
		</div>
	{/if}

	<div class="grid grid-cols-1 lg:grid-cols-3 gap-10">
		<!-- Sidebar Info -->
		<div class="space-y-6">
			<div class="bg-white p-8 rounded-[2.5rem] border border-gray-100 shadow-sm text-center">
				<div class="h-24 w-24 rounded-full bg-teal-500/10 border-4 border-white shadow-xl mx-auto flex items-center justify-center text-teal-600 text-4xl font-black mb-4">
					{auth.user?.username?.charAt(0).toUpperCase()}
				</div>
				<h2 class="text-xl font-black text-[#0a2e31]">{auth.user?.username}</h2>
				<p class="text-xs font-bold text-gray-400 uppercase tracking-widest mt-1">{auth.user?.email}</p>
				
				<div class="mt-8 pt-6 border-t border-gray-50 flex flex-col gap-2">
					<div class="flex justify-between items-center text-xs px-2">
						<span class="font-bold text-gray-400 uppercase">Tier</span>
						<span class="px-3 py-1 bg-teal-600 text-white rounded-full font-black">FREE</span>
					</div>
					<div class="flex justify-between items-center text-xs px-2">
						<span class="font-bold text-gray-400 uppercase">Bergabung</span>
						<span class="font-bold text-[#0a2e31]">Mei 2026</span>
					</div>
				</div>
			</div>

			<div class="bg-[#0a2e31] p-8 rounded-[2.5rem] text-white shadow-xl relative overflow-hidden group">
				<div class="absolute -right-4 -bottom-4 h-24 w-24 bg-white/5 rounded-full group-hover:scale-150 transition-transform duration-700"></div>
				<h3 class="text-lg font-bold mb-2 relative z-10">Upgrade ke Pro</h3>
				<p class="text-xs text-white/60 mb-6 relative z-10">Dapatkan kuota transaksi tak terbatas dan laporan PDF premium.</p>
				<button class="w-full py-3 bg-[#217b84] hover:bg-[#1a5f66] rounded-xl text-xs font-black tracking-widest uppercase transition-all relative z-10">Lihat Paket</button>
			</div>
		</div>

		<!-- Forms -->
		<div class="lg:col-span-2 space-y-8">
			<!-- Account Form -->
			<div class="bg-white p-10 rounded-[2.5rem] border border-gray-100 shadow-sm space-y-8">
				<h3 class="text-xl font-black text-[#0a2e31]">Informasi Akun</h3>
				
				<form onsubmit={updateProfile} class="space-y-6">
					<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
						<div class="space-y-2">
							<label for="username" class="block text-[11px] font-bold text-gray-400 uppercase tracking-widest px-1">Username</label>
							<input 
								id="username"
								type="text" 
								bind:value={profile.username}
								class="w-full px-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 focus:border-[#217b84] outline-none transition-all font-medium text-[#0a2e31]"
							/>
						</div>
						<div class="space-y-2">
							<label for="email" class="block text-[11px] font-bold text-gray-400 uppercase tracking-widest px-1">Email</label>
							<input 
								id="email"
								type="email" 
								bind:value={profile.email}
								class="w-full px-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 focus:border-[#217b84] outline-none transition-all font-medium text-[#0a2e31]"
							/>
						</div>
					</div>
					
					<div class="pt-4">
						<button 
							type="submit" 
							disabled={loading}
							class="px-8 py-3.5 bg-[#0a2e31] hover:bg-black text-white text-xs font-black uppercase tracking-widest rounded-2xl shadow-lg transition-all active:scale-[0.98] disabled:opacity-50"
						>
							{loading ? 'Menyimpan...' : 'Simpan Perubahan'}
						</button>
					</div>
				</form>
			</div>

			<!-- Password Form -->
			<div class="bg-white p-10 rounded-[2.5rem] border border-gray-100 shadow-sm space-y-8">
				<h3 class="text-xl font-black text-[#0a2e31]">Keamanan</h3>
				
				<form onsubmit={changePassword} class="space-y-6">
					<div class="space-y-2">
						<label for="current-pass" class="block text-[11px] font-bold text-gray-400 uppercase tracking-widest px-1">Katasandi Saat Ini</label>
						<input 
							id="current-pass"
							type="password" 
							bind:value={passwordData.current}
							class="w-full px-5 py-3.5 bg-gray-50 border border-gray-200 rounded-2xl focus:ring-4 focus:ring-teal-50 focus:border-[#217b84] outline-none transition-all"
							placeholder="••••••••"
						/>
					</div>

					<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
						<div class="space-y-2">
							<label for="new-pass" class="block text-[11px] font-bold text-gray-400 uppercase tracking-widest px-1">Katasandi Baru</label>
							<input 
								id="new-pass"
								type="password" 
								bind:value={passwordData.new}
								class="w-full px-5 py-3.5 bg-gray-50 border border-gray-200 rounded-2xl focus:ring-4 focus:ring-teal-50 focus:border-[#217b84] outline-none transition-all"
								placeholder="••••••••"
							/>
						</div>
						<div class="space-y-2">
							<label for="confirm-pass" class="block text-[11px] font-bold text-gray-400 uppercase tracking-widest px-1">Konfirmasi Baru</label>
							<input 
								id="confirm-pass"
								type="password" 
								bind:value={passwordData.confirm}
								class="w-full px-5 py-3.5 bg-gray-50 border border-gray-200 rounded-2xl focus:ring-4 focus:ring-teal-50 focus:border-[#217b84] outline-none transition-all"
								placeholder="••••••••"
							/>
						</div>
					</div>
					
					<div class="pt-4 text-right">
						<button 
							type="submit" 
							disabled={loading || !passwordData.new}
							class="px-8 py-3.5 bg-white border-2 border-gray-100 hover:border-teal-500 hover:text-teal-600 text-[#0a2e31] text-xs font-black uppercase tracking-widest rounded-2xl transition-all active:scale-[0.98] disabled:opacity-50"
						>
							Ganti Katasandi
						</button>
					</div>
				</form>
			</div>

			<!-- Notification Preferences Form -->
			<div class="bg-white p-10 rounded-[2.5rem] border border-gray-100 shadow-sm space-y-8">
				<div class="flex justify-between items-center">
					<h3 class="text-xl font-black text-[#0a2e31]">Pengaturan Notifikasi</h3>
					<div class="h-10 w-10 bg-teal-50 rounded-2xl flex items-center justify-center text-teal-600">
						<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
						</svg>
					</div>
				</div>
				
				<div class="space-y-6">
					<div class="flex items-center justify-between p-4 bg-gray-50 rounded-2xl">
						<div class="space-y-0.5">
							<p class="text-sm font-black text-[#0a2e31]">Notifikasi Email</p>
							<p class="text-[11px] text-gray-400 font-bold uppercase tracking-tight">Kirim ringkasan dan berita ke email Anda</p>
						</div>
						<label class="relative inline-flex items-center cursor-pointer">
							<input type="checkbox" bind:checked={notificationPrefs.email_enabled} class="sr-only peer">
							<div class="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-teal-600"></div>
						</label>
					</div>

					<div class="flex items-center justify-between p-4 bg-gray-50 rounded-2xl">
						<div class="space-y-0.5">
							<p class="text-sm font-black text-[#0a2e31]">Peringatan Pembayaran</p>
							<p class="text-[11px] text-gray-400 font-bold uppercase tracking-tight">Beritahu saat tagihan paket berhasil dibayar</p>
						</div>
						<label class="relative inline-flex items-center cursor-pointer">
							<input type="checkbox" bind:checked={notificationPrefs.payment_alerts_enabled} class="sr-only peer">
							<div class="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-teal-600"></div>
						</label>
					</div>

					<div class="flex items-center justify-between p-4 bg-gray-50 rounded-2xl">
						<div class="space-y-0.5">
							<p class="text-sm font-black text-[#0a2e31]">Peringatan Kedaluwarsa</p>
							<p class="text-[11px] text-gray-400 font-bold uppercase tracking-tight">Ingatkan sebelum paket langganan habis</p>
						</div>
						<label class="relative inline-flex items-center cursor-pointer">
							<input type="checkbox" bind:checked={notificationPrefs.expiry_alerts_enabled} class="sr-only peer">
							<div class="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-teal-600"></div>
						</label>
					</div>
					
					<div class="pt-4">
						<button 
							onclick={updatePreferences}
							disabled={prefLoading}
							class="px-8 py-3.5 bg-[#0a2e31] hover:bg-black text-white text-xs font-black uppercase tracking-widest rounded-2xl shadow-lg transition-all active:scale-[0.98] disabled:opacity-50"
						>
							{prefLoading ? 'Menyimpan...' : 'Simpan Preferensi'}
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
