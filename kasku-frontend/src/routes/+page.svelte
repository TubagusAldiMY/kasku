<script lang="ts">
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.svelte';

	// Masuk mode demo (mock) langsung dari landing — sama seperti tombol di halaman login.
	function handleDemo() {
		auth.setToken('mock-jwt-token');
		auth.setUser({ id: 'mock-user-id', email: 'demo@kasku.id', username: 'Juragan Demo' });
		localStorage.setItem('kasku_mock_mode', 'true');
		goto(resolve('/dashboard'));
	}

	function formatPrice(val: number) {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(val);
	}

	const features = [
		{
			title: 'Laporan & arus kas',
			body: 'Lihat ke mana uangmu pergi — rincian per kategori dan tren pemasukan vs pengeluaran tiap bulan.'
		},
		{
			title: 'Transaksi & rekening',
			body: 'Catat pemasukan, pengeluaran, dan transfer lintas rekening dalam hitungan detik.'
		},
		{
			title: 'Anggaran per kategori',
			body: 'Tetapkan batas bulanan tiap kategori dan lihat progresnya sebelum kebablasan.'
		},
		{
			title: 'Investasi harga live',
			body: 'Pantau kripto, saham, dan emas dengan harga terkini — untung/rugi dihitung otomatis.'
		},
		{
			title: 'Hutang & piutang',
			body: 'Lacak siapa berhutang padamu dan tagihan yang jatuh tempo, tanpa ada yang terlewat.'
		},
		{
			title: 'Kekayaan bersih, jujur',
			body: 'Saldo + investasi − hutang. Satu angka yang tidak berbohong tentang kondisimu.'
		}
	];

	type Plan = {
		name: string;
		price: number | null;
		desc: string;
		features: string[];
		cta: string;
		popular: boolean;
		contact?: boolean;
	};

	const plans: Plan[] = [
		{
			name: 'Gratis',
			price: 0,
			desc: 'Cocok untuk yang baru mulai mencatat.',
			features: [
				'50 transaksi / bulan',
				'3 rekening keuangan',
				'Riwayat 3 bulan terakhir',
				'Akses web & mobile'
			],
			cta: 'Mulai gratis',
			popular: false
		},
		{
			name: 'Pro',
			price: 29000,
			desc: 'Untuk pengelolaan aset yang lebih serius.',
			features: [
				'Transaksi tak terbatas',
				'Rekening tak terbatas',
				'Riwayat selamanya',
				'Ekspor PDF & CSV',
				'Grafik analisis detail'
			],
			cta: 'Pilih Pro',
			popular: true
		},
		{
			name: 'Enterprise',
			price: null,
			desc: 'Untuk tim, keluarga & bisnis skala besar.',
			features: [
				'Semua fitur Pro',
				'Multi-user (5 user)',
				'Analisis prediktif AI',
				'API access',
				'Dukungan prioritas'
			],
			cta: 'Hubungi kami',
			popular: false,
			contact: true
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
				<a href="#harga" class="hover:text-ink">Harga</a>
				<a href="#tentang" class="hover:text-ink">Tentang</a>
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
				Pencatat keuangan pribadi
			</p>
			<h1
				class="mx-auto max-w-3xl font-serif text-5xl leading-[1.05] tracking-tight text-ink sm:text-6xl lg:text-7xl"
			>
				Uangmu tercatat.<br /><em class="text-teal">Pikiranmu tenang.</em>
			</h1>
			<p class="mx-auto mt-7 max-w-xl text-base leading-relaxed text-ink/65">
				Catat transaksi, anggaran, investasi, sampai hutang-piutang — semuanya rapi di satu tempat,
				cepat dan tanpa ribet.
			</p>
			<div class="mt-9 flex flex-col items-center justify-center gap-3 sm:flex-row">
				{#if auth.isAuthenticated}
					<a
						href={resolve('/dashboard')}
						class="rounded-full bg-teal px-8 py-3.5 text-[15px] font-semibold text-card transition-colors hover:bg-ink"
					>
						Buka dashboard
					</a>
					<a
						href="#fitur"
						class="rounded-full border border-ink/25 px-8 py-3.5 text-[15px] font-semibold text-ink transition-colors hover:border-ink/40"
					>
						Lihat fitur
					</a>
				{:else}
					<a
						href={resolve('/register')}
						class="rounded-full bg-teal px-8 py-3.5 text-[15px] font-semibold text-card transition-colors hover:bg-ink"
					>
						Mulai mencatat — gratis
					</a>
					<button
						type="button"
						onclick={handleDemo}
						class="rounded-full border border-ink/25 px-8 py-3.5 text-[15px] font-semibold text-ink transition-colors hover:border-ink/40"
					>
						Coba tanpa daftar
					</button>
				{/if}
			</div>
		</section>

		<!-- Fitur -->
		<section id="fitur" class="scroll-mt-8 border-t border-ink/10 py-20">
			<div class="mb-12 max-w-2xl">
				<p class="mb-3 text-[12px] font-semibold tracking-[0.16em] text-teal uppercase">Fitur</p>
				<h2 class="font-serif text-4xl leading-tight tracking-tight text-ink sm:text-5xl">
					Semua sisi keuanganmu, satu aplikasi.
				</h2>
				<p class="mt-4 text-base leading-relaxed text-ink/60">
					Dari catatan harian sampai kekayaan bersih — cepat, rapi, dan tanpa basa-basi.
				</p>
			</div>
			<div class="grid gap-x-10 gap-y-10 sm:grid-cols-2 lg:grid-cols-3">
				{#each features as f (f.title)}
					<div class="border-t-2 border-ink pt-4">
						<p class="font-serif text-[22px] text-ink">{f.title}</p>
						<p class="mt-2 text-[13.5px] leading-relaxed text-ink/60">{f.body}</p>
					</div>
				{/each}
			</div>
		</section>

		<!-- Harga -->
		<section id="harga" class="scroll-mt-8 border-t border-ink/10 py-20">
			<div class="mb-12 max-w-2xl">
				<p class="mb-3 text-[12px] font-semibold tracking-[0.16em] text-teal uppercase">Harga</p>
				<h2 class="font-serif text-4xl leading-tight tracking-tight text-ink sm:text-5xl">
					Mulai gratis. Upgrade saat siap.
				</h2>
				<p class="mt-4 text-base leading-relaxed text-ink/60">
					Tanpa kartu kredit untuk memulai. Batalkan kapan saja.
				</p>
			</div>
			<div class="grid gap-6 lg:grid-cols-3">
				{#each plans as p (p.name)}
					<div
						class="flex flex-col rounded-2xl border p-8 {p.popular
							? 'border-teal bg-card'
							: 'border-ink/12'}"
					>
						<div class="flex items-center justify-between">
							<p class="font-serif text-2xl text-ink">{p.name}</p>
							{#if p.popular}
								<span
									class="rounded-full border border-teal/30 px-2.5 py-0.5 text-[11px] font-semibold text-teal"
								>
									Populer
								</span>
							{/if}
						</div>

						<p class="mt-5 font-serif text-4xl tracking-tight text-ink tabular-nums">
							{#if p.price === null}
								Custom
							{:else if p.price === 0}
								Gratis
							{:else}
								{formatPrice(p.price)}<span class="ml-1 font-sans text-base text-ink/45">/bln</span>
							{/if}
						</p>
						<p class="mt-3 text-[13.5px] leading-relaxed text-ink/60">{p.desc}</p>

						<ul
							class="mt-6 flex-1 space-y-2.5 border-t border-ink/10 pt-6 text-[13.5px] text-ink/70"
						>
							{#each p.features as feat (feat)}
								<li class="flex items-baseline gap-2.5">
									<svg
										class="h-3.5 w-3.5 shrink-0 translate-y-0.5 text-teal"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2.5"
									>
										<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
									</svg>
									{feat}
								</li>
							{/each}
						</ul>

						{#if p.contact}
							<a
								href="mailto:admin@tubsamy.tech"
								class="mt-8 rounded-full border border-ink/20 py-3 text-center text-sm font-semibold text-ink transition-colors hover:border-ink/40"
							>
								{p.cta}
							</a>
						{:else}
							<a
								href={resolve('/register')}
								class="mt-8 rounded-full py-3 text-center text-sm font-semibold transition-colors {p.popular
									? 'bg-teal text-card hover:bg-ink'
									: 'border border-ink/20 text-ink hover:border-ink/40'}"
							>
								{p.cta}
							</a>
						{/if}
					</div>
				{/each}
			</div>
		</section>

		<!-- Tentang -->
		<section id="tentang" class="scroll-mt-8 border-t border-ink/10 py-20">
			<div class="grid gap-10 lg:grid-cols-[1fr_1.4fr] lg:gap-16">
				<div>
					<p class="mb-3 text-[12px] font-semibold tracking-[0.16em] text-teal uppercase">
						Tentang
					</p>
					<h2 class="font-serif text-4xl leading-tight tracking-tight text-ink sm:text-5xl">
						Tertib dulu, kaya kemudian.
					</h2>
				</div>
				<div class="space-y-5 text-base leading-relaxed text-ink/70">
					<p>
						KasKu lahir dari keyakinan sederhana: kamu tidak perlu jadi kaya untuk mulai tertib —
						kamu perlu tertib untuk mulai kaya. Kami membuat pencatatan keuangan terasa ringan,
						bukan pekerjaan rumah.
					</p>
					<p>
						Kami menaruh <span class="font-semibold text-ink">privasi</span> di urutan pertama. Catatanmu
						adalah milikmu — tersimpan aman, tidak pernah kami perjualbelikan, dan tidak kami pakai untuk
						hal yang tidak kamu setujui.
					</p>
					<p>
						Dibuat di Indonesia untuk cara orang Indonesia mengatur uang — dari rupiah dan e-wallet,
						emas dan saham, sampai arisan dan hutang-piutang. Satu tempat, satu angka jujur soal ke
						mana uangmu pergi.
					</p>
				</div>
			</div>
		</section>
	</div>

	<!-- Footer -->
	<footer class="border-t border-ink/10">
		<div
			class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-3 px-6 py-6 text-xs text-ink/45 sm:flex-row lg:px-10"
		>
			<span>© 2026 KasKu</span>
			<nav class="flex gap-5">
				<a href="#fitur" class="hover:text-teal">Fitur</a>
				<a href="#harga" class="hover:text-teal">Harga</a>
				<a href="#tentang" class="hover:text-teal">Tentang</a>
				<a href={resolve('/privacy')} class="hover:text-teal">Privasi</a>
				<a href={resolve('/terms')} class="hover:text-teal">Ketentuan</a>
			</nav>
		</div>
	</footer>
</div>
