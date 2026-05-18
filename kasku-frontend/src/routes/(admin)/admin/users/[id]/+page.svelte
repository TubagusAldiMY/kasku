<script lang="ts">
	import { page } from '$app/state';

	// Halaman Detail Pengguna — belum diimplementasi
	const userId = $derived(page.params.id);
</script>

<div class="space-y-4">
	<h1 class="text-3xl font-black text-[#0a2e31]">Detail Pengguna</h1>
	<p class="text-sm text-gray-500">ID: {userId}</p>
	<p class="text-sm text-gray-500">Fitur ini sedang dalam pengembangan.</p>
</div>
