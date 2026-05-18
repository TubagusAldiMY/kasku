<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { adminAuth } from '$lib/stores/admin_auth.svelte';
	import { adminApiFetch } from '$lib/api/admin_client';

	let { children } = $props();

	const isLoginRoute = $derived(page.url.pathname.endsWith('/admin/login'));

	let online = $state(typeof navigator !== 'undefined' ? navigator.onLine : true);

	function setOnline() {
		online = true;
	}
	function setOffline() {
		online = false;
	}

	onMount(() => {
		window.addEventListener('online', setOnline);
		window.addEventListener('offline', setOffline);
		return () => {
			window.removeEventListener('online', setOnline);
			window.removeEventListener('offline', setOffline);
		};
	});

	$effect(() => {
		if (isLoginRoute) return;
		if (!adminAuth.isAuthenticated) {
			goto(resolve('/admin/login'));
		}
	});

	async function handleLogout() {
		try {
			await adminApiFetch('/admin/auth/logout', { method: 'POST' });
		} catch {
			// Best effort — token blacklist sisi server hanya bonus.
		} finally {
			adminAuth.logout();
			goto(resolve('/admin/login'));
		}
	}

	type NavRoute =
		| '/(admin)/admin/dashboard'
		| '/(admin)/admin/users'
		| '/(admin)/admin/payments'
		| '/(admin)/admin/audit-log';
	type NavItem = { route: NavRoute; label: string };
	const navItems: NavItem[] = [
		{ route: '/(admin)/admin/dashboard', label: 'Dashboard' },
		{ route: '/(admin)/admin/users', label: 'Pengguna' },
		{ route: '/(admin)/admin/payments', label: 'Pembayaran' },
		{ route: '/(admin)/admin/audit-log', label: 'Audit Log' }
	];

	function isActive(route: NavRoute): boolean {
		return page.url.pathname.startsWith(resolve(route));
	}
</script>

{#if isLoginRoute}
	{@render children()}
{:else}
	{#if !online}
		<div
			class="fixed top-0 right-0 left-0 z-[80] flex items-center justify-center gap-2 bg-red-500 py-2 text-[11px] font-bold tracking-wider text-white uppercase"
			role="status"
		>
			<span>Mode Offline · Panel admin memerlukan koneksi internet</span>
		</div>
	{/if}

	<div class="flex min-h-screen bg-gray-100">
		<aside class="flex w-64 flex-col bg-[#0a2e31] p-6 text-white shadow-xl">
			<div class="mb-10 flex items-center gap-2">
				<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-red-500">
					<svg class="h-5 w-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
						/></svg
					>
				</div>
				<span class="text-xl font-black"
					>KasKu <span class="ml-1 rounded bg-red-500 px-1.5 py-0.5 text-[10px]">ADMIN</span></span
				>
			</div>

			<nav class="flex-1 space-y-1">
				{#each navItems as item (item.route)}
					<a
						href={resolve(item.route)}
						class="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold transition-colors {isActive(
							item.route
						)
							? 'bg-white/10 text-white'
							: 'text-white/60 hover:bg-white/5 hover:text-white'}"
					>
						{item.label}
					</a>
				{/each}
			</nav>

			<div class="mt-auto border-t border-white/10 pt-6">
				<button
					onclick={handleLogout}
					class="w-full rounded-xl border border-white/10 py-3 text-xs font-bold transition-all hover:bg-red-500/10 hover:text-red-400"
				>
					Keluar Panel
				</button>
			</div>
		</aside>

		<div class="flex flex-1 flex-col overflow-hidden">
			<header class="flex h-16 items-center justify-between border-b border-gray-200 bg-white px-8">
				<h2 class="font-black text-[#0a2e31]">Administrator Panel</h2>
				<div class="flex items-center gap-3">
					<div class="text-right">
						<p class="text-xs font-black text-[#0a2e31]">
							{adminAuth.admin?.username ?? '—'}
						</p>
						<p class="text-[10px] font-bold tracking-widest text-gray-400 uppercase">
							{adminAuth.admin?.role ?? 'Admin'}
						</p>
					</div>
					<div class="h-8 w-8 rounded-full border border-gray-200 bg-gray-100"></div>
				</div>
			</header>

			<main class="flex-1 overflow-y-auto p-8">
				<div class="mx-auto max-w-6xl">
					{@render children()}
				</div>
			</main>
		</div>
	</div>
{/if}
