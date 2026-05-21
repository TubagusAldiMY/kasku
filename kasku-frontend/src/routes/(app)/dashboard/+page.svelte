<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { resolve } from '$app/paths';
	import { SvelteDate } from 'svelte/reactivity';
	import {
		accountsRepo,
		transactionsRepo,
		categoriesRepo,
		type AccountRow,
		type TransactionRow,
		type CategoryRow
	} from '$lib/db';
	import { syncStatus, triggerManualSync } from '$lib/sync';

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

	const ACCOUNT_COLORS = [
		'bg-blue-600',
		'bg-orange-500',
		'bg-teal-500',
		'bg-purple-500',
		'bg-red-500'
	];

	let stats = $state({
		totalBalance: 0,
		monthlyIncome: 0,
		monthlyExpense: 0,
		savingsRate: 0
	});

	let recentTransactions = $state<RecentTransaction[]>([]);
	let accounts = $state<AccountSummary[]>([]);
	let loading = $state(true);
	let balanceHidden = $state(false);

	function toggleBalance() {
		balanceHidden = !balanceHidden;
		localStorage.setItem('kasku_balance_hidden', String(balanceHidden));
	}

	let weeklyData = $state([
		{ day: 'Sen', amount: 0 },
		{ day: 'Sel', amount: 0 },
		{ day: 'Rab', amount: 0 },
		{ day: 'Kam', amount: 0 },
		{ day: 'Jum', amount: 0 },
		{ day: 'Sab', amount: 0 },
		{ day: 'Min', amount: 0 }
	]);

	let maxAmount = $derived(Math.max(1, ...weeklyData.map((d) => d.amount)));

	function projectAccounts(rows: AccountRow[]): { summaries: AccountSummary[]; total: number } {
		let total = 0;
		const summaries = rows.map((a, i) => {
			total += a.balance;
			return {
				name: a.name,
				balance: a.balance,
				color: ACCOUNT_COLORS[i % ACCOUNT_COLORS.length]
			};
		});
		return { summaries, total };
	}

	function projectDashboard(
		accRows: AccountRow[],
		txRows: TransactionRow[],
		catRows: CategoryRow[]
	) {
		const categoryMap = new Map(catRows.map((c) => [c.id, c.name]));

		const { summaries, total } = projectAccounts(accRows);
		accounts = summaries;

		const sortedTx = [...txRows].sort((a, b) => (a.transaction_date < b.transaction_date ? 1 : -1));

		recentTransactions = sortedTx.slice(0, 5).map((t) => ({
			id: t.id,
			title: t.notes ?? t.transaction_type,
			category: categoryMap.get(t.category_id) ?? 'Umum',
			amount: t.transaction_type === 'INCOME' ? t.amount_idr : -t.amount_idr,
			date: new SvelteDate(t.transaction_date).toLocaleDateString('id-ID', {
				day: 'numeric',
				month: 'short'
			})
		}));

		const now = new SvelteDate();
		const currentMonth = now.getMonth();
		const currentYear = now.getFullYear();

		let mIncome = 0;
		let mExpense = 0;
		for (const t of txRows) {
			const d = new SvelteDate(t.transaction_date);
			if (d.getMonth() === currentMonth && d.getFullYear() === currentYear) {
				if (t.transaction_type === 'INCOME') mIncome += t.amount_idr;
				else if (t.transaction_type === 'EXPENSE') mExpense += t.amount_idr;
			}
		}

		const days = ['Min', 'Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab'];
		const last7Days = Array.from({ length: 7 }, (_unused, i) => {
			const d = new SvelteDate();
			d.setDate(d.getDate() - i);
			d.setHours(0, 0, 0, 0);
			return d;
		}).reverse();

		weeklyData = last7Days.map((date) => {
			const dayAmount = txRows
				.filter((t) => {
					const td = new SvelteDate(t.transaction_date);
					td.setHours(0, 0, 0, 0);
					return td.getTime() === date.getTime() && t.transaction_type === 'EXPENSE';
				})
				.reduce((sum, t) => sum + t.amount_idr, 0);
			return { day: days[date.getDay()], amount: dayAmount };
		});

		const savings = mIncome - mExpense;
		const sRate = mIncome > 0 && savings > 0 ? Math.round((savings / mIncome) * 100) : 0;

		stats = {
			totalBalance: total,
			monthlyIncome: mIncome,
			monthlyExpense: mExpense,
			savingsRate: sRate
		};
	}

	async function reloadFromLocal() {
		try {
			const [accRows, txRows, catRows] = await Promise.all([
				accountsRepo.getAll(),
				transactionsRepo.getAll(),
				categoriesRepo.getAll()
			]);
			projectDashboard(accRows, txRows, catRows);
		} catch (err) {
			console.error('Gagal memuat dashboard dari penyimpanan lokal:', err);
		}
	}

	async function hydrateCategoriesFromServer() {
		// Categories belum ikut sync engine — fetch on-demand & cache ke IDB.
		// TODO(sync): integrasikan categories ke SyncableResource saat siap.
		try {
			const res = await apiFetch('/categories');
			const result = await res.json();
			if (result.success && Array.isArray(result.data)) {
				await categoriesRepo.clear();
				await categoriesRepo.putMany(result.data as CategoryRow[]);
				await reloadFromLocal();
			}
		} catch {
			// Offline → cukup pakai cache IDB.
		}
	}

	$effect(() => {
		void syncStatus.dataVersion;
		void reloadFromLocal();
	});

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(val);
	}

	onMount(async () => {
		balanceHidden = localStorage.getItem('kasku_balance_hidden') === 'true';
		await reloadFromLocal();
		loading = false;
		void hydrateCategoriesFromServer();
		void triggerManualSync();
	});
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
					<div class="flex items-center gap-2">
						<span class="text-[11px] font-bold tracking-[0.2em] text-teal-400 uppercase"
							>Total Net Worth</span
						>
						<button
							onclick={toggleBalance}
							class="text-teal-400 opacity-60 transition-opacity hover:opacity-100"
							aria-label={balanceHidden ? 'Tampilkan saldo' : 'Sembunyikan saldo'}
						>
							{#if balanceHidden}
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
									/>
								</svg>
							{:else}
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
									/>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
									/>
								</svg>
							{/if}
						</button>
					</div>
					<div class="text-5xl font-black tracking-tighter">
						{loading ? '...' : balanceHidden ? 'Rp ••••••••' : formatCurrency(stats.totalBalance)}
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
							<p class="text-sm font-bold text-white"
								>{balanceHidden ? '••••••' : formatCurrency(stats.monthlyIncome)}</p
							>
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
							<p class="text-sm font-bold text-white"
								>{balanceHidden ? '••••••' : formatCurrency(stats.monthlyExpense)}</p
							>
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

		<div class="relative flex h-64 w-full items-stretch justify-between gap-4 px-2">
			{#each weeklyData as data (data.day)}
				<div class="group relative flex flex-1 flex-col items-center gap-2">
					<!-- Tooltip -->
					<div
						class="pointer-events-none absolute bottom-full mb-2 rounded-lg bg-[#0a2e31] px-3 py-1.5 text-[10px] font-bold whitespace-nowrap text-white opacity-0 shadow-xl transition-opacity group-hover:opacity-100"
					>
						{formatCurrency(data.amount)}
					</div>

					<!-- Bar area: fills remaining height, bar grows from bottom -->
					<div class="flex w-full flex-1 items-end justify-center">
						<div
							class="w-full max-w-[40px] overflow-hidden rounded-2xl bg-gray-50 transition-all duration-500 group-hover:shadow-lg"
							style="height: {Math.max(data.amount > 0 ? 3 : 0, (data.amount / maxAmount) * 100)}%"
						>
							<div
								class="h-full bg-gradient-to-t from-[#217b84] to-[#4fd1c5] opacity-80 transition-opacity group-hover:opacity-100"
							></div>
						</div>
					</div>

					<!-- Day Label -->
					<span
						class="shrink-0 text-[11px] font-black tracking-tighter text-gray-400 uppercase transition-colors group-hover:text-[#0a2e31]"
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
