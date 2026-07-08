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
		{ route: '/(admin)/admin/dashboard', label: 'Ikhtisar' },
		{ route: '/(admin)/admin/users', label: 'Pengguna' },
		{ route: '/(admin)/admin/payments', label: 'Pembayaran' },
		{ route: '/(admin)/admin/audit-log', label: 'Audit log' }
	];

	function isActive(route: NavRoute): boolean {
		return page.url.pathname.startsWith(resolve(route));
	}
</script>

{#if isLoginRoute}
	{@render children()}
{:else}
	<div class="flex min-h-screen flex-col bg-paper">
		{#if !online}
			<div
				class="flex items-center justify-center gap-2 bg-clay py-2 text-[11px] font-semibold tracking-wider text-card uppercase"
				role="status"
			>
				<span>Mode offline · panel admin memerlukan koneksi internet</span>
			</div>
		{/if}

		<!-- Dark editorial top bar (mockup admin variant) -->
		<header class="bg-ink text-card">
			<div
				class="mx-auto flex max-w-6xl items-center justify-between gap-4 px-5 py-4 sm:px-6 lg:px-10"
			>
				<div class="flex items-center gap-6 lg:gap-10">
					<span class="flex items-baseline gap-2.5">
						<span class="font-serif text-[24px] leading-none tracking-tight"
							>Kas<em class="text-mint">Ku</em></span
						>
						<span
							class="rounded-full bg-mint/15 px-2.5 py-0.5 text-[10px] font-semibold tracking-[0.14em] text-mint uppercase"
						>
							Admin
						</span>
					</span>
					<nav class="hidden items-center gap-7 text-[13.5px] font-medium sm:flex">
						{#each navItems as item (item.route)}
							<a
								href={resolve(item.route)}
								class="pb-1 transition-colors {isActive(item.route)
									? 'border-b-2 border-mint text-card'
									: 'text-card/55 hover:text-card'}"
							>
								{item.label}
							</a>
						{/each}
					</nav>
				</div>

				<div class="flex items-center gap-4">
					<div class="hidden text-right sm:block">
						<p class="text-xs font-semibold text-card">{adminAuth.admin?.username ?? '—'}</p>
						<p class="text-[10px] font-semibold tracking-widest text-card/45 uppercase">
							{adminAuth.admin?.role ?? 'Admin'}
						</p>
					</div>
					<div
						class="flex h-9 w-9 items-center justify-center rounded-full border border-mint/40 bg-mint/15 text-[13px] font-semibold text-mint"
					>
						{adminAuth.admin?.username?.charAt(0).toUpperCase() ?? 'A'}
					</div>
					<button
						onclick={handleLogout}
						class="rounded-full border border-card/20 px-4 py-1.5 text-xs font-semibold text-card/80 transition-colors hover:border-mint/40 hover:text-mint"
					>
						Keluar
					</button>
				</div>
			</div>

			<!-- Mobile nav row -->
			<nav
				class="flex gap-6 overflow-x-auto border-t border-card/10 px-5 py-2.5 text-[13px] font-medium sm:hidden"
			>
				{#each navItems as item (item.route)}
					<a
						href={resolve(item.route)}
						class="whitespace-nowrap {isActive(item.route) ? 'text-mint' : 'text-card/55'}"
					>
						{item.label}
					</a>
				{/each}
			</nav>
		</header>

		<main class="mx-auto w-full max-w-6xl flex-1 px-5 py-10 sm:px-6 lg:px-10">
			{@render children()}
		</main>
	</div>
{/if}
