<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { auth } from '$lib/stores/auth.svelte';

	onMount(() => {
		if (auth.isAuthenticated) {
			goto(resolve('/dashboard'));
		} else {
			goto(resolve('/login'));
		}
	});
</script>

<div class="flex min-h-screen items-center justify-center">
	<div class="h-8 w-8 animate-spin rounded-full border-b-2 border-indigo-600"></div>
</div>
