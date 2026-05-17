<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { resolve } from '$app/paths';
	import { SvelteDate } from 'svelte/reactivity';

	type RecentTransaction = {
		id: string;
		title: string;
		category: string;
		amount: number;
		date: string;
	};
	type AccountSummary = {
		name: string;
		balance: number;
		color: string;
	};
	type BackendCategory = { id: string; name: string };
	type BackendAccount = { id: string; name: string; balance: number };
	type BackendTransaction = {
		id: string;
		notes?: string;
		transaction_type: 'INCOME' | 'EXPENSE';
		category_id: string;
		amount_idr: number;
		transaction_date: string;
	};

	let stats = $state({
		totalBalance: 0,
		monthlyIncome: 0,
		monthlyExpense: 0,
		savingsRate: 0
	});

	let recentTransactions = $state<RecentTransaction[]>([]);
	let accounts = $state<AccountSummary[]>([]);
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
					{
						id: '1',
						title: 'Kopi Kenangan',
						category: 'Makanan',
						amount: -25000,
						date: 'Hari ini'
					},
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
			// Fetch accounts, transactions and categories
			const [accRes, txRes, catRes] = await Promise.all([
				apiFetch('/accounts'),
				apiFetch('/transactions'),
				apiFetch('/categories')
			]);

			const accData = await accRes.json();
			const txData = await txRes.json();
			const catData = await catRes.json();

			const categoryMap: Record<string, string> = {};
			if (catData.success && catData.data) {
				catData.data.forEach((c: BackendCategory) => {
					categoryMap[c.id] = c.name;
				});
			}

			let totalBal = 0;
			if (accData.success && accData.data) {
				const colors = [
					'bg-blue-600',
					'bg-orange-500',
					'bg-teal-500',
					'bg-purple-500',
					'bg-red-500'
				];
				accounts = accData.data.map((a: BackendAccount, i: number) => {
					totalBal += a.balance;
					return {
						name: a.name,
						balance: a.balance,
						color: colors[i % colors.length]
					};
				});
			}

			let mIncome = 0;
			let mExpense = 0;

			if (txData.success && txData.data) {
				const now = new SvelteDate();
				const currentMonth = now.getMonth();
				const currentYear = now.getFullYear();

				recentTransactions = txData.data.slice(0, 5).map((t: BackendTransaction) => ({
					id: t.id,
					title: t.notes || t.transaction_type,
					category: categoryMap[t.category_id] || 'Umum',
					amount: t.transaction_type === 'INCOME' ? t.amount_idr : -t.amount_idr,
					date: new SvelteDate(t.transaction_date).toLocaleDateString('id-ID', {
						day: 'numeric',
						month: 'short'
					})
				}));

				txData.data.forEach((t: BackendTransaction) => {
					const txDate = new SvelteDate(t.transaction_date);
					if (txDate.getMonth() === currentMonth && txDate.getFullYear() === currentYear) {
						if (t.transaction_type === 'INCOME') {
							mIncome += t.amount_idr;
						} else if (t.transaction_type === 'EXPENSE') {
							mExpense += t.amount_idr;
						}
					}
				});

				const days = ['Min', 'Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab'];
				const last7Days = Array.from({ length: 7 }, (_unused, i) => {
					const d = new SvelteDate();
					d.setDate(d.getDate() - i);
					d.setHours(0, 0, 0, 0);
					return d;
				}).reverse();

				weeklyData = last7Days.map((date) => {
					const dayAmount = txData.data
						.filter((t: BackendTransaction) => {
							const txDate = new SvelteDate(t.transaction_date);
							txDate.setHours(0, 0, 0, 0);
							return txDate.getTime() === date.getTime() && t.transaction_type === 'EXPENSE';
						})
						.reduce((sum: number, t: BackendTransaction) => sum + t.amount_idr, 0);

					return {
						day: days[date.getDay()],
						amount: dayAmount
					};
				});
			}

			let sRate = 0;
			if (mIncome > 0) {
				const savings = mIncome - mExpense;
				sRate = savings > 0 ? Math.round((savings / mIncome) * 100) : 0;
			}

			stats = {
				totalBalance: totalBal,
				monthlyIncome: mIncome,
				monthlyExpense: mExpense,
				savingsRate: sRate
			};
		} catch (e) {
			console.error(e);
		} finally {
			loading = false;
		}
	}

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(val);
	}

	onMount(loadDashboardData);
</script>

<div class="animate-in fade-in space-y-10 duration-700">
	<!-- Top Section: Welcome & Quick Stats -->
	<div class="flex flex-col justify-between gap-6 md:flex-row md:items-end">
		<div class="space-y-1">
			<h1 class="text-3xl font-black text-[#0a2e31]">Halo, {auth.user?.username || 'Juragan'}!</h1>
			<p class="font-medium text-gray-500">Berikut adalah ringkasan finansial Anda bulan ini.</p>
		</div>
		<div class="flex gap-3">
			<a
				href={resolve('/transactions')}
				class="rounded-xl border border-gray-200 bg-white px-5 py-2.5 text-sm font-bold text-[#0a2e31] shadow-sm transition-all hover:bg-gray-50"
				>Riwayat</a
			>
			<a
				href={resolve('/reports')}
				class="rounded-xl bg-[#217b84] px-5 py-2.5 text-sm font-bold text-white shadow-lg shadow-teal-900/10 transition-all hover:bg-[#1a5f66]"
				>Laporan</a
			>
		</div>
	</div>

	<!-- Hero Cards -->
	<div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
		<!-- Main Balance Card -->
		<div
			class="relative overflow-hidden rounded-[2.5rem] bg-[#0a2e31] p-10 text-white shadow-2xl lg:col-span-2"
		>
			<div class="absolute top-0 right-0 p-8 opacity-10">
				<svg class="h-32 w-32" fill="currentColor" viewBox="0 0 24 24"
					><path
						d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1.75 14.83h-3.5v-1.45c-1.43-.31-2.47-1.12-2.57-2.43h1.49c.09.68.74 1.19 1.48 1.19.8 0 1.42-.42 1.42-1.16 0-.61-.31-1.01-1.63-1.35-1.57-.41-2.61-1-2.61-2.53 0-1.28.97-2.18 2.42-2.49V7.12h3.5v1.44c1.23.27 2.1 1.01 2.24 2.15h-1.47c-.12-.66-.66-1.07-1.32-1.07-.73 0-1.25.4-1.25 1.05 0 .61.46.91 1.76 1.3 1.54.45 2.49 1.07 2.49 2.56 0 1.25-.9 2.18-2.47 2.52v1.41z"
					/></svg
				>
			</div>

			<div class="relative z-10 space-y-6">
				<div class="space-y-1">
					<span class="text-[11px] font-bold tracking-[0.2em] text-teal-400 uppercase"
						>Total Net Worth</span
					>
					<div class="text-5xl font-black tracking-tighter">
						{loading ? '...' : formatCurrency(stats.totalBalance)}
					</div>
				</div>

				<div class="flex items-center gap-6 pt-4">
					<div class="flex items-center gap-2">
						<div
							class="flex h-10 w-10 items-center justify-center rounded-2xl bg-white/10 text-green-400"
						>
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"
								><path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="3"
									d="M5 10l7-7m0 0l7 7m-7-7v18"
								/></svg
							>
						</div>
						<div>
							<p class="text-[10px] font-bold tracking-wider text-white/40 uppercase">Income</p>
							<p class="text-sm font-bold text-white">{formatCurrency(stats.monthlyIncome)}</p>
						</div>
					</div>
					<div class="flex items-center gap-2">
						<div
							class="flex h-10 w-10 items-center justify-center rounded-2xl bg-white/10 text-red-400"
						>
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"
								><path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="3"
									d="M19 14l-7 7m0 0l-7-7m7 7V3"
								/></svg
							>
						</div>
						<div>
							<p class="text-[10px] font-bold tracking-wider text-white/40 uppercase">Expense</p>
							<p class="text-sm font-bold text-white">{formatCurrency(stats.monthlyExpense)}</p>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Savings Card -->
		<div
			class="flex flex-col justify-between rounded-[2.5rem] border border-gray-100 bg-white p-10 shadow-sm"
		>
			<div class="space-y-4">
				<span class="text-[11px] font-bold tracking-[0.2em] text-gray-400 uppercase"
					>Savings Rate</span
				>
				<div class="relative mx-auto h-32 w-32">
					<!-- Simple Circle Progress -->
					<svg class="h-full w-full -rotate-90 transform">
						<circle
							cx="64"
							cy="64"
							r="58"
							stroke="currentColor"
							stroke-width="12"
							fill="transparent"
							class="text-gray-50"
						/>
						<circle
							cx="64"
							cy="64"
							r="58"
							stroke="currentColor"
							stroke-width="12"
							fill="transparent"
							stroke-dasharray="364.4"
							stroke-dashoffset={364.4 - (364.4 * stats.savingsRate) / 100}
							class="text-[#217b84] transition-all duration-1000"
						/>
					</svg>
					<div class="absolute inset-0 flex items-center justify-center">
						<span class="text-3xl font-black text-[#0a2e31]">{stats.savingsRate}%</span>
					</div>
				</div>
			</div>
			<p class="text-center text-xs leading-relaxed font-medium text-gray-400">
				Anda menyisihkan <span class="font-bold text-[#217b84]">{stats.savingsRate}%</span> pendapatan
				bulan ini. Pertahankan!
			</p>
		</div>
	</div>

	<!-- Analytics Section: Expense Chart -->
	<div class="space-y-8 rounded-[2.5rem] border border-gray-100 bg-white p-10 shadow-sm">
		<div class="flex items-center justify-between">
			<div class="space-y-1">
				<h3 class="text-xl font-bold text-[#0a2e31]">Pengeluaran 7 Hari Terakhir</h3>
				<p class="text-xs font-bold tracking-wider text-gray-400 uppercase">Statistik Mingguan</p>
			</div>
			<div class="text-right">
				<p class="text-sm font-bold text-red-500">Puncak: {formatCurrency(maxAmount)}</p>
			</div>
		</div>

		<div class="relative flex h-64 w-full items-end justify-between gap-4 px-2">
			{#each weeklyData as data (data.day)}
				<div class="group flex flex-1 flex-col items-center gap-4">
					<!-- Tooltip -->
					<div
						class="pointer-events-none absolute bottom-full mb-2 rounded-lg bg-[#0a2e31] px-3 py-1.5 text-[10px] font-bold whitespace-nowrap text-white opacity-0 shadow-xl transition-opacity group-hover:opacity-100"
					>
						{formatCurrency(data.amount)}
					</div>

					<!-- Bar -->
					<div
						class="relative w-full max-w-[40px] overflow-hidden rounded-2xl bg-gray-50 transition-all duration-500 group-hover:shadow-lg"
						style="height: {(data.amount / maxAmount) * 100}%"
					>
						<div
							class="absolute inset-0 bg-gradient-to-t from-[#217b84] to-[#4fd1c5] opacity-80 transition-opacity group-hover:opacity-100"
						></div>
					</div>

					<!-- Day Label -->
					<span
						class="text-[11px] font-black tracking-tighter text-gray-400 uppercase transition-colors group-hover:text-[#0a2e31]"
						>{data.day}</span
					>
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
	<div class="grid grid-cols-1 gap-8 lg:grid-cols-3">
		<!-- Recent Transactions -->
		<div class="space-y-6 lg:col-span-2">
			<div class="flex items-center justify-between px-2">
				<h3 class="text-xl font-bold text-[#0a2e31]">Aktivitas Terakhir</h3>
				<a href={resolve('/transactions')} class="text-xs font-bold text-[#217b84] hover:underline"
					>Lihat Semua</a
				>
			</div>

			<div class="overflow-hidden rounded-[2.5rem] border border-gray-100 bg-white shadow-sm">
				{#if loading}
					<div class="space-y-4 p-8">
						{#each [0, 1, 2] as i (i)}
							<div class="h-14 animate-pulse rounded-2xl bg-gray-50"></div>
						{/each}
					</div>
				{:else}
					<div class="divide-y divide-gray-50">
						{#each recentTransactions as tx (tx.id)}
							<div class="flex items-center justify-between p-6 transition-colors hover:bg-gray-50">
								<div class="flex items-center gap-4">
									<div
										class="flex h-12 w-12 items-center justify-center rounded-2xl bg-gray-50 text-xs font-bold text-[#0a2e31] uppercase"
									>
										{tx.category.substring(0, 2)}
									</div>
									<div>
										<p class="font-bold text-[#0a2e31]">{tx.title}</p>
										<p class="text-[10px] font-bold tracking-wider text-gray-400 uppercase">
											{tx.category} • {tx.date}
										</p>
									</div>
								</div>
								<p
									class="font-black tracking-tight {tx.amount > 0
										? 'text-green-600'
										: 'text-[#0a2e31]'}"
								>
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
				<a href={resolve('/accounts')} class="text-xs font-bold text-[#217b84] hover:underline"
					>Kelola</a
				>
			</div>

			<div class="space-y-4">
				{#each accounts as acc (acc.name)}
					<div
						class="group flex items-center justify-between rounded-[1.8rem] border border-gray-100 bg-white p-5 shadow-sm transition-all hover:shadow-lg"
					>
						<div class="flex items-center gap-3">
							<div class="h-3 w-3 rounded-full {acc.color} shadow-lg shadow-current/20"></div>
							<p class="text-sm font-bold text-[#0a2e31]">{acc.name}</p>
						</div>
						<p
							class="text-sm font-black text-[#0a2e31] opacity-60 transition-opacity group-hover:opacity-100"
						>
							{formatCurrency(acc.balance)}
						</p>
					</div>
				{/each}

				<a
					href={resolve('/accounts')}
					class="block w-full rounded-[1.8rem] border-2 border-dashed border-gray-100 py-4 text-center text-xs font-bold text-gray-400 transition-all hover:border-teal-100 hover:bg-gray-50 hover:text-teal-600"
				>
					+ Tambah Akun Baru
				</a>
			</div>
		</div>
	</div>
</div>
