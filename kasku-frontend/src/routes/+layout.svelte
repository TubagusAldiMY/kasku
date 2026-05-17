<script lang="ts">
	import './layout.css';
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import { apiFetch } from '$lib/api/client';

	const { children } = $props();

	onMount(async () => {
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
</script>

{@render children()}
