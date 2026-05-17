<script lang="ts">
	import { resolve } from '$app/paths';

	function retry() {
		if (typeof location !== 'undefined') location.reload();
	}
</script>

<svelte:head>
	<title>Mode Offline · KasKu</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-gray-50 px-6">
	<div class="max-w-md space-y-6 text-center">
		<div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-teal-50">
			<svg
				class="h-8 w-8 text-teal-700"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				stroke-width="2"
				aria-hidden="true"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="M18.364 5.636l-12.728 12.728m0-12.728l12.728 12.728"
				/>
			</svg>
		</div>
		<div class="space-y-2">
			<h1 class="text-2xl font-black text-[#0a2e31]">Tidak Ada Koneksi</h1>
			<p class="text-sm leading-relaxed font-medium text-gray-500">
				Anda sedang offline. Data terakhir tetap tersimpan di perangkat — perubahan akan
				disinkronkan otomatis saat koneksi pulih.
			</p>
		</div>
		<div class="flex flex-col items-stretch gap-3 sm:flex-row sm:justify-center">
			<button
				type="button"
				onclick={retry}
				class="rounded-2xl bg-[#217b84] px-6 py-3 text-sm font-bold text-white shadow-lg transition-all hover:bg-[#1a5f66] active:scale-95"
			>
				Coba Sambungkan Ulang
			</button>
			<a
				href={resolve('/dashboard')}
				class="rounded-2xl border border-gray-200 bg-white px-6 py-3 text-sm font-bold text-[#0a2e31] transition-colors hover:bg-gray-50"
			>
				Buka Dashboard Tersimpan
			</a>
		</div>
	</div>
</div>
