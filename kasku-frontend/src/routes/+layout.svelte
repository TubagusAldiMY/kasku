<script lang="ts">
	import './layout.css';
	import '@fontsource/instrument-serif/400.css';
	import '@fontsource/instrument-serif/400-italic.css';
	import '@fontsource-variable/instrument-sans/wght.css';
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import { apiFetch } from '$lib/api/client';
	import { registerServiceWorker, skipWaiting } from '$lib/sw-bridge';

	const { children } = $props();

	let updateAvailable = $state(false);
	let lastUpdateShown = 0;

	function onUpdateAvailable() {
		const now = Date.now();
		if (now - lastUpdateShown < 10_000) return;
		lastUpdateShown = now;
		updateAvailable = true;
	}

	onMount(async () => {
		void registerServiceWorker({ onUpdateAvailable });

		if (auth.accessToken) {
			auth.setLoading(false);
			return;
		}

		if (localStorage.getItem('kasku_mock_mode') === 'true') {
			auth.setLoading(false);
			return;
		}

		try {
			const response = await apiFetch('/auth/refresh', {
				method: 'POST',
				skipAuth: true
			});
			const result = await response.json();
			if (result.success && result.data?.access_token) {
				auth.setToken(result.data.access_token);
			} else {
				auth.setLoading(false);
			}
		} catch {
			auth.setLoading(false);
		}
	});

	function reloadForUpdate() {
		updateAvailable = false;
		skipWaiting();
		if ('serviceWorker' in navigator) {
			navigator.serviceWorker.addEventListener('controllerchange', () => location.reload(), {
				once: true
			});
		} else {
			location.reload();
		}
	}
</script>

{#if updateAvailable}
	<div
		class="fixed top-3 left-1/2 z-[100] flex w-max max-w-[calc(100vw-2rem)] -translate-x-1/2 items-center gap-2.5 rounded-full border border-ink/15 bg-card px-4 py-2 text-xs font-semibold text-ink shadow-lg"
		role="status"
	>
		<span class="whitespace-nowrap">Pembaruan tersedia</span>
		<button
			type="button"
			onclick={reloadForUpdate}
			class="rounded-full bg-teal px-3.5 py-1 text-[11px] font-semibold whitespace-nowrap text-card transition-colors hover:bg-ink"
		>
			Muat ulang
		</button>
	</div>
{/if}

{@render children()}
