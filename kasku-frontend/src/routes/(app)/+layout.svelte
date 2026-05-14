<script lang="ts">
	import { onMount } from 'svelte';
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
</script>

{#if auth.loading}
	<div class="flex min-h-screen items-center justify-center bg-gray-50">
		<div class="flex flex-col items-center">
			<div class="h-10 w-10 animate-spin rounded-full border-b-2 border-indigo-600"></div>
			<p class="mt-4 text-sm text-gray-500 font-bold uppercase tracking-widest animate-pulse">KasKu</p>
		</div>
	</div>
{:else}
	<div class="min-h-screen bg-gray-50 flex">
		<!-- Simple Sidebar Placeholder -->
		<aside class="hidden lg:flex w-64 flex-col bg-[#0a2e31] text-white p-6">
			<div class="flex items-center gap-2 mb-10">
				<div class="h-8 w-8 rounded-lg bg-[#217b84] flex items-center justify-center">
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
				</div>
				<span class="text-xl font-black">KasKu</span>
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
					<div class="h-10 w-10 rounded-full bg-teal-500/20 border border-teal-500/50 flex items-center justify-center text-teal-400 font-black">
						{auth.user?.username?.charAt(0).toUpperCase()}
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
				<button onclick={handleLogout} class="text-xs font-bold text-red-500">Logout</button>
			</header>

			<main class="flex-1 overflow-y-auto p-6 lg:p-12">
				<div class="max-w-6xl mx-auto">
					{@render children()}
				</div>
			</main>
		</div>
	</div>
{/if}
