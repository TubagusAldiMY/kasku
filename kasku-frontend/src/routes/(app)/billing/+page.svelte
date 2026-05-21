<script lang="ts">
	import { fly } from 'svelte/transition';
	import { apiFetch } from '$lib/api/client';
	import { onMount } from 'svelte';

	type BackendPlan = { name: string; price_idr: number };
	type Subscription = {
		plan_id: string;
		status?: string;
		expires_at?: string | null;
		started_at?: string | null;
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

	let billingCycle = $state<'monthly' | 'yearly'>('monthly');
	let loading = $state(true);
	let subscribing = $state<string | null>(null);
	let currentSub = $state<Subscription | null>(null);
	let message = $state<{ tone: 'info' | 'error' | 'success'; text: string } | null>(null);

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
				// Update prices if backend has them
				plans = plans.map((p) => {
					const backendPlan = plansData.data.find(
						(bp: BackendPlan) => bp.name.toUpperCase() === p.name.toUpperCase()
					);
					if (backendPlan) {
						return {
							...p,
							priceMonthly: backendPlan.price_idr,
							priceYearly: backendPlan.price_idr * 10
						}; // simplified
					}
					return p;
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
				body: JSON.stringify({ plan_id: planId })
			});
			const result = (await res.json()) as {
				success: boolean;
				data?: { snap_url?: string; redirect_url?: string };
				error?: { code?: string; message?: string };
			};

			if (result.success && (result.data?.snap_url || result.data?.redirect_url)) {
				const target = result.data.snap_url ?? result.data.redirect_url;
				if (target) {
					window.location.href = target;
					return;
				}
			}

			if (result.error?.code === 'COMING_SOON') {
				message = {
					tone: 'info',
					text: 'Pembayaran Midtrans masih dalam tahap pengerjaan akhir. Sementara, hubungi admin@tubsamy.tech untuk aktivasi manual.'
				};
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
</script>

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

		{#if currentSub && currentSub.plan_id && !currentSub.plan_id.includes('free')}
			<div
				class="rounded-2xl border border-teal-100 bg-teal-50 p-4 text-left text-xs font-bold text-teal-800"
			>
				<div class="flex items-center justify-between gap-3">
					<div class="space-y-1">
						<p class="text-[10px] font-black tracking-widest text-teal-600 uppercase">
							Langganan Aktif
						</p>
						<p class="text-sm font-black text-teal-900">{currentSub.plan_id}</p>
					</div>
					<div class="space-y-1 text-right">
						<p class="text-[10px] font-black tracking-widest text-teal-600 uppercase">
							Aktif Hingga
						</p>
						<p class="text-sm font-bold text-teal-900">{formatDate(currentSub.expires_at)}</p>
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
				currentSub &&
				(plan.id === 'FREE'
					? currentSub.plan_id.includes('free')
					: currentSub.plan_id.toUpperCase().includes(plan.id))}
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
						disabled={loading || isCurrent || plan.id === 'FREE' || subscribing !== null}
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
