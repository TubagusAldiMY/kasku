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
	let showMenu = $state(false);
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

	// Desktop primary nav (mockup: 6 editorial links)
	const topNav = [
		{ href: '/dashboard', label: 'Ringkasan' },
		{ href: '/transactions', label: 'Transaksi' },
		{ href: '/accounts', label: 'Rekening' },
		{ href: '/budgets', label: 'Anggaran' },
		{ href: '/investments', label: 'Investasi' },
		{ href: '/reports', label: 'Laporan' }
	] as const;

	// Avatar dropdown — full secondary set, so every page stays reachable on mobile too.
	const menuLinks = [
		{ href: '/budgets', label: 'Anggaran' },
		{ href: '/reports', label: 'Laporan' },
		{ href: '/categories', label: 'Kategori' },
		{ href: '/debts', label: 'Hutang & Piutang' },
		{ href: '/billing', label: 'Paket' },
		{ href: '/profile', label: 'Profil' }
	] as const;

	// Mobile bottom nav (mockup: 5 tabs; Profil is the catch-all hub).
	const bottomNav = [
		{
			href: '/dashboard',
			label: 'Beranda',
			icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6'
		},
		{
			href: '/transactions',
			label: 'Transaksi',
			icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01m-.01 4h.01'
		},
		{
			href: '/accounts',
			label: 'Rekening',
			icon: 'M3 10h18M7 10V7a5 5 0 0110 0v3M4 10v10a1 1 0 001 1h14a1 1 0 001-1V10M10 14v4M14 14v4'
		},
		{
			href: '/investments',
			label: 'Investasi',
			icon: 'M13 7h8m0 0v8m0-8l-8 8-4-4-6 6'
		},
		{
			href: '/profile',
			label: 'Profil',
			icon: 'M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z',
			match: ['/profile', '/categories', '/reports', '/billing', '/budgets', '/debts']
		}
	] as const;

	function bottomActive(item: { href: string; match?: readonly string[] }) {
		return (item.match ?? [item.href]).some((p) => isActive(p));
	}
</script>

{#if auth.loading}
	<div class="flex min-h-screen items-center justify-center bg-paper">
		<div class="flex flex-col items-center gap-4">
			<div class="h-10 w-10 animate-spin rounded-full border-2 border-ink/15 border-t-teal"></div>
			<p class="font-serif text-2xl text-ink">Kas<em class="text-teal">Ku</em></p>
		</div>
	</div>
{:else}
	<div class="flex min-h-screen flex-col">
		{#if !syncStatus.online}
			<div
				class="flex items-center justify-center gap-2 bg-ink py-2 text-[11px] font-semibold tracking-wider text-mint uppercase"
				role="status"
			>
				<svg
					class="h-3.5 w-3.5"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2.5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M18.364 5.636l-12.728 12.728m0-12.728l12.728 12.728"
					/>
				</svg>
				<span>Mode offline · perubahan tersimpan, tersinkron otomatis saat tersambung</span>
			</div>
		{/if}

		<!-- ═══════════ Header — editorial top nav ═══════════ -->
		<header class="sticky top-0 z-40 border-b border-ink/10 bg-paper/90 backdrop-blur-sm">
			<div class="mx-auto flex max-w-6xl items-center justify-between px-5 py-4 sm:px-6 lg:px-10">
				<div class="flex items-center gap-6 lg:gap-10">
					<a
						href={resolve('/dashboard')}
						class="font-serif text-[26px] leading-none tracking-tight text-ink"
					>
						Kas<em class="text-teal">Ku</em>
					</a>
					<nav class="hidden items-center gap-7 text-[13.5px] font-medium lg:flex">
						{#each topNav as item (item.href)}
							<a
								href={resolve(item.href)}
								class="pb-1 transition-colors {isActive(item.href)
									? 'border-b-2 border-teal text-ink'
									: 'text-ink/55 hover:text-ink'}"
							>
								{item.label}
							</a>
						{/each}
					</nav>
				</div>

				<div class="flex items-center gap-2 sm:gap-3">
					<a
						href={resolve('/transactions')}
						class="hidden rounded-full bg-teal px-5 py-2 text-[13px] font-semibold text-card transition-colors hover:bg-ink sm:inline-flex"
					>
						+ Catat
					</a>

					<button
						onclick={() => (showNotifications = true)}
						class="relative rounded-full p-2 text-ink/60 transition-colors hover:bg-ink/5 hover:text-ink"
						aria-label="Notifikasi"
					>
						<svg
							class="h-5 w-5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.8"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
							/>
						</svg>
						{#if unreadCount > 0}
							<span
								class="absolute top-1 right-1 flex h-4 w-4 items-center justify-center rounded-full border-2 border-paper bg-teal text-[9px] font-semibold text-card"
							>
								{unreadCount}
							</span>
						{/if}
					</button>

					<!-- Avatar + dropdown -->
					<div class="relative">
						<button
							onclick={() => (showMenu = !showMenu)}
							class="flex h-9 w-9 items-center justify-center rounded-full bg-teal text-[13px] font-semibold text-card transition-transform hover:scale-105"
							aria-label="Menu akun"
							aria-expanded={showMenu}
						>
							{auth.user?.username?.charAt(0).toUpperCase() ?? 'J'}
							{#if syncStatus.queuedCount > 0}
								<span
									class="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border border-paper bg-gold"
									aria-hidden="true"
								></span>
							{:else if syncStatus.error}
								<span
									class="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border border-paper bg-clay"
									aria-hidden="true"
								></span>
							{/if}
						</button>

						{#if showMenu}
							<!-- svelte-ignore a11y_click_events_have_key_events -->
							<!-- svelte-ignore a11y_no_static_element_interactions -->
							<div class="fixed inset-0 z-40" onclick={() => (showMenu = false)}></div>
							<div
								class="absolute right-0 z-50 mt-2 w-64 overflow-hidden rounded-2xl border border-ink/10 bg-card shadow-xl"
								transition:fly={{ y: -6, duration: 160 }}
							>
								<div class="border-b border-ink/10 px-5 py-4">
									<p class="truncate text-sm font-semibold text-ink">
										{auth.user?.username ?? 'Juragan'}
									</p>
									<p class="truncate text-xs text-ink/50">{auth.user?.email ?? ''}</p>
								</div>
								<nav class="py-1.5">
									{#each menuLinks as item (item.href)}
										<a
											href={resolve(item.href)}
											onclick={() => (showMenu = false)}
											class="flex items-center justify-between px-5 py-2.5 text-sm transition-colors hover:bg-ink/5 {isActive(
												item.href
											)
												? 'font-semibold text-ink'
												: 'text-ink/70'}"
										>
											{item.label}
											{#if isActive(item.href)}
												<span class="h-1.5 w-1.5 rounded-full bg-teal"></span>
											{/if}
										</a>
									{/each}
								</nav>
								<div class="border-t border-ink/10 px-3 py-2">
									<button
										onclick={handleSync}
										disabled={syncLoading}
										class="flex w-full items-center gap-2.5 rounded-xl px-2 py-2 text-left text-sm text-ink/70 transition-colors hover:bg-ink/5 disabled:opacity-50"
									>
										<svg
											class="h-4 w-4 {syncLoading ? 'animate-spin' : ''}"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											stroke-width="2"
										>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
											/>
										</svg>
										<span class="flex-1">
											{syncLoading
												? 'Menyinkronkan…'
												: syncStatus.error
													? 'Sinkron gagal — coba lagi'
													: syncStatus.queuedCount > 0
														? `Sinkronkan (${syncStatus.queuedCount} menunggu)`
														: 'Sinkronkan data'}
										</span>
									</button>
								</div>
								<div class="border-t border-ink/10 p-3">
									<button
										onclick={handleLogout}
										class="w-full rounded-xl border border-ink/15 py-2.5 text-sm font-semibold text-clay transition-colors hover:border-clay/30 hover:bg-clay/5"
									>
										Keluar
									</button>
								</div>
							</div>
						{/if}
					</div>
				</div>
			</div>
		</header>

		<!-- ═══════════ Notification drawer ═══════════ -->
		{#if showNotifications}
			<!-- svelte-ignore a11y_click_events_have_key_events -->
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div
				class="fixed inset-0 z-[60] bg-ink/25 backdrop-blur-sm transition-opacity"
				onclick={() => (showNotifications = false)}
			></div>
			<div
				class="fixed top-0 right-0 z-[70] flex h-full w-full max-w-sm flex-col bg-card shadow-2xl"
				transition:fly={{ x: 400, duration: 400 }}
			>
				<div class="flex items-center justify-between border-b border-ink/10 p-7">
					<div>
						<h2 class="font-serif text-2xl text-ink">Notifikasi</h2>
						<p class="mt-1 text-[10px] font-semibold tracking-widest text-ink/40 uppercase">
							Pesan terbaru Anda
						</p>
					</div>
					<button
						aria-label="Tutup notifikasi"
						onclick={() => (showNotifications = false)}
						class="rounded-full p-2 text-ink/40 transition-colors hover:bg-ink/5 hover:text-ink"
					>
						<svg
							class="h-5 w-5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"
						>
							<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<div class="flex-1 space-y-3 overflow-y-auto p-5">
					{#each notifications as n (n.id)}
						<!-- svelte-ignore a11y_click_events_have_key_events -->
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<div
							class="group cursor-pointer rounded-2xl border p-4 transition-all {n.read
								? 'border-ink/10 bg-transparent opacity-60'
								: 'border-teal/25 bg-field'}"
							onclick={() => markRead(n.id)}
						>
							<div class="mb-1.5 flex items-start justify-between gap-3">
								<h3 class="text-sm font-semibold text-ink transition-colors group-hover:text-teal">
									{n.title}
								</h3>
								{#if !n.read}
									<div class="mt-1 h-2 w-2 shrink-0 rounded-full bg-teal"></div>
								{/if}
							</div>
							<p class="mb-2.5 text-[13px] leading-relaxed text-ink/60">{n.message}</p>
							<p class="text-[10px] font-semibold tracking-widest text-ink/35 uppercase">
								{n.time}
							</p>
						</div>
					{/each}
				</div>

				<div class="border-t border-ink/10 p-5">
					<button
						onclick={markAllRead}
						class="w-full rounded-full border border-ink/15 py-3 text-[11px] font-semibold tracking-widest text-ink uppercase transition-all hover:bg-ink hover:text-card"
					>
						Tandai semua dibaca
					</button>
				</div>
			</div>
		{/if}

		<!-- ═══════════ Page body ═══════════ -->
		<main class="flex-1 px-5 pt-8 pb-28 sm:px-6 lg:px-10 lg:pt-12 lg:pb-16">
			<div class="mx-auto max-w-6xl">
				{@render children()}
			</div>
		</main>

		<!-- ═══════════ Mobile bottom nav ═══════════ -->
		<nav
			class="fixed right-0 bottom-0 left-0 z-50 border-t border-ink/10 bg-card lg:hidden"
			style="padding-bottom: env(safe-area-inset-bottom);"
		>
			<div class="flex items-stretch">
				{#each bottomNav as item (item.href)}
					<a
						href={resolve(item.href)}
						class="flex flex-1 flex-col items-center gap-1 py-2.5 text-[9px] font-semibold transition-colors {bottomActive(
							item
						)
							? 'text-ink'
							: 'text-ink/40'}"
					>
						<svg
							class="h-[18px] w-[18px]"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width={bottomActive(item) ? 2.4 : 1.8}
						>
							<path stroke-linecap="round" stroke-linejoin="round" d={item.icon} />
						</svg>
						<span>{item.label}</span>
						<span class="h-0.5 w-3 rounded-full {bottomActive(item) ? 'bg-teal' : 'bg-transparent'}"
						></span>
					</a>
				{/each}
			</div>
		</nav>

		<!-- Mobile quick-add FAB -->
		<a
			href={resolve('/transactions')}
			class="fixed right-5 bottom-20 z-50 flex h-14 w-14 items-center justify-center rounded-full bg-teal text-3xl font-light text-card shadow-lg shadow-ink/30 transition-transform hover:scale-105 lg:hidden"
			style="margin-bottom: env(safe-area-inset-bottom);"
			aria-label="Catat transaksi"
		>
			+
		</a>
	</div>
{/if}
