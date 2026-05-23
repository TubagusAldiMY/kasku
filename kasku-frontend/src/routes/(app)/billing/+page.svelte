<script lang="ts">
	import { fly, fade } from 'svelte/transition';
	import { apiFetch } from '$lib/api/client';
	import { onMount } from 'svelte';
	import QRCode from 'qrcode';

	type BackendPlan = { id: string; name: string; price_idr: number };
	type Subscription = {
		plan_id: string;
		status?: string;
		current_period_end?: string | null;
		current_period_start?: string | null;
	};
	type Plan = {
		id: string;
		name: string;
		priceMonthly: number;
		priceYearly: number;
		desc: string;
		features: string[];
		isPopular: boolean;
		btnText: string;
		disabled: boolean;
	};
	type QRPayment = {
		qr_string: string;
		amount_idr: number;
		expires_at: string | null | undefined;
		order_id: string;
	};

	let billingCycle = $state<'monthly' | 'yearly'>('monthly');
	let loading = $state(true);
	let subscribing = $state<string | null>(null);
	let currentSub = $state<Subscription | null>(null);
	let message = $state<{ tone: 'info' | 'error' | 'success'; text: string } | null>(null);
	let qrModal = $state<QRPayment | null>(null);
	let qrDataURL = $state('');
	let qrSecondsLeft = $state(0);

	// Generate QR image data URL saat modal terbuka
	$effect(() => {
		if (qrModal?.qr_string) {
			QRCode.toDataURL(qrModal.qr_string, {
				width: 240,
				margin: 2,
				color: { dark: '#0a2e31', light: '#ffffff' }
			}).then((url) => {
				qrDataURL = url;
			});
		} else {
			qrDataURL = '';
		}
	});

	// Countdown timer menuju expires_at
	$effect(() => {
		if (!qrModal?.expires_at) {
			qrSecondsLeft = 0;
			return;
		}
		const target = new Date(qrModal.expires_at).getTime();
		const tick = () => {
			qrSecondsLeft = Math.max(0, Math.floor((target - Date.now()) / 1000));
		};
		tick();
		const timer = setInterval(tick, 1000);
		return () => clearInterval(timer);
	});

	// Poll subscription setiap 3 detik saat QR modal terbuka — tutup otomatis jika sudah aktif
	$effect(() => {
		if (!qrModal) return;
		const interval = setInterval(async () => {
			try {
				const res = await apiFetch('/billing/subscription');
				const data = await res.json();
				if (data.success && data.data?.status === 'ACTIVE') {
					currentSub = data.data;
					qrModal = null;
					message = { tone: 'success', text: 'Pembayaran berhasil! Langganan Anda telah diaktifkan.' };
				}
			} catch {
				// polling — abaikan error sementara
			}
		}, 3000);
		return () => clearInterval(interval);
	});

	function formatDate(iso?: string | null) {
		if (!iso) return '—';
		try {
			return new Date(iso).toLocaleDateString('id-ID', {
				day: 'numeric',
				month: 'long',
				year: 'numeric'
			});
		} catch {
			return iso;
		}
	}

	let plans = $state<Plan[]>([
		{
			id: 'FREE',
			name: 'Gratis',
			priceMonthly: 0,
			priceYearly: 0,
			desc: 'Cocok untuk individu yang baru mulai mencatat.',
			features: [
				'Maks. 50 Transaksi / Bulan',
				'Maks. 3 Rekening Keuangan',
				'Riwayat 3 Bulan Terakhir',
				'Akses Web & PWA'
			],
			isPopular: false,
			btnText: 'Paket Saat Ini',
			disabled: true
		},
		{
			id: 'PRO',
			name: 'Pro',
			priceMonthly: 29000,
			priceYearly: 299000,
			desc: 'Untuk pengelolaan aset yang lebih serius.',
			features: [
				'Transaksi Tak Terbatas',
				'Rekening Tak Terbatas',
				'Riwayat Selamanya',
				'Ekspor PDF & CSV',
				'Grafik Analisis Detail'
			],
			isPopular: true,
			btnText: 'Upgrade ke Pro',
			disabled: false
		},
		{
			id: 'ULTIMATE',
			name: 'Enterprise',
			priceMonthly: 0,
			priceYearly: 0,
			desc: 'Solusi lengkap untuk tim, keluarga & bisnis skala besar.',
			features: ['Semua Fitur Pro', 'Multi-User (5 User)', 'Analisis Prediktif AI', 'API Access', 'Dukungan Prioritas'],
			isPopular: false,
			btnText: 'Hubungi Tim KasKu',
			disabled: false
		}
	]);

	async function fetchBillingData() {
		loading = true;
		try {
			const [plansRes, subRes] = await Promise.all([
				apiFetch('/billing/plans'),
				apiFetch('/billing/subscription')
			]);

			const plansData = await plansRes.json();
			const subData = await subRes.json();

			if (plansData.success && plansData.data.length > 0) {
				// Update id (UUID) dan harga dari backend — id frontend adalah slug fallback,
				// subscribe endpoint memerlukan UUID dari database.
				plans = plans.map((p) => {
					const backendPlan = plansData.data.find(
						(bp: BackendPlan) => bp.name.toUpperCase() === p.name.toUpperCase()
					);
					if (backendPlan) {
						return {
							...p,
							id: backendPlan.id, // UUID dari database — wajib untuk POST /subscribe
							priceMonthly: backendPlan.price_idr,
							priceYearly: Math.round(backendPlan.price_idr * 10)
						};
					}
					// Plan tidak ada di backend (mis. ULTIMATE/Enterprise = contact sales) → tetap disabled
					return { ...p, disabled: true };
				});
			}

			if (subData.success) {
				currentSub = subData.data;
			}
		} catch (err) {
			console.error('Gagal memuat data billing:', err);
		} finally {
			loading = false;
		}
	}

	async function handleSubscribe(planId: string) {
		message = null;
		subscribing = planId;
		try {
			const res = await apiFetch('/billing/subscribe', {
				method: 'POST',
				body: JSON.stringify({ plan_id: planId, billing_cycle: billingCycle })
			});
			const result = (await res.json()) as {
				success: boolean;
				data?: {
					payment_url?: string;
					qr_string?: string;
					amount_idr?: number;
					expires_at?: string | null;
					order_id?: string;
				};
				error?: { code?: string; message?: string };
			};

			if (result.success && result.data?.qr_string) {
				// QRIS: tampilkan QR code untuk di-scan user
				qrModal = {
					qr_string: result.data.qr_string,
					amount_idr: result.data.amount_idr ?? 0,
					expires_at: result.data.expires_at,
					order_id: result.data.order_id ?? ''
				};
				return;
			}

			if (result.success && result.data?.payment_url) {
				// Non-QRIS (VA, e-wallet): redirect ke halaman pembayaran
				window.location.href = result.data.payment_url;
				return;
			}

			message = {
				tone: 'error',
				text: result.error?.message ?? 'Gagal memulai langganan. Silakan coba lagi.'
			};
		} catch {
			message = { tone: 'error', text: 'Terjadi kesalahan koneksi.' };
		} finally {
			subscribing = null;
		}
	}

	onMount(fetchBillingData);

	function formatPrice(val: number) {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(val);
	}

	function formatCountdown(seconds: number): string {
		if (seconds <= 0) return 'Kedaluwarsa';
		const m = Math.floor(seconds / 60).toString().padStart(2, '0');
		const s = (seconds % 60).toString().padStart(2, '0');
		return `${m}:${s}`;
	}
</script>

{#if qrModal}
	<!-- QR Payment Modal -->
	<div
		role="dialog"
		aria-modal="true"
		aria-label="Bayar dengan QRIS"
		class="fixed inset-0 z-50 flex items-center justify-center p-4"
		transition:fade={{ duration: 200 }}
	>
		<!-- Backdrop -->
		<button
			type="button"
			class="absolute inset-0 bg-black/50 backdrop-blur-sm"
			onclick={() => (qrModal = null)}
			aria-label="Tutup modal"
		></button>

		<!-- Modal Card -->
		<div
			class="relative z-10 w-full max-w-sm rounded-3xl bg-white p-6 shadow-2xl sm:p-8"
			transition:fly={{ y: 20, duration: 250 }}
		>
			<!-- Header -->
			<div class="mb-5 flex items-start justify-between">
				<div>
					<p class="text-[10px] font-black tracking-widest text-teal-600 uppercase">Bayar dengan QRIS</p>
					<p class="mt-0.5 text-xl font-black text-[#0a2e31]">{formatPrice(qrModal.amount_idr)}</p>
				</div>
				<button
					type="button"
					onclick={() => (qrModal = null)}
					class="rounded-xl p-2 text-gray-400 transition hover:bg-gray-100 hover:text-gray-700"
					aria-label="Tutup"
				>
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<!-- QR Code -->
			<div class="mb-5 flex justify-center">
				{#if qrDataURL}
					<div class="rounded-2xl border border-gray-100 bg-white p-3 shadow-inner">
						<img src={qrDataURL} alt="QR Code QRIS" class="h-52 w-52 rounded-xl" />
					</div>
				{:else}
					<div class="flex h-52 w-52 items-center justify-center rounded-2xl border border-gray-100 bg-gray-50">
						<div class="h-8 w-8 animate-spin rounded-full border-2 border-teal-600 border-t-transparent"></div>
					</div>
				{/if}
			</div>

			<!-- Countdown -->
			<div class="mb-5 text-center">
				{#if qrSecondsLeft > 0}
					<p class="text-xs font-medium text-gray-500">Bayar sebelum</p>
					<p class="text-2xl font-black tabular-nums {qrSecondsLeft < 60 ? 'text-red-600' : 'text-[#0a2e31]'}">
						{formatCountdown(qrSecondsLeft)}
					</p>
				{:else if qrModal.expires_at}
					<p class="text-sm font-bold text-red-600">QR Code kedaluwarsa</p>
				{/if}
			</div>

			<!-- Instruksi -->
			<ol class="mb-5 space-y-1.5 text-left">
				{#each ['Buka aplikasi mobile banking / e-wallet Anda', 'Pilih menu Scan QR / QRIS', 'Arahkan kamera ke QR di atas', 'Konfirmasi pembayaran'] as step, i}
					<li class="flex items-center gap-2.5 text-xs font-medium text-gray-600">
						<span class="flex h-4 w-4 flex-shrink-0 items-center justify-center rounded-full bg-teal-50 text-[9px] font-black text-teal-700">{i + 1}</span>
						{step}
					</li>
				{/each}
			</ol>

			<!-- Status hint -->
			<p class="text-center text-[10px] font-medium text-gray-400">
				Halaman akan otomatis diperbarui setelah pembayaran berhasil
			</p>
		</div>
	</div>
{/if}

<div class="animate-in fade-in space-y-8 px-1 pb-20 duration-700 md:space-y-12">
	<!-- Header -->
	<div class="mx-auto max-w-2xl space-y-4 px-2 text-center">
		<h1 class="text-2xl leading-tight font-black tracking-tight text-[#0a2e31] sm:text-4xl">
			Pilih Paket Finansial Anda
		</h1>
		<p class="text-sm leading-relaxed font-medium text-gray-500 sm:text-lg">
			Tingkatkan pengalaman mengelola aset dengan fitur premium KasKu.
		</p>

		{#if message}
			{@const tone = message.tone}
			<div
				in:fly={{ y: -10 }}
				class="rounded-2xl border p-4 text-left text-sm font-bold
				{tone === 'error'
					? 'border-red-100 bg-red-50 text-red-700'
					: tone === 'success'
						? 'border-green-100 bg-green-50 text-green-700'
						: 'border-amber-100 bg-amber-50 text-amber-800'}"
			>
				{message.text}
			</div>
		{/if}

		{#if currentSub && currentSub.status === 'ACTIVE'}
			{@const activePlan = plans.find((p) => p.id === currentSub!.plan_id)}
			<div
				class="rounded-2xl border border-teal-100 bg-teal-50 p-4 text-left text-xs font-bold text-teal-800"
			>
				<div class="flex items-center justify-between gap-3">
					<div class="space-y-1">
						<p class="text-[10px] font-black tracking-widest text-teal-600 uppercase">
							Langganan Aktif
						</p>
						<p class="text-sm font-black text-teal-900">{activePlan?.name ?? currentSub.plan_id}</p>
					</div>
					<div class="space-y-1 text-right">
						<p class="text-[10px] font-black tracking-widest text-teal-600 uppercase">
							Aktif Hingga
						</p>
						<p class="text-sm font-bold text-teal-900">{formatDate(currentSub.current_period_end)}</p>
					</div>
				</div>
			</div>
		{/if}

		<!-- Toggle Billing Cycle (Fixed & Responsive) -->
		<div class="flex flex-wrap items-center justify-center gap-3 pt-6">
			<span
				class="text-[12px] font-bold sm:text-sm {billingCycle === 'monthly'
					? 'text-[#0a2e31]'
					: 'text-gray-400'} transition-colors">Bulanan</span
			>
			<button
				type="button"
				onclick={() => (billingCycle = billingCycle === 'monthly' ? 'yearly' : 'monthly')}
				class="relative h-6 w-12 rounded-full border border-gray-200 bg-gray-100 p-1 transition-all focus:outline-none sm:h-7 sm:w-14"
				aria-label="Toggle billing cycle"
			>
				<div
					class="h-4 w-4 rounded-full bg-[#217b84] shadow-md transition-transform duration-300 sm:h-5 sm:w-5 {billingCycle ===
					'yearly'
						? 'translate-x-6 sm:translate-x-7'
						: 'translate-x-0'}"
				></div>
			</button>
			<div class="flex items-center gap-2">
				<span
					class="text-[12px] font-bold sm:text-sm {billingCycle === 'yearly'
						? 'text-[#0a2e31]'
						: 'text-gray-400'} transition-colors">Tahunan</span
				>
				<span
					class="rounded-full bg-green-100 px-2 py-0.5 text-[9px] font-black tracking-tighter text-green-700 uppercase sm:text-[10px]"
					>Hemat 15%</span
				>
			</div>
		</div>
	</div>

	<!-- Pricing Cards (Grid Responsive) -->
	<div class="mx-auto grid max-w-6xl grid-cols-1 items-end gap-6 md:grid-cols-3 lg:gap-8">
		{#each plans as plan (plan.id)}
			{@const isCurrent =
				// Bandingkan UUID-to-UUID setelah fetchBillingData mengisi plan.id dari backend.
				// Sebelum fetch selesai currentSub masih null, sehingga ekspresi ini aman.
				currentSub !== null && currentSub.plan_id === plan.id}
			<div
				class="relative flex flex-col rounded-[2rem] border bg-white transition-all duration-500 hover:shadow-2xl sm:rounded-[2.5rem] {plan.isPopular
					? 'z-10 border-[#217b84] p-6 shadow-xl shadow-teal-900/5 sm:p-10 lg:scale-105'
					: 'border-gray-100 p-6 shadow-sm sm:p-8'}"
			>
				{#if plan.isPopular}
					<div
						class="absolute -top-4 left-1/2 -translate-x-1/2 rounded-full bg-[#217b84] px-5 py-2 text-[9px] font-black tracking-[0.2em] whitespace-nowrap text-white uppercase shadow-lg sm:text-[10px]"
					>
						Paling Populer
					</div>
				{/if}

				<div class="mb-6 space-y-2 sm:mb-8">
					<div class="flex items-center justify-between">
						<h3 class="text-lg font-black text-[#0a2e31] sm:text-xl">{plan.name}</h3>
						{#if isCurrent}
							<span
								class="rounded-lg border border-green-100 bg-green-50 px-2 py-1 text-[9px] font-black tracking-widest text-green-600 uppercase"
								>Aktif</span
							>
						{/if}
					</div>
					<p class="text-[11px] leading-relaxed font-medium text-gray-500 sm:text-xs">
						{plan.desc}
					</p>
				</div>

				<div class="mb-8 sm:mb-10">
					{#if plan.id === 'ULTIMATE'}
						<div class="space-y-1">
							<p class="text-2xl font-black text-[#0a2e31] sm:text-3xl">Harga Khusus</p>
							<p class="text-[11px] font-medium text-gray-400 sm:text-xs">
								Sesuai kebutuhan tim Anda
							</p>
						</div>
					{:else}
						<div class="flex flex-wrap items-baseline gap-1">
							<span class="text-2xl font-black text-[#0a2e31] sm:text-4xl">
								{formatPrice(billingCycle === 'monthly' ? plan.priceMonthly : plan.priceYearly)}
							</span>
							<span class="text-xs font-bold text-gray-400 sm:text-sm"
								>/{billingCycle === 'monthly' ? 'bln' : 'thn'}</span
							>
						</div>
					{/if}
				</div>

				<div class="mb-8 flex-1 space-y-3 sm:mb-10 sm:space-y-4">
					{#each plan.features as feature (feature)}
						<div class="flex items-start gap-3">
							<div
								class="mt-0.5 flex h-4 w-4 flex-shrink-0 items-center justify-center rounded-full bg-teal-50 text-teal-600 sm:h-5 sm:w-5"
							>
								<svg
									class="h-2.5 w-2.5 sm:h-3 sm:w-3"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="4"
									><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" /></svg
								>
							</div>
							<span class="text-[12px] leading-tight font-medium text-gray-600 sm:text-sm"
								>{feature}</span
							>
						</div>
					{/each}
				</div>

				{#if plan.id === 'ULTIMATE'}
					<a
						href="mailto:admin@tubsamy.tech?subject=Pertanyaan%20Paket%20Enterprise%20KasKu"
						class="flex w-full items-center justify-center gap-2 rounded-xl border border-[#0a2e31] bg-[#0a2e31] py-3.5 text-[11px] font-black tracking-widest text-white uppercase transition-all hover:bg-[#0d3b3f] active:scale-[0.98] sm:rounded-2xl sm:py-4 sm:text-sm"
					>
						<svg class="h-3.5 w-3.5 sm:h-4 sm:w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
						</svg>
						Hubungi Tim KasKu
					</a>
				{:else}
					<button
						onclick={() => handleSubscribe(plan.id)}
						disabled={loading || isCurrent || plan.disabled || subscribing !== null}
						class="w-full rounded-xl py-3.5 text-[11px] font-black tracking-widest uppercase transition-all active:scale-[0.98] sm:rounded-2xl sm:py-4 sm:text-sm {plan.isPopular
							? 'bg-[#217b84] text-white shadow-xl shadow-teal-900/20 hover:bg-[#1a5f66]'
							: 'border border-gray-100 bg-gray-50 text-[#0a2e31] hover:bg-gray-100'} disabled:cursor-default disabled:opacity-50"
					>
						{#if subscribing === plan.id}
							Menyiapkan…
						{:else if isCurrent}
							Paket Saat Ini
						{:else}
							{plan.btnText}
						{/if}
					</button>
				{/if}
			</div>
		{/each}
	</div>

	<!-- FAQ Subtle Section -->
	<div class="mt-12 border-t border-gray-100 pt-12 sm:mt-20 sm:pt-20">
		<div class="mx-auto grid max-w-4xl grid-cols-1 gap-8 px-4 sm:gap-12 md:grid-cols-2">
			<div class="space-y-2">
				<h4 class="text-sm font-bold text-[#0a2e31] sm:text-base">
					Dapatkah saya membatalkan kapan saja?
				</h4>
				<p class="text-xs leading-relaxed font-medium text-gray-500 sm:text-sm">
					Ya, Anda dapat membatalkan langganan kapan pun Anda mau tanpa biaya tambahan.
				</p>
			</div>
			<div class="space-y-2">
				<h4 class="text-sm font-bold text-[#0a2e31] sm:text-base">
					Bagaimana dengan keamanan data saya?
				</h4>
				<p class="text-xs leading-relaxed font-medium text-gray-500 sm:text-sm">
					Kami menggunakan enkripsi tingkat bank untuk memastikan data Anda terisolasi secara aman.
				</p>
			</div>
		</div>
	</div>
</div>
