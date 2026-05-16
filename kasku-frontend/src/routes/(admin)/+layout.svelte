<script lang="ts">
	import { auth } from '$lib/stores/auth.svelte';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => {
		// Mock Admin Role Check
		if (!auth.accessToken) {
			// goto('/login');
		}
	});

	function handleLogout() {
		auth.logout();
		goto('/login');
	}
</script>

<div class="min-h-screen bg-gray-100 flex">
	<!-- Admin Sidebar -->
	<aside class="w-64 bg-[#0a2e31] text-white flex flex-col p-6 shadow-xl">
		<div class="flex items-center gap-2 mb-10">
			<div class="h-8 w-8 rounded-lg bg-red-500 flex items-center justify-center">
				<svg class="h-5 w-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" /></svg>
			</div>
			<span class="text-xl font-black">KasKu <span class="text-[10px] bg-red-500 px-1.5 py-0.5 rounded ml-1">ADMIN</span></span>
		</div>

		<nav class="space-y-1 flex-1">
			<a href="/admin/dashboard" class="flex items-center gap-3 px-4 py-3 rounded-xl bg-white/10 font-bold text-sm">
				Dashboard
			</a>
			<div class="px-4 py-3 text-[10px] font-black text-white/30 uppercase tracking-widest mt-4">Manajemen</div>
			<button class="w-full flex items-center gap-3 px-4 py-3 rounded-xl hover:bg-white/5 text-white/60 font-bold text-sm text-left">
				Pengguna
			</button>
			<button class="w-full flex items-center gap-3 px-4 py-3 rounded-xl hover:bg-white/5 text-white/60 font-bold text-sm text-left">
				Langganan
			</button>
			<button class="w-full flex items-center gap-3 px-4 py-3 rounded-xl hover:bg-white/5 text-white/60 font-bold text-sm text-left">
				Sistem & Logs
			</button>
		</nav>

		<div class="mt-auto pt-6 border-t border-white/10">
			<button onclick={handleLogout} class="w-full py-3 rounded-xl border border-white/10 hover:bg-red-500/10 hover:text-red-400 font-bold text-xs">
				Keluar Panel
			</button>
		</div>
	</aside>

	<!-- Main Content -->
	<div class="flex-1 flex flex-col overflow-hidden">
		<header class="h-16 bg-white border-b border-gray-200 flex items-center justify-between px-8">
			<h2 class="font-black text-[#0a2e31]">Administrator Panel</h2>
			<div class="flex items-center gap-3">
				<div class="text-right">
					<p class="text-xs font-black text-[#0a2e31]">Super Admin</p>
					<p class="text-[10px] text-gray-400 font-bold">admin@kasku.id</p>
				</div>
				<div class="h-8 w-8 rounded-full bg-gray-100 border border-gray-200"></div>
			</div>
		</header>

		<main class="flex-1 overflow-y-auto p-8">
			<div class="max-w-6xl mx-auto">
				{@render children()}
			</div>
		</main>
	</div>
</div>
