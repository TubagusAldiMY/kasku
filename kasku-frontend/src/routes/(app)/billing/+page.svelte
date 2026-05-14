<script lang="ts">
	import { fade, fly } from 'svelte/transition';

	let billingCycle = $state<'monthly' | 'yearly'>('monthly');

	const plans = [
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
			priceMonthly: 99000,
			priceYearly: 999000,
			desc: 'Solusi lengkap untuk keluarga & bisnis.',
			features: [
				'Semua Fitur Pro',
				'Multi-User (5 User)',
				'Analisis Prediktif AI',
				'API Access'
			],
			isPopular: false,
			btnText: 'Pilih Enterprise',
			disabled: false
		}
	];

	function formatPrice(val: number) {
		return new Intl.NumberFormat('id-ID', { 
			style: 'currency', 
			currency: 'IDR', 
			minimumFractionDigits: 0 
		}).format(val);
	}
</script>

<div class="space-y-8 md:space-y-12 animate-in fade-in duration-700 pb-20 px-1">
	<!-- Header -->
	<div class="text-center space-y-4 max-w-2xl mx-auto px-2">
		<h1 class="text-2xl sm:text-4xl font-black text-[#0a2e31] tracking-tight leading-tight">Pilih Paket Finansial Anda</h1>
		<p class="text-gray-500 font-medium text-sm sm:text-lg leading-relaxed">
			Tingkatkan pengalaman mengelola aset dengan fitur premium KasKu.
		</p>
		
		<!-- Toggle Billing Cycle (Fixed & Responsive) -->
		<div class="flex flex-wrap items-center justify-center gap-3 pt-6">
			<span class="text-[12px] sm:text-sm font-bold {billingCycle === 'monthly' ? 'text-[#0a2e31]' : 'text-gray-400'} transition-colors">Bulanan</span>
			<button 
				type="button"
				onclick={() => billingCycle = (billingCycle === 'monthly' ? 'yearly' : 'monthly')}
				class="w-12 h-6 sm:w-14 sm:h-7 bg-gray-100 rounded-full p-1 relative transition-all border border-gray-200 focus:outline-none"
				aria-label="Toggle billing cycle"
			>
				<div class="w-4 h-4 sm:w-5 sm:h-5 bg-[#217b84] rounded-full shadow-md transition-transform duration-300 {billingCycle === 'yearly' ? 'translate-x-6 sm:translate-x-7' : 'translate-x-0'}"></div>
			</button>
			<div class="flex items-center gap-2">
				<span class="text-[12px] sm:text-sm font-bold {billingCycle === 'yearly' ? 'text-[#0a2e31]' : 'text-gray-400'} transition-colors">Tahunan</span>
				<span class="text-[9px] sm:text-[10px] bg-green-100 text-green-700 px-2 py-0.5 rounded-full font-black uppercase tracking-tighter">Hemat 15%</span>
			</div>
		</div>
	</div>

	<!-- Pricing Cards (Grid Responsive) -->
	<div class="grid grid-cols-1 md:grid-cols-3 gap-6 lg:gap-8 items-end max-w-6xl mx-auto">
		{#each plans as plan}
			<div 
				class="relative bg-white rounded-[2rem] sm:rounded-[2.5rem] border transition-all duration-500 hover:shadow-2xl flex flex-col {plan.isPopular ? 'p-6 sm:p-10 border-[#217b84] shadow-xl shadow-teal-900/5 lg:scale-105 z-10' : 'p-6 sm:p-8 border-gray-100 shadow-sm'}"
			>
				{#if plan.isPopular}
					<div class="absolute -top-4 left-1/2 -translate-x-1/2 bg-[#217b84] text-white text-[9px] sm:text-[10px] font-black uppercase tracking-[0.2em] px-5 py-2 rounded-full shadow-lg whitespace-nowrap">
						Paling Populer
					</div>
				{/if}

				<div class="space-y-2 mb-6 sm:mb-8">
					<h3 class="text-lg sm:text-xl font-black text-[#0a2e31]">{plan.name}</h3>
					<p class="text-[11px] sm:text-xs text-gray-500 font-medium leading-relaxed">{plan.desc}</p>
				</div>

				<div class="mb-8 sm:mb-10">
					<div class="flex items-baseline gap-1 flex-wrap">
						<span class="text-2xl sm:text-4xl font-black text-[#0a2e31]">
							{formatPrice(billingCycle === 'monthly' ? plan.priceMonthly : plan.priceYearly)}
						</span>
						<span class="text-xs sm:text-sm font-bold text-gray-400">/{billingCycle === 'monthly' ? 'bln' : 'thn'}</span>
					</div>
				</div>

				<div class="space-y-3 sm:space-y-4 mb-8 sm:mb-10 flex-1">
					{#each plan.features as feature}
						<div class="flex items-start gap-3">
							<div class="h-4 w-4 sm:h-5 sm:w-5 rounded-full bg-teal-50 flex items-center justify-center text-teal-600 mt-0.5 flex-shrink-0">
								<svg class="h-2.5 w-2.5 sm:h-3 sm:w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="4"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" /></svg>
							</div>
							<span class="text-[12px] sm:text-sm font-medium text-gray-600 leading-tight">{feature}</span>
						</div>
					{/each}
				</div>

				<button 
					disabled={plan.disabled}
					class="w-full py-3.5 sm:py-4 rounded-xl sm:rounded-2xl font-black text-[11px] sm:text-sm uppercase tracking-widest transition-all active:scale-[0.98] {plan.isPopular ? 'bg-[#217b84] hover:bg-[#1a5f66] text-white shadow-xl shadow-teal-900/20' : 'bg-gray-50 hover:bg-gray-100 text-[#0a2e31] border border-gray-100'} disabled:opacity-50 disabled:cursor-default"
				>
					{plan.btnText}
				</button>
			</div>
		{/each}
	</div>

	<!-- FAQ Subtle Section -->
	<div class="mt-12 sm:mt-20 pt-12 sm:pt-20 border-t border-gray-100">
		<div class="grid grid-cols-1 md:grid-cols-2 gap-8 sm:gap-12 max-w-4xl mx-auto px-4">
			<div class="space-y-2">
				<h4 class="text-sm sm:text-base font-bold text-[#0a2e31]">Dapatkah saya membatalkan kapan saja?</h4>
				<p class="text-xs sm:text-sm text-gray-500 leading-relaxed font-medium">Ya, Anda dapat membatalkan langganan kapan pun Anda mau tanpa biaya tambahan.</p>
			</div>
			<div class="space-y-2">
				<h4 class="text-sm sm:text-base font-bold text-[#0a2e31]">Bagaimana dengan keamanan data saya?</h4>
				<p class="text-xs sm:text-sm text-gray-500 leading-relaxed font-medium">Kami menggunakan enkripsi tingkat bank untuk memastikan data Anda terisolasi secara aman.</p>
			</div>
		</div>
	</div>
</div>
