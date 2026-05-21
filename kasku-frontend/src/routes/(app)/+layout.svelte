<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { fly } from 'svelte/transition';
	import { auth } from '$lib/stores/auth.svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { apiFetch } from '$lib/api/client';
	import { initSyncTriggers, teardownSyncTriggers, triggerManualSync, syncStatus } from '$lib/sync';

	let { children } = $props();

	onMount(() => {
		// Root layout sudah handle silent refresh untuk cold-start.
		// (app) guard hanya enforce: jika setelah loading selesai tetap tidak ada token, kick ke login.
		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';
		if (isMock && !auth.accessToken) {
			auth.setToken('mock-jwt-token');
			auth.setUser({ id: 'mock-uid', email: 'demo@kasku.id', username: 'Juragan Demo' });
		}
		initSyncTriggers();
	});

	onDestroy(() => {
		teardownSyncTriggers();
	});

	$effect(() => {
		if (!auth.loading && !auth.isAuthenticated) {
			goto(resolve('/login'));
		}
	});

	async function handleLogout() {
		localStorage.removeItem('kasku_mock_mode');
		try {
			await apiFetch('/auth/logout', { method: 'POST' });
		} catch {
			// Ignore error on logout if BE is down
		} finally {
			auth.logout();
			goto(resolve('/login'));
		}
	}

	// Notification Drawer State
	let showNotifications = $state(false);
	let notifications = $state([
		{
			id: 1,
			title: 'Pembayaran Berhasil',
			message: 'Langganan Pro Anda telah aktif.',
			time: '2 menit lalu',
			read: false,
			type: 'success'
		},
		{
			id: 2,
			title: 'Aset Baru Tercatat',
			message: 'Anda baru saja menambah 10 unit Saham BBCA.',
			time: '1 jam lalu',
			read: false,
			type: 'info'
		},
		{
			id: 3,
			title: 'Sandi Diubah',
			message: 'Kata sandi akun Anda berhasil diperbarui.',
			time: 'Kemarin',
			read: true,
			type: 'warning'
		}
	]);

	const unreadCount = $derived(notifications.filter((n) => !n.read).length);

	function markAllRead() {
		notifications = notifications.map((n) => ({ ...n, read: true }));
	}

	function markRead(id: number) {
		notifications = notifications.map((n) => (n.id === id ? { ...n, read: true } : n));
	}

	const syncLoading = $derived(syncStatus.running);
	async function handleSync() {
		await triggerManualSync();
	}

	function isActive(path: string) {
		return $page.url.pathname === path || $page.url.pathname.startsWith(path + '/');
	}
</script>

{#if auth.loading}
	<div class="flex min-h-screen items-center justify-center bg-gray-50">
		<div class="flex flex-col items-center">
			<div class="h-10 w-10 animate-spin rounded-full border-b-2 border-indigo-600"></div>
			<p class="mt-4 animate-pulse text-sm font-bold tracking-widest text-gray-500 uppercase">
				KasKu
			</p>
		</div>
	</div>
{:else}
	{#if !syncStatus.online}
		<div
			class="fixed top-0 right-0 left-0 z-[80] flex items-center justify-center gap-2 bg-[#0a2e31] py-2 text-[11px] font-bold tracking-wider text-white uppercase"
			role="status"
		>
			<svg
				class="h-3.5 w-3.5"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				stroke-width="2.5"
				><path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="M18.364 5.636l-12.728 12.728m0-12.728l12.728 12.728"
				/></svg
			>
			<span>Mode Offline · Perubahan tersimpan, disinkronkan otomatis saat tersambung</span>
		</div>
	{/if}

	<div class="relative flex min-h-screen overflow-hidden bg-gray-50">
		<!-- Notification Drawer Overlay -->
		{#if showNotifications}
			<!-- svelte-ignore a11y_click_events_have_key_events -->
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div
				class="fixed inset-0 z-[60] bg-[#0a2e31]/20 backdrop-blur-sm transition-opacity"
				onclick={() => (showNotifications = false)}
			></div>
			<div
				class="fixed top-0 right-0 z-[70] flex h-full w-full max-w-sm flex-col bg-white shadow-2xl"
				transition:fly={{ x: 400, duration: 400 }}
			>
				<div class="flex items-center justify-between border-b border-gray-100 p-8">
					<div>
						<h2 class="text-xl font-black text-[#0a2e31]">Notifikasi</h2>
						<p class="mt-1 text-[10px] font-bold tracking-widest text-gray-400 uppercase">
							Pesan Terbaru Anda
						</p>
					</div>
					<button
						aria-label="Tutup notifikasi"
						onclick={() => (showNotifications = false)}
						class="p-2 text-gray-300 transition-colors hover:text-gray-600"
					>
						<svg
							class="h-6 w-6"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2.5"><path d="M6 18L18 6M6 6l12 12" /></svg
						>
					</button>
				</div>

				<div class="flex-1 space-y-4 overflow-y-auto p-6">
					{#each notifications as n (n.id)}
						<!-- svelte-ignore a11y_click_events_have_key_events -->
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<div
							class="group cursor-pointer rounded-[2rem] border p-5 transition-all {n.read
								? 'border-gray-100 bg-white opacity-60'
								: 'border-teal-100 bg-teal-50/30 shadow-sm'}"
							onclick={() => markRead(n.id)}
						>
							<div class="mb-2 flex items-start justify-between">
								<h3
									class="text-sm font-black text-[#0a2e31] transition-colors group-hover:text-teal-600"
								>
									{n.title}
								</h3>
								{#if !n.read}
									<div class="h-2 w-2 rounded-full bg-teal-500"></div>
								{/if}
							</div>
							<p class="mb-3 text-xs leading-relaxed font-medium text-gray-500">{n.message}</p>
							<p class="text-[10px] font-bold tracking-widest text-gray-300 uppercase">{n.time}</p>
						</div>
					{/each}
				</div>

				<div class="border-t border-gray-100 bg-gray-50 p-6">
					<button
						onclick={markAllRead}
						class="w-full rounded-2xl border border-gray-200 bg-white py-4 text-[11px] font-black tracking-widest text-[#0a2e31] uppercase shadow-sm transition-all hover:bg-[#0a2e31] hover:text-white"
					>
						Tandai Semua Dibaca
					</button>
				</div>
			</div>
		{/if}

		<!-- Simple Sidebar Placeholder -->
		<aside class="hidden w-64 flex-col bg-[#0a2e31] p-6 text-white lg:flex">
			<div class="mb-10 flex items-center justify-between">
				<div class="flex items-center gap-2">
					<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-[#217b84]">
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
					</div>
					<span class="text-xl font-black">KasKu</span>
				</div>
				<button
					onclick={() => (showNotifications = true)}
					class="relative p-2 text-white/60 transition-colors hover:text-white"
				>
					<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
						/></svg
					>
					{#if unreadCount > 0}
						<span
							class="absolute top-1.5 right-1.5 flex h-4 w-4 items-center justify-center rounded-full border-2 border-[#0a2e31] bg-teal-500 text-[9px] font-black text-white"
						>
							{unreadCount}
						</span>
					{/if}
				</button>
			</div>

			<nav class="space-y-1">
				<a
					href={resolve('/dashboard')}
					class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold transition-colors hover:bg-white/10"
				>
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
						/></svg
					>
					Dashboard
				</a>
				<a
					href={resolve('/accounts')}
					class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold transition-colors hover:bg-white/10"
				>
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M3 10h18M7 10V7a5 5 0 0110 0v3M4 10v10a1 1 0 001 1h14a1 1 0 001-1V10M10 14v4M14 14v4"
						/></svg
					>
					Rekening
				</a>
				<a
					href={resolve('/transactions')}
					class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold transition-colors hover:bg-white/10"
				>
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01m-.01 4h.01"
						/></svg
					>
					Transaksi
				</a>
				<a
					href={resolve('/budgets')}
					class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold transition-colors hover:bg-white/10"
				>
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 11h6m-3-8a9 9 0 100 18 9 9 0 000-18z"
						/></svg
					>
					Anggaran
				</a>
				<a
					href={resolve('/categories')}
					class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold transition-colors hover:bg-white/10"
				>
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M4 6h16M4 10h16M4 14h16M4 18h16"
						/></svg
					>
					Kategori
				</a>
				<a
					href={resolve('/investments')}
					class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold transition-colors hover:bg-white/10"
				>
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6"
						/></svg
					>
					Investasi
				</a>
				<a
					href={resolve('/profile')}
					class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold transition-colors hover:bg-white/10"
				>
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
						/></svg
					>
					Profil
				</a>
				<a
					href={resolve('/reports')}
					class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold transition-colors hover:bg-white/10"
				>
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M9 17v-2a4 4 0 00-4-4H5a2 2 0 00-2 2v2a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-2a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z"
						/><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z"
						/></svg
					>
					Laporan
				</a>
				<a
					href={resolve('/billing')}
					class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold transition-colors hover:bg-white/10"
				>
					<svg class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
						/></svg
					>
					Paket
				</a>
			</nav>

			<div class="mt-auto border-t border-white/10 pt-6">
				<div class="mb-4 flex items-center gap-3 px-2">
					<div
						class="relative flex h-10 w-10 items-center justify-center rounded-full border border-teal-500/50 bg-teal-500/20 font-black text-teal-400"
					>
						{auth.user?.username?.charAt(0).toUpperCase()}
						<button
							onclick={handleSync}
							disabled={syncLoading}
							class="absolute -right-1 -bottom-1 flex h-5 w-5 items-center justify-center rounded-full border border-gray-100 bg-white text-[#0a2e31] shadow-sm transition-colors hover:text-teal-600 disabled:opacity-50"
							title={syncStatus.error
								? `Sinkron gagal: ${syncStatus.error}`
								: syncLoading
									? 'Sinkronisasi berjalan…'
									: syncStatus.queuedCount > 0
										? `${syncStatus.queuedCount} perubahan menunggu sinkronisasi`
										: 'Sinkronisasi data offline'}
							aria-label="Sinkronisasi data"
						>
							<svg
								class="h-3 w-3 {syncLoading ? 'animate-spin' : ''}"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="3"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
								/>
							</svg>
						</button>
						{#if syncStatus.queuedCount > 0}
							<span
								class="absolute -top-1 -right-1 flex h-4 min-w-[1rem] items-center justify-center rounded-full border border-[#0a2e31] bg-orange-400 px-1 text-[9px] font-black text-[#0a2e31]"
								aria-label="{syncStatus.queuedCount} perubahan menunggu sync"
							>
								{syncStatus.queuedCount > 9 ? '9+' : syncStatus.queuedCount}
							</span>
						{:else if syncStatus.error}
							<span
								class="absolute -top-1 -right-1 h-3 w-3 rounded-full border border-[#0a2e31] bg-red-500"
								aria-label="Sinkron gagal"
							></span>
						{/if}
					</div>
					<div class="flex-1 overflow-hidden">
						<p class="truncate text-sm font-bold">{auth.user?.username}</p>
						<p class="truncate text-[10px] text-white/40">{auth.user?.email}</p>
					</div>
				</div>
				<button
					onclick={handleLogout}
					class="w-full rounded-xl border border-white/10 py-3 text-xs font-bold transition-all hover:border-red-500/20 hover:bg-red-500/10 hover:text-red-400"
				>
					Logout
				</button>
			</div>
		</aside>

		<div class="flex min-h-screen flex-1 flex-col">
			<!-- Header Mobile -->
			<header
				class="flex h-16 items-center justify-between border-b border-gray-100 bg-white px-6 lg:hidden"
			>
				<span class="font-black text-[#0a2e31]">KasKu</span>
				<div class="flex items-center gap-4">
					<button onclick={() => (showNotifications = true)} class="relative p-1 text-[#0a2e31]">
						<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"
							><path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
							/></svg
						>
						{#if unreadCount > 0}
							<span
								class="absolute -top-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full border-2 border-white bg-teal-500 text-[9px] font-black text-white"
							>
								{unreadCount}
							</span>
						{/if}
					</button>
					<button onclick={handleLogout} class="text-xs font-bold text-red-500">Logout</button>
				</div>
			</header>

			<main class="flex-1 overflow-y-auto p-6 pb-24 lg:p-12 lg:pb-12">
				<div class="mx-auto max-w-6xl">
					{@render children()}
				</div>
			</main>

			<!-- Bottom Navigation — mobile only -->
			<nav
				class="fixed right-0 bottom-0 left-0 z-50 border-t border-gray-100 bg-white lg:hidden"
				style="padding-bottom: env(safe-area-inset-bottom);"
			>
				<div class="flex items-stretch">
					<a
						href={resolve('/dashboard')}
						class="flex flex-1 flex-col items-center gap-1 py-3 text-[10px] font-bold tracking-wide transition-colors {isActive('/dashboard')
							? 'text-[#0a2e31]'
							: 'text-gray-400'}"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width={isActive('/dashboard') ? 2.5 : 1.8}>
							<path stroke-linecap="round" stroke-linejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
						</svg>
						<span>Dashboard</span>
						{#if isActive('/dashboard')}
							<span class="h-1 w-4 rounded-full bg-[#0a2e31]"></span>
						{:else}
							<span class="h-1 w-4"></span>
						{/if}
					</a>

					<a
						href={resolve('/transactions')}
						class="flex flex-1 flex-col items-center gap-1 py-3 text-[10px] font-bold tracking-wide transition-colors {isActive('/transactions')
							? 'text-[#0a2e31]'
							: 'text-gray-400'}"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width={isActive('/transactions') ? 2.5 : 1.8}>
							<path stroke-linecap="round" stroke-linejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01m-.01 4h.01" />
						</svg>
						<span>Transaksi</span>
						{#if isActive('/transactions')}
							<span class="h-1 w-4 rounded-full bg-[#0a2e31]"></span>
						{:else}
							<span class="h-1 w-4"></span>
						{/if}
					</a>

					<a
						href={resolve('/accounts')}
						class="flex flex-1 flex-col items-center gap-1 py-3 text-[10px] font-bold tracking-wide transition-colors {isActive('/accounts')
							? 'text-[#0a2e31]'
							: 'text-gray-400'}"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width={isActive('/accounts') ? 2.5 : 1.8}>
							<path stroke-linecap="round" stroke-linejoin="round" d="M3 10h18M7 10V7a5 5 0 0110 0v3M4 10v10a1 1 0 001 1h14a1 1 0 001-1V10M10 14v4M14 14v4" />
						</svg>
						<span>Rekening</span>
						{#if isActive('/accounts')}
							<span class="h-1 w-4 rounded-full bg-[#0a2e31]"></span>
						{:else}
							<span class="h-1 w-4"></span>
						{/if}
					</a>

					<a
						href={resolve('/investments')}
						class="flex flex-1 flex-col items-center gap-1 py-3 text-[10px] font-bold tracking-wide transition-colors {isActive('/investments')
							? 'text-[#0a2e31]'
							: 'text-gray-400'}"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width={isActive('/investments') ? 2.5 : 1.8}>
							<path stroke-linecap="round" stroke-linejoin="round" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
						</svg>
						<span>Investasi</span>
						{#if isActive('/investments')}
							<span class="h-1 w-4 rounded-full bg-[#0a2e31]"></span>
						{:else}
							<span class="h-1 w-4"></span>
						{/if}
					</a>

					<a
						href={resolve('/profile')}
						class="flex flex-1 flex-col items-center gap-1 py-3 text-[10px] font-bold tracking-wide transition-colors {isActive('/profile') || isActive('/categories') || isActive('/reports') || isActive('/billing') || isActive('/budgets')
							? 'text-[#0a2e31]'
							: 'text-gray-400'}"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width={isActive('/profile') ? 2.5 : 1.8}>
							<path stroke-linecap="round" stroke-linejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
						</svg>
						<span>Profil</span>
						{#if isActive('/profile') || isActive('/categories') || isActive('/reports') || isActive('/billing') || isActive('/budgets')}
							<span class="h-1 w-4 rounded-full bg-[#0a2e31]"></span>
						{:else}
							<span class="h-1 w-4"></span>
						{/if}
					</a>
				</div>
			</nav>
		</div>
	</div>
{/if}
