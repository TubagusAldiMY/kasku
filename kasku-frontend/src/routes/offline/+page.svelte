<script lang="ts">
	import { resolve } from '$app/paths';

	function retry() {
		if (typeof location !== 'undefined') location.reload();
	}
</script>

<svelte:head>
	<title>Mode Offline · KasKu</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-paper px-6">
	<div class="max-w-md space-y-8 text-center">
		<a href={resolve('/')} class="font-serif text-[26px] leading-none tracking-tight text-ink">
			Kas<em class="text-teal">Ku</em>
		</a>

		<div class="space-y-3">
			<h1 class="font-serif text-3xl leading-tight text-ink">Kamu sedang offline</h1>
			<p class="text-[15px] leading-relaxed text-ink/60">
				Data terakhir tetap tersimpan di perangkat — perubahan akan disinkronkan otomatis saat
				koneksi pulih.
			</p>
		</div>

		<div class="flex flex-col items-stretch gap-3 sm:flex-row sm:justify-center">
			<button
				type="button"
				onclick={retry}
				class="rounded-full bg-teal px-6 py-3 text-[15px] font-semibold text-card transition-colors hover:bg-ink"
			>
				Coba Sambungkan Ulang
			</button>
			<a
				href={resolve('/dashboard')}
				class="rounded-full border border-ink/25 px-6 py-3 text-[15px] font-semibold text-ink transition-colors hover:border-ink/40"
			>
				Buka Dashboard Tersimpan
			</a>
		</div>
	</div>
</div>
