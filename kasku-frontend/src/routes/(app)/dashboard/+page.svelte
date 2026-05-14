<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { fade, fly } from 'svelte/transition';

	let stats = $state({
		totalBalance: 0,
		monthlyIncome: 0,
		monthlyExpense: 0,
		savingsRate: 0
	});

	let recentTransactions = $state<any[]>([]);
	let accounts = $state<any[]>([]);
	let loading = $state(true);

	let weeklyData = $state([
		{ day: 'Sen', amount: 450000 },
		{ day: 'Sel', amount: 1200000 },
		{ day: 'Rab', amount: 300000 },
		{ day: 'Kam', amount: 850000 },
		{ day: 'Jum', amount: 2100000 },
		{ day: 'Sab', amount: 950000 },
		{ day: 'Min', amount: 500000 }
	]);

	let maxAmount = $derived(Math.max(...weeklyData.map((d) => d.amount)));

	async function loadDashboardData() {
		loading = true;
		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';

		if (isMock) {
			// Simulasi data dashboard
			setTimeout(() => {
				stats = {
					totalBalance: 128450000,
					monthlyIncome: 15000000,
					monthlyExpense: 4250000,
					savingsRate: 71
				};
				recentTransactions = [
					{ id: '1', title: 'Kopi Kenangan', category: 'Makanan', amount: -25000, date: 'Hari ini' },
					{ id: '2', title: 'Gaji Mei', category: 'Gaji', amount: 15000000, date: 'Kemarin' },
					{ id: '3', title: 'Listrik Token', category: 'Tagihan', amount: -500000, date: '9 Mei' }
				];
				accounts = [
					{ name: 'BCA Utama', balance: 85000000, color: 'bg-blue-600' },
					{ name: 'Mandiri', balance: 40450000, color: 'bg-orange-500' },
					{ name: 'Dompet Jajan', balance: 3000000, color: 'bg-teal-500' }
				];
				loading = false;
			}, 800);
			return;
		}

		try {
			// Integrasi real nantinya ke aggregator endpoint dashboard
			const res = await apiFetch('/dashboard/summary');
			const result = await res.json();
			if (result.success) {
				stats = result.data.stats;
				recentTransactions = result.data.recent;
			}
		} catch (e) {
			console.error(e);
		} finally {
			loading = false;
		}
	}

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(val);
	}

	onMount(loadDashboardData);
</script>

<div class="space-y-10 animate-in fade-in duration-700">
	<!-- Top Section: Welcome & Quick Stats -->
	<div class="flex flex-col md:flex-row md:items-end justify-between gap-6">
		<div class="space-y-1">
			<h1 class="text-3xl font-black text-[#0a2e31]">Halo, Juragan!</h1>
			<p class="text-gray-500 font-medium">Berikut adalah ringkasan finansial Anda bulan ini.</p>
		</div>
		<div class="flex gap-3">
			<a href="/transactions" class="px-5 py-2.5 bg-white border border-gray-200 rounded-xl text-sm font-bold text-[#0a2e31] hover:bg-gray-50 transition-all shadow-sm">Riwayat</a>
			<button class="px-5 py-2.5 bg-[#217b84] text-white rounded-xl text-sm font-bold shadow-lg shadow-teal-900/10 hover:bg-[#1a5f66] transition-all">Laporan</button>
		</div>
	</div>

	<!-- Hero Cards -->
	<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
		<!-- Main Balance Card -->
		<div class="lg:col-span-2 relative overflow-hidden bg-[#0a2e31] rounded-[2.5rem] p-10 text-white shadow-2xl">
			<div class="absolute top-0 right-0 p-8 opacity-10">
				<svg class="h-32 w-32" fill="currentColor" viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1.75 14.83h-3.5v-1.45c-1.43-.31-2.47-1.12-2.57-2.43h1.49c.09.68.74 1.19 1.48 1.19.8 0 1.42-.42 1.42-1.16 0-.61-.31-1.01-1.63-1.35-1.57-.41-2.61-1-2.61-2.53 0-1.28.97-2.18 2.42-2.49V7.12h3.5v1.44c1.23.27 2.1 1.01 2.24 2.15h-1.47c-.12-.66-.66-1.07-1.32-1.07-.73 0-1.25.4-1.25 1.05 0 .61.46.91 1.76 1.3 1.54.45 2.49 1.07 2.49 2.56 0 1.25-.9 2.18-2.47 2.52v1.41z"/></svg>
			</div>
			
			<div class="relative z-10 space-y-6">
				<div class="space-y-1">
					<span class="text-[11px] font-bold uppercase tracking-[0.2em] text-teal-400">Total Net Worth</span>
					<div class="text-5xl font-black tracking-tighter">
						{loading ? '...' : formatCurrency(stats.totalBalance)}
					</div>
				</div>
				
				<div class="pt-4 flex items-center gap-6">
					<div class="flex items-center gap-2">
						<div class="h-10 w-10 rounded-2xl bg-white/10 flex items-center justify-center text-green-400">
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 10l7-7m0 0l7 7m-7-7v18" /></svg>
						</div>
						<div>
							<p class="text-[10px] font-bold uppercase text-white/40 tracking-wider">Income</p>
							<p class="text-sm font-bold text-white">{formatCurrency(stats.monthlyIncome)}</p>
						</div>
					</div>
					<div class="flex items-center gap-2">
						<div class="h-10 w-10 rounded-2xl bg-white/10 flex items-center justify-center text-red-400">
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M19 14l-7 7m0 0l-7-7m7 7V3" /></svg>
						</div>
						<div>
							<p class="text-[10px] font-bold uppercase text-white/40 tracking-wider">Expense</p>
							<p class="text-sm font-bold text-white">{formatCurrency(stats.monthlyExpense)}</p>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Savings Card -->
		<div class="bg-white rounded-[2.5rem] p-10 border border-gray-100 shadow-sm flex flex-col justify-between">
			<div class="space-y-4">
				<span class="text-[11px] font-bold uppercase tracking-[0.2em] text-gray-400">Savings Rate</span>
				<div class="relative h-32 w-32 mx-auto">
					<!-- Simple Circle Progress -->
					<svg class="h-full w-full transform -rotate-90">
						<circle cx="64" cy="64" r="58" stroke="currentColor" stroke-width="12" fill="transparent" class="text-gray-50" />
						<circle cx="64" cy="64" r="58" stroke="currentColor" stroke-width="12" fill="transparent" stroke-dasharray="364.4" stroke-dashoffset={364.4 - (364.4 * stats.savingsRate / 100)} class="text-[#217b84] transition-all duration-1000" />
					</svg>
					<div class="absolute inset-0 flex items-center justify-center">
						<span class="text-3xl font-black text-[#0a2e31]">{stats.savingsRate}%</span>
					</div>
				</div>
			</div>
			<p class="text-xs text-center text-gray-400 font-medium leading-relaxed">
				Anda menyisihkan <span class="text-[#217b84] font-bold">{stats.savingsRate}%</span> pendapatan bulan ini. Pertahankan!
			</p>
		</div>
	</div>

	<!-- Analytics Section: Expense Chart -->
	<div class="bg-white rounded-[2.5rem] p-10 border border-gray-100 shadow-sm space-y-8">
		<div class="flex items-center justify-between">
			<div class="space-y-1">
				<h3 class="text-xl font-bold text-[#0a2e31]">Pengeluaran 7 Hari Terakhir</h3>
				<p class="text-xs text-gray-400 font-bold uppercase tracking-wider">Statistik Mingguan</p>
			</div>
			<div class="text-right">
				<p class="text-sm font-bold text-red-500">Puncak: {formatCurrency(maxAmount)}</p>
			</div>
		</div>

		<div class="relative h-64 w-full flex items-end justify-between gap-4 px-2">
			{#each weeklyData as data}
				<div class="flex-1 flex flex-col items-center gap-4 group">
					<!-- Tooltip -->
					<div class="opacity-0 group-hover:opacity-100 transition-opacity absolute bottom-full mb-2 bg-[#0a2e31] text-white text-[10px] font-bold py-1.5 px-3 rounded-lg shadow-xl pointer-events-none whitespace-nowrap">
						{formatCurrency(data.amount)}
					</div>
					
					<!-- Bar -->
					<div 
						class="w-full max-w-[40px] bg-gray-50 rounded-2xl relative overflow-hidden transition-all duration-500 group-hover:shadow-lg"
						style="height: {(data.amount / maxAmount) * 100}%"
					>
						<div class="absolute inset-0 bg-gradient-to-t from-[#217b84] to-[#4fd1c5] opacity-80 group-hover:opacity-100 transition-opacity"></div>
					</div>
					
					<!-- Day Label -->
					<span class="text-[11px] font-black text-gray-400 uppercase tracking-tighter group-hover:text-[#0a2e31] transition-colors">{data.day}</span>
				</div>
			{/each}
			
			<!-- Background Grid Lines -->
			<div class="absolute inset-0 -z-10 flex flex-col justify-between py-10 opacity-[0.03]">
				<div class="w-full border-t border-black"></div>
				<div class="w-full border-t border-black"></div>
				<div class="w-full border-t border-black"></div>
			</div>
		</div>
	</div>

	<!-- Bottom Grid: Recent Transactions & Accounts -->
	<div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
		<!-- Recent Transactions -->
		<div class="lg:col-span-2 space-y-6">
			<div class="flex items-center justify-between px-2">
				<h3 class="text-xl font-bold text-[#0a2e31]">Aktivitas Terakhir</h3>
				<a href="/transactions" class="text-xs font-bold text-[#217b84] hover:underline">Lihat Semua</a>
			</div>
			
			<div class="bg-white rounded-[2.5rem] border border-gray-100 shadow-sm overflow-hidden">
				{#if loading}
					<div class="p-8 space-y-4">
						{#each Array(3) as _}
							<div class="h-14 bg-gray-50 rounded-2xl animate-pulse"></div>
						{/each}
					</div>
				{:else}
					<div class="divide-y divide-gray-50">
						{#each recentTransactions as tx}
							<div class="flex items-center justify-between p-6 hover:bg-gray-50 transition-colors">
								<div class="flex items-center gap-4">
									<div class="h-12 w-12 rounded-2xl bg-gray-50 flex items-center justify-center text-[#0a2e31] font-bold text-xs uppercase">
										{tx.category.substring(0, 2)}
									</div>
									<div>
										<p class="font-bold text-[#0a2e31]">{tx.title}</p>
										<p class="text-[10px] text-gray-400 font-bold uppercase tracking-wider">{tx.category} • {tx.date}</p>
									</div>
								</div>
								<p class="font-black tracking-tight {tx.amount > 0 ? 'text-green-600' : 'text-[#0a2e31]'}">
									{formatCurrency(tx.amount)}
								</p>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>

		<!-- Accounts Summary -->
		<div class="space-y-6">
			<div class="flex items-center justify-between px-2">
				<h3 class="text-xl font-bold text-[#0a2e31]">Rekening</h3>
				<a href="/accounts" class="text-xs font-bold text-[#217b84] hover:underline">Kelola</a>
			</div>

			<div class="space-y-4">
				{#each accounts as acc}
					<div class="bg-white p-5 rounded-[1.8rem] border border-gray-100 shadow-sm flex items-center justify-between group hover:shadow-lg transition-all">
						<div class="flex items-center gap-3">
							<div class="h-3 w-3 rounded-full {acc.color} shadow-lg shadow-current/20"></div>
							<p class="text-sm font-bold text-[#0a2e31]">{acc.name}</p>
						</div>
						<p class="text-sm font-black text-[#0a2e31] opacity-60 group-hover:opacity-100 transition-opacity">
							{formatCurrency(acc.balance)}
						</p>
					</div>
				{/each}
				
				<button class="w-full py-4 border-2 border-dashed border-gray-100 rounded-[1.8rem] text-gray-400 text-xs font-bold hover:bg-gray-50 hover:border-teal-100 hover:text-teal-600 transition-all">
					+ Tambah Akun Baru
				</button>
			</div>
		</div>
	</div>
</div>
