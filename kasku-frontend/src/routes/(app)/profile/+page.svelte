<script lang="ts">
	import { auth } from '$lib/stores/auth.svelte';
	import { fly } from 'svelte/transition';
	import { apiFetch } from '$lib/api/client';
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';

	let loading = $state(false);
	let message = $state<{ type: 'success' | 'error'; text: string } | null>(null);

	// User info state
	let profile = $state({
		username: auth.user?.username || '',
		email: auth.user?.email || '',
		displayName: '',
		createdAt: ''
	});

	// Subscription state
	let subscriptionPlanName = $state<string>('—');
	let subscriptionStatus = $state<string>('');
	let subscriptionLoading = $state(true);

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
				profile.createdAt = result.data.created_at || '';

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

	async function fetchSubscription() {
		subscriptionLoading = true;
		try {
			const [plansRes, subRes] = await Promise.all([
				apiFetch('/billing/plans'),
				apiFetch('/billing/subscription')
			]);
			const plansData = await plansRes.json();
			const subData = await subRes.json();

			if (subData.success && subData.data) {
				subscriptionStatus = subData.data.status ?? '';
				const planId: string = subData.data.plan_id ?? '';
				if (plansData.success && Array.isArray(plansData.data)) {
					const matched = (plansData.data as { id: string; name: string }[]).find(
						(p) => p.id === planId
					);
					subscriptionPlanName = matched?.name ?? planId;
				} else {
					subscriptionPlanName = planId;
				}
			} else {
				subscriptionPlanName = 'Gratis';
				subscriptionStatus = 'FREE';
			}
		} catch {
			subscriptionPlanName = '—';
		} finally {
			subscriptionLoading = false;
		}
	}

	function formatJoinDate(iso: string): string {
		if (!iso) return '—';
		try {
			return new Date(iso).toLocaleDateString('id-ID', { month: 'long', year: 'numeric' });
		} catch {
			return '—';
		}
	}

	const tierBadgeClass = $derived(() => {
		const name = subscriptionPlanName.toLowerCase();
		if (name.includes('ultimate')) return 'border-gold/30 text-gold';
		if (name.includes('pro') || name.includes('premium')) return 'border-teal/30 text-teal';
		return 'border-ink/15 text-ink/60';
	});

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
		fetchSubscription();
	});

	async function updateProfile(e: SubmitEvent) {
		e.preventDefault();

		const trimmedUsername = profile.username.trim();
		if (trimmedUsername && !/^[a-zA-Z0-9_]{3,50}$/.test(trimmedUsername)) {
			message = {
				type: 'error',
				text: 'Username hanya boleh huruf, angka, dan underscore (3–50 karakter).'
			};
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

<div class="animate-in fade-in mx-auto max-w-4xl pb-4 duration-700">
	<div class="border-b border-ink/10 pb-8">
		<h1 class="font-serif text-4xl tracking-tight text-ink">Pengaturan Profil</h1>
		<p class="mt-2 text-sm text-ink/60">Kelola informasi akun dan keamanan Anda.</p>
	</div>

	{#if message}
		<div
			in:fly={{ y: -10, duration: 400 }}
			class="mt-6 flex items-center gap-3 rounded-xl border px-4 py-3 {message.type === 'success'
				? 'border-teal/25 bg-teal/5 text-teal'
				: 'border-clay/25 bg-clay/5 text-clay'}"
		>
			<svg class="h-5 w-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				{#if message.type === 'success'}
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M5 13l4 4L19 7"
					/>
				{:else}
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				{/if}
			</svg>
			<span class="text-sm font-medium">{message.text}</span>
		</div>
	{/if}

	<div class="mt-8 grid grid-cols-1 gap-8 lg:grid-cols-3">
		<!-- Sidebar Info -->
		<div class="space-y-6">
			<div class="rounded-2xl border border-ink/10 bg-card p-8 text-center">
				<div
					class="mx-auto mb-4 flex h-24 w-24 items-center justify-center rounded-full border border-teal/25 bg-teal/5 font-serif text-4xl text-teal"
				>
					{auth.user?.username?.charAt(0).toUpperCase()}
				</div>
				<h2 class="font-serif text-2xl text-ink">{auth.user?.username}</h2>
				<p class="mt-1 text-xs text-ink/45">
					{auth.user?.email}
				</p>

				<div class="mt-8 flex flex-col gap-3 border-t border-ink/10 pt-6 text-left">
					<div class="flex items-center justify-between text-sm">
						<span class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
							>Paket</span
						>
						{#if subscriptionLoading}
							<span class="h-5 w-16 animate-pulse rounded-full bg-ink/10"></span>
						{:else}
							<span
								class="rounded-full border px-2 py-px text-[12px] font-semibold {tierBadgeClass()}"
							>
								{subscriptionPlanName}
							</span>
						{/if}
					</div>
					{#if subscriptionStatus && subscriptionStatus !== 'FREE'}
						<div class="flex items-center justify-between text-sm">
							<span class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>Status</span
							>
							<span
								class="font-semibold {subscriptionStatus === 'ACTIVE' ? 'text-teal' : 'text-gold'}"
							>
								{subscriptionStatus === 'ACTIVE' ? 'Aktif' : subscriptionStatus}
							</span>
						</div>
					{/if}
					<div class="flex items-center justify-between text-sm">
						<span class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
							>Bergabung</span
						>
						<span class="font-medium text-ink">{formatJoinDate(profile.createdAt)}</span>
					</div>
				</div>
			</div>

			<div class="rounded-2xl bg-ink p-8 text-card">
				<h3 class="font-serif text-2xl">Upgrade ke Pro</h3>
				<p class="mt-2 mb-6 text-sm text-mint/80">
					Dapatkan kuota transaksi tak terbatas dan laporan PDF premium.
				</p>
				<a
					href={resolve('/billing')}
					class="block w-full rounded-full bg-teal py-3 text-center text-[13px] font-semibold text-card transition-colors hover:bg-mint hover:text-ink"
					>Lihat Paket</a
				>
			</div>
		</div>

		<!-- Forms -->
		<div class="space-y-8 lg:col-span-2">
			<!-- Account Form -->
			<div class="space-y-6 rounded-2xl border border-ink/10 bg-card p-8 sm:p-10">
				<h3 class="font-serif text-2xl text-ink">Informasi Akun</h3>

				<form onsubmit={updateProfile} class="space-y-6">
					<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
						<div class="space-y-2">
							<label
								for="username"
								class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>Username</label
							>
							<input
								id="username"
								type="text"
								bind:value={profile.username}
								class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-ink transition-colors outline-none focus:border-teal"
							/>
						</div>
						<div class="space-y-2">
							<label
								for="email"
								class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>Email</label
							>
							<input
								id="email"
								type="email"
								bind:value={profile.email}
								class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-ink transition-colors outline-none focus:border-teal"
							/>
						</div>
					</div>

					<div class="pt-2">
						<button
							type="submit"
							disabled={loading}
							class="rounded-full bg-teal px-6 py-3 text-[13px] font-semibold text-card transition-colors hover:bg-ink disabled:opacity-50"
						>
							{loading ? 'Menyimpan...' : 'Simpan Perubahan'}
						</button>
					</div>
				</form>
			</div>

			<!-- Password Form -->
			<div class="space-y-6 rounded-2xl border border-ink/10 bg-card p-8 sm:p-10">
				<h3 class="font-serif text-2xl text-ink">Keamanan</h3>

				<form onsubmit={changePassword} class="space-y-6">
					<div class="space-y-2">
						<label
							for="current-pass"
							class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
							>Katasandi Saat Ini</label
						>
						<input
							id="current-pass"
							type="password"
							bind:value={passwordData.current}
							class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-ink transition-colors outline-none focus:border-teal"
							placeholder="••••••••"
						/>
					</div>

					<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
						<div class="space-y-2">
							<label
								for="new-pass"
								class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>Katasandi Baru</label
							>
							<input
								id="new-pass"
								type="password"
								bind:value={passwordData.new}
								class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-ink transition-colors outline-none focus:border-teal"
								placeholder="••••••••"
							/>
						</div>
						<div class="space-y-2">
							<label
								for="confirm-pass"
								class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>Konfirmasi Baru</label
							>
							<input
								id="confirm-pass"
								type="password"
								bind:value={passwordData.confirm}
								class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-ink transition-colors outline-none focus:border-teal"
								placeholder="••••••••"
							/>
						</div>
					</div>

					<div class="pt-2 text-right">
						<button
							type="submit"
							disabled={loading || !passwordData.new}
							class="rounded-full border border-ink/15 px-6 py-3 text-[13px] font-semibold text-ink transition-colors hover:border-teal/40 hover:text-teal disabled:opacity-50"
						>
							Ganti Katasandi
						</button>
					</div>
				</form>
			</div>

			<!-- Notification Preferences Form -->
			<div class="space-y-6 rounded-2xl border border-ink/10 bg-card p-8 sm:p-10">
				<div class="flex items-center justify-between">
					<h3 class="font-serif text-2xl text-ink">Pengaturan Notifikasi</h3>
					<div
						class="flex h-10 w-10 items-center justify-center rounded-full border border-teal/25 text-teal"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="1.8"
								d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
							/>
						</svg>
					</div>
				</div>

				<div>
					<div class="flex items-center justify-between border-t border-ink/8 py-4">
						<div class="space-y-0.5">
							<p class="text-sm font-medium text-ink">Notifikasi Email</p>
							<p class="text-[12px] text-ink/50">Kirim ringkasan dan berita ke email Anda</p>
						</div>
						<label class="relative inline-flex cursor-pointer items-center">
							<input
								type="checkbox"
								bind:checked={notificationPrefs.email_enabled}
								class="peer sr-only"
							/>
							<div
								class="peer h-6 w-11 rounded-full bg-ink/15 peer-checked:bg-teal peer-focus:outline-none after:absolute after:top-[2px] after:left-[2px] after:h-5 after:w-5 after:rounded-full after:bg-card after:transition-all after:content-[''] peer-checked:after:translate-x-full"
							></div>
						</label>
					</div>

					<div class="flex items-center justify-between border-t border-ink/8 py-4">
						<div class="space-y-0.5">
							<p class="text-sm font-medium text-ink">Peringatan Pembayaran</p>
							<p class="text-[12px] text-ink/50">Beritahu saat tagihan paket berhasil dibayar</p>
						</div>
						<label class="relative inline-flex cursor-pointer items-center">
							<input
								type="checkbox"
								bind:checked={notificationPrefs.payment_alerts_enabled}
								class="peer sr-only"
							/>
							<div
								class="peer h-6 w-11 rounded-full bg-ink/15 peer-checked:bg-teal peer-focus:outline-none after:absolute after:top-[2px] after:left-[2px] after:h-5 after:w-5 after:rounded-full after:bg-card after:transition-all after:content-[''] peer-checked:after:translate-x-full"
							></div>
						</label>
					</div>

					<div class="flex items-center justify-between border-t border-b border-ink/8 py-4">
						<div class="space-y-0.5">
							<p class="text-sm font-medium text-ink">Peringatan Kedaluwarsa</p>
							<p class="text-[12px] text-ink/50">Ingatkan sebelum paket langganan habis</p>
						</div>
						<label class="relative inline-flex cursor-pointer items-center">
							<input
								type="checkbox"
								bind:checked={notificationPrefs.expiry_alerts_enabled}
								class="peer sr-only"
							/>
							<div
								class="peer h-6 w-11 rounded-full bg-ink/15 peer-checked:bg-teal peer-focus:outline-none after:absolute after:top-[2px] after:left-[2px] after:h-5 after:w-5 after:rounded-full after:bg-card after:transition-all after:content-[''] peer-checked:after:translate-x-full"
							></div>
						</label>
					</div>

					<div class="pt-6">
						<button
							onclick={updatePreferences}
							disabled={prefLoading}
							class="rounded-full bg-teal px-6 py-3 text-[13px] font-semibold text-card transition-colors hover:bg-ink disabled:opacity-50"
						>
							{prefLoading ? 'Menyimpan...' : 'Simpan Preferensi'}
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
