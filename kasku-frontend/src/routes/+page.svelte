<script lang="ts">
	import { resolve } from '$app/paths';
	import { auth } from '$lib/stores/auth.svelte';

	const features = [
		{
			title: 'Bekerja offline',
			body: 'PWA dengan penyimpanan lokal — catat di gunung, sinkron di kota. Antrian perubahan aman.'
		},
		{
			title: 'Satu tempat untuk semua',
			body: 'Rekening, transaksi, anggaran, investasi dengan harga live, dan hutang-piutang.'
		},
		{
			title: 'Kekayaan bersih, jujur',
			body: 'Saldo + investasi − hutang. Satu angka yang tidak berbohong tentang kondisimu.'
		}
	];
</script>

<div class="min-h-screen">
	<div class="mx-auto max-w-6xl px-6 lg:px-10">
		<!-- Nav -->
		<header class="flex items-center justify-between border-b border-ink/10 py-6">
			<a href={resolve('/')} class="font-serif text-[26px] leading-none tracking-tight text-ink">
				Kas<em class="text-teal">Ku</em>
			</a>
			<nav class="hidden items-center gap-7 text-[13.5px] font-medium text-ink/60 sm:flex">
				<a href="#fitur" class="hover:text-ink">Fitur</a>
				<a href="#fitur" class="hover:text-ink">Harga</a>
				<a href="#fitur" class="hover:text-ink">Tentang</a>
			</nav>
			<div class="flex items-center gap-2 sm:gap-3">
				{#if auth.isAuthenticated}
					<a
						href={resolve('/dashboard')}
						class="rounded-full bg-ink px-5 py-2.5 text-[13.5px] font-semibold text-card transition-colors hover:bg-teal"
					>
						Buka dashboard
					</a>
				{:else}
					<a
						href={resolve('/login')}
						class="px-4 py-2 text-[13.5px] font-semibold text-ink hover:text-teal"
					>
						Masuk
					</a>
					<a
						href={resolve('/register')}
						class="rounded-full bg-ink px-5 py-2.5 text-[13.5px] font-semibold text-card transition-colors hover:bg-teal"
					>
						Daftar gratis
					</a>
				{/if}
			</div>
		</header>

		<!-- Hero -->
		<section class="px-2 py-20 text-center sm:py-28">
			<p class="mb-5 text-[12px] font-semibold tracking-[0.16em] text-teal uppercase">
				Pencatat keuangan pribadi · Offline-first
			</p>
			<h1
				class="mx-auto max-w-3xl font-serif text-5xl leading-[1.05] tracking-tight text-ink sm:text-6xl lg:text-7xl"
			>
				Uangmu tercatat.<br /><em class="text-teal">Pikiranmu tenang.</em>
			</h1>
			<p class="mx-auto mt-7 max-w-xl text-base leading-relaxed text-ink/65">
				Catat transaksi, anggaran, investasi, sampai hutang-piutang — tetap jalan tanpa internet,
				tersinkron otomatis saat tersambung.
			</p>
			<div class="mt-9 flex flex-col items-center justify-center gap-3 sm:flex-row">
				<a
					href={resolve(auth.isAuthenticated ? '/dashboard' : '/register')}
					class="rounded-full bg-teal px-8 py-3.5 text-[15px] font-semibold text-card transition-colors hover:bg-ink"
				>
					Mulai mencatat — gratis
				</a>
				<a
					href={resolve('/login')}
					class="rounded-full border border-ink/25 px-8 py-3.5 text-[15px] font-semibold text-ink transition-colors hover:border-ink/40"
				>
					Lihat demo
				</a>
			</div>
		</section>

		<!-- Product strip -->
		<section
			id="fitur"
			class="grid gap-10 rounded-t-3xl border border-b-0 border-ink/12 bg-field px-8 pt-9 pb-0 sm:grid-cols-3 sm:px-10"
		>
			{#each features as f (f.title)}
				<div class="border-t-2 border-ink pt-4 pb-8">
					<p class="font-serif text-[22px] text-ink">{f.title}</p>
					<p class="mt-2 text-[13.5px] leading-relaxed text-ink/60">{f.body}</p>
				</div>
			{/each}
		</section>
	</div>

	<!-- Footer -->
	<footer class="border-t border-ink/10">
		<div
			class="mx-auto flex max-w-6xl items-center justify-between px-6 py-5 text-xs text-ink/45 lg:px-10"
		>
			<span>© 2026 KasKu</span>
			<span class="flex gap-5">
				<a href={resolve('/privacy')} class="hover:text-teal">Privasi</a>
				<a href={resolve('/terms')} class="hover:text-teal">Ketentuan</a>
			</span>
		</div>
	</footer>
</div>
