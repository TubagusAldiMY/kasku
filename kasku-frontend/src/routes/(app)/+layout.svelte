<script lang="ts">
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import { auth } from '$lib/stores/auth.svelte';
	import { goto } from '$app/navigation';
	import { apiFetch } from '$lib/api/client';

	let { children } = $props();

	onMount(async () => {
		// Jika dalam mode mock, isi data dummy jika kosong
		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';

		if (isMock) {
			if (!auth.accessToken) {
				auth.setToken('mock-jwt-token');
				auth.setUser({ id: 'mock-uid', email: 'demo@kasku.id', username: 'Juragan Demo' });
			}
			auth.setLoading(false);
			return;
		}

		if (!auth.accessToken) {
			try {
				const response = await apiFetch('/auth/refresh', {
					method: 'POST',
					skipAuth: true
				});

				const result = await response.json();

				if (result.success && result.data.access_token) {
					auth.setToken(result.data.access_token);
				} else {
					auth.logout();
					goto('/login');
				}
			} catch (err) {
				console.error('Initial refresh failed:', err);
				auth.logout();
				goto('/login');
			}
		} else {
			auth.setLoading(false);
		}
	});

	async function handleLogout() {
		localStorage.removeItem('kasku_mock_mode');
		try {
			await apiFetch('/auth/logout', { method: 'POST' });
		} catch (e) {
			// Ignore error on logout if BE is down
		} finally {
			auth.logout();
			goto('/login');
		}
	}

	// Notification Drawer State
	let showNotifications = $state(false);
	let notifications = $state([
		{ id: 1, title: 'Pembayaran Berhasil', message: 'Langganan Pro Anda telah aktif.', time: '2 menit lalu', read: false, type: 'success' },
		{ id: 2, title: 'Aset Baru Tercatat', message: 'Anda baru saja menambah 10 unit Saham BBCA.', time: '1 jam lalu', read: false, type: 'info' },
		{ id: 3, title: 'Sandi Diubah', message: 'Kata sandi akun Anda berhasil diperbarui.', time: 'Kemarin', read: true, type: 'warning' }
	]);

	const unreadCount = $derived(notifications.filter(n => !n.read).length);

	function markAllRead() {
		notifications = notifications.map(n => ({ ...n, read: true }));
	}

	function markRead(id: number) {
		notifications = notifications.map(n => n.id === id ? { ...n, read: true } : n);
	}

	// Offline Sync State
	let syncLoading = $state(false);
	async function handleSync() {
		syncLoading = true;
		// Mock delay for sync process
		await new Promise(resolve => setTimeout(resolve, 2000));
		syncLoading = false;
		// In a real scenario, this would call apiFetch('/sync/pull') and apiFetch('/sync/push')
	}
</script>

{#if auth.loading}
	<div class="flex min-h-screen items-center justify-center bg-gray-50">
		<div class="flex flex-col items-center">
			<div class="h-10 w-10 animate-spin rounded-full border-b-2 border-indigo-600"></div>
			<p class="mt-4 text-sm text-gray-500 font-bold uppercase tracking-widest animate-pulse">KasKu</p>
		</div>
	</div>
{:else}
	<div class="min-h-screen bg-gray-50 flex relative overflow-hidden">
		<!-- Notification Drawer Overlay -->
		{#if showNotifications}
			<!-- svelte-ignore a11y_click_events_have_key_events -->
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div 
				class="fixed inset-0 bg-[#0a2e31]/20 backdrop-blur-sm z-[60] transition-opacity"
				onclick={() => showNotifications = false}
			></div>
			<div 
				class="fixed right-0 top-0 h-full w-full max-w-sm bg-white shadow-2xl z-[70] flex flex-col"
				transition:fly={{ x: 400, duration: 400 }}
			>
				<div class="p-8 border-b border-gray-100 flex justify-between items-center">
					<div>
						<h2 class="text-xl font-black text-[#0a2e31]">Notifikasi</h2>
						<p class="text-[10px] font-bold text-gray-400 uppercase tracking-widest mt-1">Pesan Terbaru Anda</p>
					</div>
					<button aria-label="Tutup notifikasi" onclick={() => showNotifications = false} class="p-2 text-gray-300 hover:text-gray-600 transition-colors">
						<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path d="M6 18L18 6M6 6l12 12" /></svg>
					</button>
				</div>

				<div class="flex-1 overflow-y-auto p-6 space-y-4">
					{#each notifications as n}
						<!-- svelte-ignore a11y_click_events_have_key_events -->
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<div 
							class="p-5 rounded-[2rem] border transition-all cursor-pointer group {n.read ? 'bg-white border-gray-100 opacity-60' : 'bg-teal-50/30 border-teal-100 shadow-sm'}"
							onclick={() => markRead(n.id)}
						>
							<div class="flex justify-between items-start mb-2">
								<h3 class="text-sm font-black text-[#0a2e31] group-hover:text-teal-600 transition-colors">{n.title}</h3>
								{#if !n.read}
									<div class="h-2 w-2 rounded-full bg-teal-500"></div>
								{/if}
							</div>
							<p class="text-xs text-gray-500 font-medium mb-3 leading-relaxed">{n.message}</p>
							<p class="text-[10px] font-bold text-gray-300 uppercase tracking-widest">{n.time}</p>
						</div>
					{/each}
				</div>

				<div class="p-6 bg-gray-50 border-t border-gray-100">
					<button 
						onclick={markAllRead}
						class="w-full py-4 bg-white border border-gray-200 text-[#0a2e31] text-[11px] font-black uppercase tracking-widest rounded-2xl hover:bg-[#0a2e31] hover:text-white transition-all shadow-sm"
					>
						Tandai Semua Dibaca
					</button>
				</div>
			</div>
		{/if}

		<!-- Simple Sidebar Placeholder -->
		<aside class="hidden lg:flex w-64 flex-col bg-[#0a2e31] text-white p-6">
			<div class="flex items-center justify-between mb-10">
				<div class="flex items-center gap-2">
					<div class="h-8 w-8 rounded-lg bg-[#217b84] flex items-center justify-center">
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
					</div>
					<span class="text-xl font-black">KasKu</span>
				</div>
				<button 
					onclick={() => showNotifications = true}
					class="relative p-2 text-white/60 hover:text-white transition-colors"
				>
					<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" /></svg>
					{#if unreadCount > 0}
						<span class="absolute top-1.5 right-1.5 h-4 w-4 bg-teal-500 text-white text-[9px] font-black rounded-full flex items-center justify-center border-2 border-[#0a2e31]">
							{unreadCount}
						</span>
					{/if}
				</button>
			</div>
			
			<nav class="space-y-1">
				<a href="/dashboard" class="flex items-center gap-3 px-4 py-3 rounded-xl hover:bg-white/10 transition-colors font-bold text-sm">
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" /></svg>
					Dashboard
				</a>
				<a href="/accounts" class="flex items-center gap-3 px-4 py-3 rounded-xl hover:bg-white/10 transition-colors font-bold text-sm">
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h18M7 10V7a5 5 0 0110 0v3M4 10v10a1 1 0 001 1h14a1 1 0 001-1V10M10 14v4M14 14v4" /></svg>
					Rekening
				</a>
				<a href="/transactions" class="flex items-center gap-3 px-4 py-3 rounded-xl hover:bg-white/10 transition-colors font-bold text-sm">
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01m-.01 4h.01" /></svg>
					Transaksi
				</a>
				<a href="/categories" class="flex items-center gap-3 px-4 py-3 rounded-xl hover:bg-white/10 transition-colors font-bold text-sm">
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16" /></svg>
					Kategori
				</a>
				<a href="/investments" class="flex items-center gap-3 px-4 py-3 rounded-xl hover:bg-white/10 transition-colors font-bold text-sm">
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" /></svg>
					Investasi
				</a>
				<a href="/profile" class="flex items-center gap-3 px-4 py-3 rounded-xl hover:bg-white/10 transition-colors font-bold text-sm">
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" /></svg>
					Profil
				</a>
				<a href="/reports" class="flex items-center gap-3 px-4 py-3 rounded-xl hover:bg-white/10 transition-colors font-bold text-sm">
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17v-2a4 4 0 00-4-4H5a2 2 0 00-2 2v2a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-2a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" /></svg>
					Laporan
				</a>
				<a href="/billing" class="flex items-center gap-3 px-4 py-3 rounded-xl hover:bg-white/10 transition-colors font-bold text-sm">
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" /></svg>
					Paket
				</a>
				</nav>


			<div class="mt-auto pt-6 border-t border-white/10">
				<div class="flex items-center gap-3 px-2 mb-4">
					<div class="h-10 w-10 rounded-full bg-teal-500/20 border border-teal-500/50 flex items-center justify-center text-teal-400 font-black relative">
						{auth.user?.username?.charAt(0).toUpperCase()}
						<button 
							onclick={handleSync}
							disabled={syncLoading}
							class="absolute -bottom-1 -right-1 h-5 w-5 bg-white rounded-full shadow-sm border border-gray-100 flex items-center justify-center text-[#0a2e31] hover:text-teal-600 transition-colors disabled:opacity-50"
							title="Sinkronisasi Data Offline"
						>
							<svg class="h-3 w-3 {syncLoading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
								<path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
							</svg>
						</button>
					</div>
					<div class="flex-1 overflow-hidden">
						<p class="text-sm font-bold truncate">{auth.user?.username}</p>
						<p class="text-[10px] text-white/40 truncate">{auth.user?.email}</p>
					</div>
				</div>
				<button 
					onclick={handleLogout}
					class="w-full py-3 rounded-xl border border-white/10 hover:bg-red-500/10 hover:text-red-400 hover:border-red-500/20 transition-all font-bold text-xs"
				>
					Logout
				</button>
			</div>
		</aside>

		<div class="flex-1 flex flex-col min-h-screen">
			<!-- Header Mobile -->
			<header class="lg:hidden h-16 bg-white border-b border-gray-100 flex items-center justify-between px-6">
				<span class="font-black text-[#0a2e31]">KasKu</span>
				<div class="flex items-center gap-4">
					<button 
						onclick={() => showNotifications = true}
						class="relative p-1 text-[#0a2e31]"
					>
						<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" /></svg>
						{#if unreadCount > 0}
							<span class="absolute -top-1 -right-1 h-4 w-4 bg-teal-500 text-white text-[9px] font-black rounded-full flex items-center justify-center border-2 border-white">
								{unreadCount}
							</span>
						{/if}
					</button>
					<button onclick={handleLogout} class="text-xs font-bold text-red-500">Logout</button>
				</div>
			</header>

			<main class="flex-1 overflow-y-auto p-6 lg:p-12">
				<div class="max-w-6xl mx-auto">
					{@render children()}
				</div>
			</main>
		</div>
	</div>
{/if}
