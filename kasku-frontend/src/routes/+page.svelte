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

	// Menu mobile (hamburger) — hanya untuk landing.
	let mobileOpen = $state(false);

	// Scroll-reveal: elemen fade-up saat masuk viewport (sekali saja).
	// Kelas hidden ditambah DARI action, jadi tanpa JS konten tetap terlihat (SSR aman).
	function reveal(node: HTMLElement, delay = 0) {
		if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
		node.classList.add('reveal-hidden');
		node.style.transitionDelay = `${delay}ms`;
		const io = new IntersectionObserver(
			([entry]) => {
				if (entry.isIntersecting) {
					node.classList.replace('reveal-hidden', 'reveal-visible');
					io.disconnect();
				}
			},
			{ threshold: 0.1 }
		);
		io.observe(node);
		return { destroy: () => io.disconnect() };
	}

	const navLinks = [
		{ href: '#fitur', label: 'Fitur' },
		{ href: '#harga', label: 'Harga' },
		{ href: '#tentang', label: 'Tentang' }
	] as const;

	// Ikon inline (stroke 1.8, konsisten dengan ikon app) per fitur.
	const features = [
		{
			title: 'Laporan & arus kas',
			body: 'Lihat ke mana uangmu pergi — rincian per kategori dan tren pemasukan vs pengeluaran tiap bulan.',
			icon: 'M3 3v18h18M8 17V9m4 8V5m4 12v-6'
		},
		{
			title: 'Transaksi & rekening',
			body: 'Catat pemasukan, pengeluaran, dan transfer lintas rekening dalam hitungan detik.',
			icon: 'M8 7h12m0 0l-4-4m4 4l-4 4M16 17H4m0 0l4 4m-4-4l4-4'
		},
		{
			title: 'Anggaran per kategori',
			body: 'Tetapkan batas bulanan tiap kategori dan lihat progresnya sebelum kebablasan.',
			icon: 'M12 8v4l2.5 2.5M21 12a9 9 0 11-18 0 9 9 0 0118 0z'
		},
		{
			title: 'Investasi harga live',
			body: 'Pantau kripto, saham, dan emas dengan harga terkini — untung/rugi dihitung otomatis.',
			icon: 'M13 7h8m0 0v8m0-8l-8 8-4-4-6 6'
		},
		{
			title: 'Hutang & piutang',
			body: 'Lacak siapa berhutang padamu dan tagihan yang jatuh tempo, tanpa ada yang terlewat.',
			icon: 'M17 20h5v-2a4 4 0 00-3-3.87M9 20H4v-2a4 4 0 013-3.87m6-1.13a4 4 0 10-4-4 4 4 0 004 4zm6-4a3 3 0 10-2-5.2M6 11a3 3 0 112-5.2'
		},
		{
			title: 'Kekayaan bersih, jujur',
			body: 'Saldo + investasi − hutang. Satu angka yang tidak berbohong tentang kondisimu.',
			icon: 'M12 3l8 4v5c0 5-3.5 8-8 9-4.5-1-8-4-8-9V7l8-4zM9 12l2 2 4-4'
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

	// Harga statis untuk landing publik. Sumber kebenaran = billing-service
	// (migration 000002_create_subscription_plans). Endpoint /billing/plans butuh JWT
	// sehingga halaman publik ini tidak bisa fetch harga live — jaga tetap sinkron manual.
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
			price: 99000,
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

	// Data statis mock untuk pratinjau dashboard di hero (murni visual marketing).
	const previewBars = [42, 65, 38, 78, 55, 90, 62, 84, 48, 70, 95, 60];
	const previewTx = [
		{ label: 'Gaji bulanan', cat: 'Pemasukan', amount: '+Rp 8.500.000', positive: true },
		{ label: 'Belanja mingguan', cat: 'Kebutuhan', amount: '−Rp 420.000', positive: false },
		{ label: 'Beli 0,5 gr emas', cat: 'Investasi', amount: '−Rp 650.000', positive: false }
	];
</script>

<div class="min-h-screen overflow-x-clip">
	<!-- ═══════════ Nav — fixed, glass blur ═══════════ -->
	<header
		class="fixed top-0 right-0 left-0 z-50 border-b border-ink/10 bg-paper/80 backdrop-blur-md"
	>
		<nav class="relative mx-auto flex max-w-6xl items-center justify-between px-6 py-4 lg:px-10">
			<a href={resolve('/')} class="font-serif text-[26px] leading-none tracking-tight text-ink">
				Kas<em class="text-teal">Ku</em>
			</a>

			<div
				class="absolute top-1/2 left-1/2 hidden -translate-x-1/2 -translate-y-1/2 items-center gap-8 md:flex"
			>
				{#each navLinks as l (l.href)}
					<a
						href={l.href}
						class="text-[13.5px] font-medium text-ink/55 transition-colors hover:text-ink"
					>
						{l.label}
					</a>
				{/each}
			</div>

			<div class="hidden items-center gap-3 md:flex">
				{#if auth.isAuthenticated}
					<a
						href={resolve('/dashboard')}
						class="rounded-full bg-ink px-5 py-2.5 text-[13.5px] font-semibold text-card transition-all hover:scale-105 active:scale-95"
					>
						Buka dashboard
					</a>
				{:else}
					<a
						href={resolve('/login')}
						class="rounded-full px-4 py-2 text-[13.5px] font-semibold text-ink/70 transition-colors hover:bg-ink/5 hover:text-ink"
					>
						Masuk
					</a>
					<a
						href={resolve('/register')}
						class="rounded-full bg-ink px-5 py-2.5 text-[13.5px] font-semibold text-card transition-all hover:scale-105 active:scale-95"
					>
						Daftar gratis
					</a>
				{/if}
			</div>

			<button
				type="button"
				class="rounded-lg p-2 text-ink transition-colors hover:bg-ink/5 md:hidden"
				onclick={() => (mobileOpen = !mobileOpen)}
				aria-label="Buka menu"
				aria-expanded={mobileOpen}
			>
				{#if mobileOpen}
					<svg
						class="h-6 w-6"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
					</svg>
				{:else}
					<svg
						class="h-6 w-6"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h16" />
					</svg>
				{/if}
			</button>
		</nav>

		{#if mobileOpen}
			<div class="animate-fade-up border-t border-ink/10 bg-paper/95 backdrop-blur-md md:hidden">
				<div class="flex flex-col gap-1 px-6 py-4">
					{#each navLinks as l (l.href)}
						<a
							href={l.href}
							class="rounded-lg px-3 py-2.5 text-sm font-medium text-ink/60 transition-colors hover:bg-ink/5 hover:text-ink"
							onclick={() => (mobileOpen = false)}
						>
							{l.label}
						</a>
					{/each}
					<div class="mt-3 flex flex-col gap-2 border-t border-ink/10 pt-4">
						{#if auth.isAuthenticated}
							<a
								href={resolve('/dashboard')}
								class="rounded-full bg-ink py-2.5 text-center text-sm font-semibold text-card"
							>
								Buka dashboard
							</a>
						{:else}
							<a
								href={resolve('/login')}
								class="rounded-full border border-ink/15 py-2.5 text-center text-sm font-semibold text-ink"
							>
								Masuk
							</a>
							<a
								href={resolve('/register')}
								class="rounded-full bg-ink py-2.5 text-center text-sm font-semibold text-card"
							>
								Daftar gratis
							</a>
						{/if}
					</div>
				</div>
			</div>
		{/if}
	</header>

	<!-- ═══════════ Hero ═══════════ -->
	<section class="relative px-6 pt-32 pb-20 text-center sm:pt-40">
		<!-- Glow ambient — blob radial, transform-only -->
		<div class="pointer-events-none absolute inset-0 overflow-hidden" aria-hidden="true">
			<div
				class="absolute top-[-10%] left-1/2 h-[480px] w-[720px] -translate-x-1/2 rounded-full bg-teal/15 blur-3xl"
				style="animation: kasku-drift 14s ease-in-out infinite;"
			></div>
			<div
				class="absolute top-[30%] left-[12%] h-[300px] w-[300px] rounded-full bg-steel/10 blur-3xl"
				style="animation: kasku-drift 18s ease-in-out infinite reverse;"
			></div>
		</div>

		<div class="relative">
			<!-- Badge pill -->
			<div
				class="animate-fade-up mb-8 inline-flex flex-wrap items-center justify-center gap-2 rounded-full border border-ink/15 bg-ink/5 px-4 py-2 backdrop-blur-sm"
			>
				<span class="text-xs whitespace-nowrap text-ink/55"
					>Pencatat keuangan pribadi, dibuat di Indonesia</span
				>
				<a
					href="#fitur"
					class="flex items-center gap-1 text-xs whitespace-nowrap text-teal transition-all hover:brightness-125 active:scale-95"
				>
					Lihat fitur
					<svg
						class="h-3 w-3"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M5 12h14m-7-7l7 7-7 7" />
					</svg>
				</a>
			</div>

			<h1
				class="text-gradient-hero animate-fade-up mx-auto max-w-3xl font-serif text-5xl leading-[1.05] tracking-tight sm:text-6xl lg:text-7xl"
				style="animation-delay: 0.08s;"
			>
				Uangmu tercatat.<br /><em>Pikiranmu tenang.</em>
			</h1>

			<p
				class="animate-fade-up mx-auto mt-7 max-w-xl text-base leading-relaxed text-ink/55"
				style="animation-delay: 0.16s;"
			>
				Catat transaksi, anggaran, investasi, sampai hutang-piutang — semuanya rapi di satu tempat,
				cepat dan tanpa ribet.
			</p>

			<div
				class="animate-fade-up mt-10 flex flex-col items-center justify-center gap-3 sm:flex-row"
				style="animation-delay: 0.24s;"
			>
				{#if auth.isAuthenticated}
					<a
						href={resolve('/dashboard')}
						class="rounded-lg bg-gradient-to-b from-ink via-ink to-ink/70 px-8 py-3.5 text-[15px] font-semibold text-card transition-all hover:scale-105 active:scale-95"
					>
						Buka dashboard
					</a>
					<a
						href="#fitur"
						class="rounded-lg border border-ink/20 px-8 py-3.5 text-[15px] font-semibold text-ink transition-all hover:border-ink/40 hover:bg-ink/5 active:scale-95"
					>
						Lihat fitur
					</a>
				{:else}
					<a
						href={resolve('/register')}
						class="rounded-lg bg-gradient-to-b from-ink via-ink to-ink/70 px-8 py-3.5 text-[15px] font-semibold text-card transition-all hover:scale-105 active:scale-95"
					>
						Mulai mencatat — gratis
					</a>
					<button
						type="button"
						onclick={handleDemo}
						class="rounded-lg border border-ink/20 px-8 py-3.5 text-[15px] font-semibold text-ink transition-all hover:border-ink/40 hover:bg-ink/5 active:scale-95"
					>
						Coba tanpa daftar
					</button>
				{/if}
			</div>

			<!-- Pratinjau dashboard (mock statis, murni visual) -->
			<div
				class="animate-fade-up relative mx-auto mt-16 max-w-4xl"
				style="animation-delay: 0.34s;"
				aria-hidden="true"
			>
				<div class="absolute -inset-x-8 -top-8 -bottom-4 rounded-[32px] bg-teal/10 blur-2xl"></div>
				<div
					class="relative overflow-hidden rounded-2xl border border-ink/12 bg-card/80 text-left shadow-2xl shadow-black/50 backdrop-blur-sm"
				>
					<!-- Browser chrome -->
					<div class="flex items-center gap-1.5 border-b border-ink/8 px-5 py-3.5">
						<span class="h-2.5 w-2.5 rounded-full bg-clay/70"></span>
						<span class="h-2.5 w-2.5 rounded-full bg-gold/70"></span>
						<span class="h-2.5 w-2.5 rounded-full bg-teal/70"></span>
						<span class="ml-4 rounded-md bg-ink/5 px-3 py-1 text-[10px] text-ink/40"
							>app.kasku.id/dashboard</span
						>
					</div>

					<div class="grid gap-5 p-5 sm:grid-cols-3 sm:p-7">
						<!-- Stat tiles -->
						<div class="rounded-xl border border-ink/8 bg-ink/4 p-4">
							<p class="text-[10px] font-semibold tracking-widest text-ink/40 uppercase">
								Kekayaan bersih
							</p>
							<p class="mt-2 font-serif text-2xl text-ink tabular-nums">Rp 128,4 jt</p>
							<p class="mt-1 text-[11px] text-teal">▲ 4,2% bulan ini</p>
						</div>
						<div class="rounded-xl border border-ink/8 bg-ink/4 p-4">
							<p class="text-[10px] font-semibold tracking-widest text-ink/40 uppercase">
								Pemasukan
							</p>
							<p class="mt-2 font-serif text-2xl text-ink tabular-nums">Rp 12,5 jt</p>
							<p class="mt-1 text-[11px] text-ink/40">Juli 2026</p>
						</div>
						<div class="rounded-xl border border-ink/8 bg-ink/4 p-4">
							<p class="text-[10px] font-semibold tracking-widest text-ink/40 uppercase">
								Pengeluaran
							</p>
							<p class="mt-2 font-serif text-2xl text-ink tabular-nums">Rp 7,8 jt</p>
							<p class="mt-1 text-[11px] text-clay">62% dari anggaran</p>
						</div>

						<!-- Bar chart mini -->
						<div class="rounded-xl border border-ink/8 bg-ink/4 p-4 sm:col-span-2">
							<p class="text-[10px] font-semibold tracking-widest text-ink/40 uppercase">
								Arus kas 12 bulan
							</p>
							<div class="mt-4 flex h-24 items-end gap-1.5">
								{#each previewBars as h, i (i)}
									<div
										class="flex-1 rounded-t-sm {i === 10 ? 'bg-teal' : 'bg-teal/25'}"
										style="height: {h}%;"
									></div>
								{/each}
							</div>
						</div>

						<!-- Transaksi terakhir -->
						<div class="rounded-xl border border-ink/8 bg-ink/4 p-4">
							<p class="text-[10px] font-semibold tracking-widest text-ink/40 uppercase">Terbaru</p>
							<ul class="mt-3 space-y-2.5">
								{#each previewTx as t (t.label)}
									<li class="flex items-center justify-between gap-2 text-[11px]">
										<div class="min-w-0">
											<p class="truncate font-medium text-ink">{t.label}</p>
											<p class="text-ink/40">{t.cat}</p>
										</div>
										<span
											class="shrink-0 font-semibold tabular-nums {t.positive
												? 'text-teal'
												: 'text-ink/60'}"
										>
											{t.amount}
										</span>
									</li>
								{/each}
							</ul>
						</div>
					</div>
				</div>
			</div>
		</div>
	</section>

	<div class="mx-auto max-w-6xl px-6 lg:px-10">
		<!-- ═══════════ Fitur ═══════════ -->
		<section id="fitur" class="scroll-mt-24 border-t border-ink/10 py-20">
			<div class="mb-12 max-w-2xl" use:reveal>
				<p class="mb-3 text-[12px] font-semibold tracking-[0.16em] text-teal uppercase">Fitur</p>
				<h2 class="font-serif text-4xl leading-tight tracking-tight text-ink sm:text-5xl">
					Semua sisi keuanganmu, satu aplikasi.
				</h2>
				<p class="mt-4 text-base leading-relaxed text-ink/55">
					Dari catatan harian sampai kekayaan bersih — cepat, rapi, dan tanpa basa-basi.
				</p>
			</div>
			<div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
				{#each features as f, i (f.title)}
					<div
						use:reveal={i * 60}
						class="group rounded-2xl border border-ink/10 bg-ink/4 p-6 backdrop-blur-sm transition-all duration-300 hover:-translate-y-1 hover:border-teal/30 hover:bg-ink/6"
					>
						<div
							class="mb-4 inline-flex h-10 w-10 items-center justify-center rounded-xl border border-teal/25 bg-teal/10 text-teal transition-colors group-hover:bg-teal/15"
						>
							<svg
								class="h-5 w-5"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="1.8"
							>
								<path stroke-linecap="round" stroke-linejoin="round" d={f.icon} />
							</svg>
						</div>
						<p class="font-serif text-[22px] text-ink">{f.title}</p>
						<p class="mt-2 text-[13.5px] leading-relaxed text-ink/55">{f.body}</p>
					</div>
				{/each}
			</div>
		</section>

		<!-- ═══════════ Harga ═══════════ -->
		<section id="harga" class="scroll-mt-24 border-t border-ink/10 py-20">
			<div class="mb-12 max-w-2xl" use:reveal>
				<p class="mb-3 text-[12px] font-semibold tracking-[0.16em] text-teal uppercase">Harga</p>
				<h2 class="font-serif text-4xl leading-tight tracking-tight text-ink sm:text-5xl">
					Mulai gratis. Upgrade saat siap.
				</h2>
				<p class="mt-4 text-base leading-relaxed text-ink/55">
					Tanpa kartu kredit untuk memulai. Batalkan kapan saja.
				</p>
			</div>
			<div class="grid gap-6 lg:grid-cols-3">
				{#each plans as p, i (p.name)}
					<div
						use:reveal={i * 80}
						class="flex flex-col rounded-2xl border p-8 backdrop-blur-sm transition-all duration-300 hover:-translate-y-1 {p.popular
							? 'border-teal/50 bg-ink/6 shadow-[0_0_60px_-15px] shadow-teal/25'
							: 'border-ink/10 bg-ink/4 hover:border-ink/20'}"
					>
						<div class="flex items-center justify-between">
							<p class="font-serif text-2xl text-ink">{p.name}</p>
							{#if p.popular}
								<span
									class="rounded-full border border-teal/30 bg-teal/10 px-2.5 py-0.5 text-[11px] font-semibold text-teal"
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
						<p class="mt-3 text-[13.5px] leading-relaxed text-ink/55">{p.desc}</p>

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
								class="mt-8 rounded-full border border-ink/20 py-3 text-center text-sm font-semibold text-ink transition-all hover:border-ink/40 hover:bg-ink/5 active:scale-95"
							>
								{p.cta}
							</a>
						{:else}
							<a
								href={resolve('/register')}
								class="mt-8 rounded-full py-3 text-center text-sm font-semibold transition-all active:scale-95 {p.popular
									? 'bg-gradient-to-b from-ink via-ink to-ink/70 text-card hover:scale-[1.03]'
									: 'border border-ink/20 text-ink hover:border-ink/40 hover:bg-ink/5'}"
							>
								{p.cta}
							</a>
						{/if}
					</div>
				{/each}
			</div>
		</section>

		<!-- ═══════════ Tentang ═══════════ -->
		<section id="tentang" class="scroll-mt-24 border-t border-ink/10 py-20">
			<div class="grid gap-10 lg:grid-cols-[1fr_1.4fr] lg:gap-16">
				<div use:reveal>
					<p class="mb-3 text-[12px] font-semibold tracking-[0.16em] text-teal uppercase">
						Tentang
					</p>
					<h2 class="font-serif text-4xl leading-tight tracking-tight text-ink sm:text-5xl">
						Tertib dulu, kaya kemudian.
					</h2>
				</div>
				<div class="space-y-5 text-base leading-relaxed text-ink/65" use:reveal={100}>
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

	<!-- ═══════════ Footer ═══════════ -->
	<footer class="border-t border-ink/10">
		<div
			class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-3 px-6 py-6 text-xs text-ink/45 sm:flex-row lg:px-10"
		>
			<span>© 2026 KasKu</span>
			<nav class="flex gap-5">
				<a href="#fitur" class="transition-colors hover:text-teal">Fitur</a>
				<a href="#harga" class="transition-colors hover:text-teal">Harga</a>
				<a href="#tentang" class="transition-colors hover:text-teal">Tentang</a>
				<a href={resolve('/privacy')} class="transition-colors hover:text-teal">Privasi</a>
				<a href={resolve('/terms')} class="transition-colors hover:text-teal">Ketentuan</a>
			</nav>
		</div>
	</footer>
</div>
