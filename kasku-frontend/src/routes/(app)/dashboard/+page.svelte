<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { resolve } from '$app/paths';
	import { SvelteDate } from 'svelte/reactivity';

	// ── Local view models (online-only; no IndexedDB rows) ──
	type Account = { name: string; balance: number };
	type Transaction = {
		id: string;
		transaction_type: string;
		amount_idr: number;
		transaction_date: string;
		category_id: string;
		notes: string;
	};
	type Category = { id: string; name: string };
	type Budget = {
		id: string;
		name: string;
		progress_percent: number;
		is_over_budget: boolean;
	};

	// Server payloads vary between snake_case (json tags) and PascalCase
	// (entities without tags), so every field read is defensive.
	type Raw = Record<string, unknown>;
	type ApiResult = { success?: boolean; data?: unknown; error?: { message?: string } };

	function asArray(data: unknown): Raw[] {
		return Array.isArray(data) ? (data as Raw[]) : [];
	}
	function pick(o: Raw, ...keys: string[]): unknown {
		for (const k of keys) if (o[k] != null) return o[k];
		return undefined;
	}
	function num(v: unknown): number {
		return typeof v === 'number' ? v : 0;
	}
	function str(v: unknown): string {
		return typeof v === 'string' ? v : '';
	}

	async function readApiResult(res: Response): Promise<ApiResult> {
		try {
			return (await res.json()) as ApiResult;
		} catch {
			return {
				success: false,
				error: { message: `Response API tidak valid (HTTP ${res.status})` }
			};
		}
	}

	function normAccount(o: Raw): Account {
		return { name: str(pick(o, 'name', 'Name')), balance: num(pick(o, 'balance', 'Balance')) };
	}
	function normTransaction(o: Raw): Transaction {
		return {
			id: str(pick(o, 'id', 'ID')),
			transaction_type: str(pick(o, 'transaction_type', 'TransactionType')),
			amount_idr: num(pick(o, 'amount_idr', 'AmountIDR')),
			transaction_date: str(pick(o, 'transaction_date', 'TransactionDate')),
			category_id: str(pick(o, 'category_id', 'CategoryID')),
			notes: str(pick(o, 'notes', 'Notes'))
		};
	}
	function normCategory(o: Raw): Category {
		return { id: str(pick(o, 'id', 'ID')), name: str(pick(o, 'name', 'Name')) };
	}
	function normBudget(o: Raw): Budget {
		return {
			id: str(pick(o, 'id', 'ID')),
			name: str(pick(o, 'name', 'Name')),
			progress_percent: num(pick(o, 'progress_percent', 'ProgressPercent')),
			is_over_budget: Boolean(pick(o, 'is_over_budget', 'IsOverBudget'))
		};
	}

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
	let budgets = $state<Budget[]>([]);
	let loading = $state(true);
	let errorMessage = $state('');
	let balanceHidden = $state(false);

	let investmentCostBasis = $state(0);
	let debtSummary = $state({ totalReceivable: 0, totalPayable: 0, overdueCount: 0 });

	let trueNetWorth = $derived(stats.totalBalance + investmentCostBasis - debtSummary.totalPayable);

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

	const topBudgets = $derived(
		[...budgets].sort((a, b) => b.progress_percent - a.progress_percent).slice(0, 3)
	);

	// ── Editorial presentation helpers (additive, no logic change) ──
	const monthLabel = new SvelteDate().toLocaleDateString('id-ID', {
		month: 'long',
		year: 'numeric'
	});

	const monthlySavings = $derived(stats.monthlyIncome - stats.monthlyExpense);

	// Line chart in the mockup's 1128×180 viewBox, derived from weeklyData.
	const CHART_W = 1128;
	const CHART_H = 180;
	const chartPoints = $derived.by(() => {
		const max = Math.max(1, ...weeklyData.map((d) => d.amount));
		const n = weeklyData.length;
		const padX = 40;
		return weeklyData.map((d, i) => ({
			...d,
			x: n === 1 ? CHART_W / 2 : padX + (i / (n - 1)) * (CHART_W - padX * 2),
			y: CHART_H - 20 - (d.amount / max) * (CHART_H - 40)
		}));
	});
	const chartPath = $derived(
		chartPoints
			.map((p, i) => `${i === 0 ? 'M' : 'L'}${Math.round(p.x)} ${Math.round(p.y)}`)
			.join(' ')
	);
	const peakIndex = $derived.by(() => {
		let idx = 0;
		weeklyData.forEach((d, i) => {
			if (d.amount > weeklyData[idx].amount) idx = i;
		});
		return idx;
	});

	function fmtSigned(amount: number) {
		const sign = amount < 0 ? '−' : '+';
		return sign + new Intl.NumberFormat('id-ID').format(Math.abs(amount));
	}

	function projectAccounts(rows: Account[]): { summaries: AccountSummary[]; total: number } {
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

	function projectDashboard(accRows: Account[], txRows: Transaction[], catRows: Category[]) {
		const categoryMap = new Map(catRows.map((c) => [c.id, c.name]));

		const { total } = projectAccounts(accRows);

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

	// Core summary: accounts + transactions + categories. Failure here is surfaced.
	async function loadCore() {
		try {
			const [accRes, txRes, catRes] = await Promise.all([
				apiFetch('/accounts'),
				apiFetch('/transactions'),
				apiFetch('/categories')
			]);
			const [accJson, txJson, catJson] = await Promise.all([
				readApiResult(accRes),
				readApiResult(txRes),
				readApiResult(catRes)
			]);
			if (
				!accRes.ok ||
				!accJson.success ||
				!txRes.ok ||
				!txJson.success ||
				!catRes.ok ||
				!catJson.success
			) {
				errorMessage =
					accJson.error?.message ||
					txJson.error?.message ||
					catJson.error?.message ||
					'Gagal memuat ringkasan dashboard.';
				return;
			}
			projectDashboard(
				asArray(accJson.data).map(normAccount),
				asArray(txJson.data).map(normTransaction),
				asArray(catJson.data).map(normCategory)
			);
		} catch (err) {
			console.error(err);
			errorMessage = 'Gagal memuat dashboard. Periksa koneksi atau service backend.';
		}
	}

	async function loadInvestmentValue() {
		try {
			const res = await apiFetch('/investments');
			const json = await readApiResult(res);
			if (!res.ok || !json.success) return;
			// Cost basis = quantity × avg_buy_price (online source of truth).
			investmentCostBasis = asArray(json.data).reduce(
				(sum, r) =>
					sum + num(pick(r, 'quantity', 'Quantity')) * num(pick(r, 'avg_buy_price', 'AvgBuyPrice')),
				0
			);
		} catch {
			// Tidak fatal — tampilkan 0
		}
	}

	async function loadDebtSummary() {
		try {
			const res = await apiFetch('/debts');
			const json = await res.json();
			if (!res.ok || !json.success) return;
			const now = new Date();
			// Debt entity tidak pakai json tags — field PascalCase
			const all = json.data as Array<{
				Direction: string;
				Status: string;
				RemainingAmount: number;
				DueDate?: string;
			}>;
			const active = all.filter((d) => d.Status === 'ACTIVE');
			debtSummary = {
				totalReceivable: active
					.filter((d) => d.Direction === 'RECEIVABLE')
					.reduce((s, d) => s + d.RemainingAmount, 0),
				totalPayable: active
					.filter((d) => d.Direction === 'PAYABLE')
					.reduce((s, d) => s + d.RemainingAmount, 0),
				overdueCount: active.filter((d) => d.DueDate && new Date(d.DueDate) < now).length
			};
		} catch {
			// Offline — tidak tampil
		}
	}

	async function loadBudgets() {
		try {
			const res = await apiFetch('/budgets');
			const json = await readApiResult(res);
			if (res.ok && json.success) {
				budgets = asArray(json.data).map(normBudget);
			}
		} catch {
			// Non-fatal — seksi anggaran cukup kosong.
		}
	}

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(val);
	}

	onMount(async () => {
		balanceHidden = localStorage.getItem('kasku_balance_hidden') === 'true';
		await Promise.all([loadCore(), loadBudgets(), loadInvestmentValue(), loadDebtSummary()]);
		loading = false;
	});

	const statCells = $derived([
		{ label: 'Pemasukan', value: stats.monthlyIncome, cls: 'text-ink' },
		{ label: 'Pengeluaran', value: stats.monthlyExpense, cls: 'text-ink' },
		{ label: 'Investasi', value: investmentCostBasis, cls: 'text-ink' },
		{ label: 'Hutang', value: debtSummary.totalPayable, cls: 'text-clay' }
	]);
</script>

<div class="pb-4">
	{#if errorMessage}
		<div class="mb-6 rounded-[10px] border border-clay/25 bg-clay/5 px-4 py-3 text-sm text-clay">
			{errorMessage}
		</div>
	{/if}

	<!-- ═══════════ Hero: net worth + breakdown ═══════════ -->
	<section
		class="grid gap-10 border-b border-ink/10 pb-10 lg:grid-cols-[1.4fr_1fr] lg:items-end lg:gap-16"
	>
		<div>
			<div class="mb-2.5 flex items-center gap-2">
				<p class="text-[12px] font-semibold tracking-[0.14em] text-ink/45 uppercase">
					Kekayaan bersih · {monthLabel}
				</p>
				<button
					onclick={toggleBalance}
					class="text-ink/35 transition-colors hover:text-ink"
					aria-label={balanceHidden ? 'Tampilkan saldo' : 'Sembunyikan saldo'}
				>
					{#if balanceHidden}
						<svg
							class="h-4 w-4"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.8"
							><path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.542 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
							/></svg
						>
					{:else}
						<svg
							class="h-4 w-4"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.8"
							><path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
							/><path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
							/></svg
						>
					{/if}
				</button>
			</div>
			<p
				class="font-serif text-5xl leading-none tracking-tight text-ink tabular-nums sm:text-6xl lg:text-7xl"
			>
				{loading ? '…' : balanceHidden ? 'Rp ••••••••' : formatCurrency(trueNetWorth)}
			</p>
			<p class="mt-4 max-w-md text-sm leading-relaxed text-ink/60">
				{#if loading}
					Memuat ringkasan…
				{:else if stats.monthlyIncome === 0 && stats.monthlyExpense === 0}
					Belum ada aktivitas keuangan bulan ini. Ketuk “+ Catat” untuk memulai.
				{:else if monthlySavings > 0}
					Menyisihkan <span class="font-semibold text-teal">{fmtSigned(monthlySavings)}</span> bulan
					ini — {stats.savingsRate}% dari pemasukan. Pertahankan.
				{:else if monthlySavings < 0}
					Pengeluaran melebihi pemasukan bulan ini. Saatnya sedikit menahan laju.
				{:else}
					Pemasukan dan pengeluaran seimbang bulan ini.
				{/if}
			</p>
		</div>

		<div class="grid grid-cols-2 border-t border-ink/12 pt-6 lg:border-t-0 lg:border-l lg:pt-0">
			{#each statCells as s, i (s.label)}
				<div
					class="lg:pl-7 {i < 2 ? 'pr-4 pb-5' : 'border-t border-ink/12 pt-5 pr-4'} {i % 2 === 1
						? 'border-l border-ink/12 pl-4 lg:border-l-0 lg:pl-7'
						: ''}"
				>
					<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">{s.label}</p>
					<p class="mt-1.5 font-serif text-2xl {s.cls} tabular-nums">
						{loading || balanceHidden ? 'Rp •••' : formatCurrency(s.value)}
					</p>
				</div>
			{/each}
		</div>
	</section>

	<!-- ═══════════ Weekly spending line chart ═══════════ -->
	<section class="border-b border-ink/10 py-10">
		<div class="mb-4 flex items-baseline justify-between">
			<p class="text-[13px] font-semibold text-ink">Pengeluaran 7 hari terakhir</p>
			<p class="text-xs text-ink/50">
				Puncak {weeklyData[peakIndex].day} · {formatCurrency(weeklyData[peakIndex].amount)}
			</p>
		</div>
		<svg
			viewBox="0 0 {CHART_W} {CHART_H}"
			class="block h-[180px] w-full"
			preserveAspectRatio="none"
		>
			<line x1="0" y1="60" x2={CHART_W} y2="60" stroke="rgba(233,237,244,0.07)" stroke-width="1" />
			<line x1="0" y1="110" x2={CHART_W} y2="110" stroke="rgba(233,237,244,0.07)" stroke-width="1" />
			<line x1="0" y1="160" x2={CHART_W} y2="160" stroke="rgba(233,237,244,0.12)" stroke-width="1" />
			<path
				d={chartPath}
				fill="none"
				stroke="var(--color-teal)"
				stroke-width="2.5"
				stroke-linejoin="round"
				stroke-linecap="round"
				vector-effect="non-scaling-stroke"
			/>
			{#each chartPoints as p, i (p.day + i)}
				{#if i === peakIndex && p.amount > 0}
					<circle cx={p.x} cy={p.y} r="5" fill="var(--color-teal)" />
				{:else}
					<circle cx={p.x} cy={p.y} r="4" fill="#f7f6f2" stroke="var(--color-teal)" stroke-width="2.5" />
				{/if}
			{/each}
		</svg>
		<div class="flex justify-between px-4 pt-2 text-[11px] font-medium text-ink/45">
			{#each weeklyData as d, i (d.day + i)}
				<span>{d.day}</span>
			{/each}
		</div>
	</section>

	<!-- ═══════════ Activity + budget/debt ═══════════ -->
	<section class="grid gap-10 pt-10 lg:grid-cols-[1.4fr_1fr] lg:gap-16">
		<!-- Recent activity -->
		<div>
			<div class="mb-1 flex items-baseline justify-between">
				<p class="text-[13px] font-semibold text-ink">Aktivitas terakhir</p>
				<a
					href={resolve('/transactions')}
					class="text-xs font-semibold text-teal transition-colors hover:text-ink"
				>
					Lihat semua →
				</a>
			</div>

			{#if loading}
				<div class="space-y-3 pt-4">
					{#each [0, 1, 2, 3] as i (i)}
						<div class="h-6 animate-pulse rounded bg-ink/5"></div>
					{/each}
				</div>
			{:else if recentTransactions.length === 0}
				<p class="pt-6 text-sm text-ink/45">Belum ada transaksi. Ketuk “+ Catat” untuk memulai.</p>
			{:else}
				{#each recentTransactions as tx (tx.id)}
					<div
						class="flex items-baseline justify-between gap-4 border-b border-ink/8 py-4 last:border-0"
					>
						<div class="flex min-w-0 items-baseline gap-3.5">
							<span class="w-11 shrink-0 text-xs text-ink/40">{tx.date}</span>
							<span class="truncate text-sm font-medium text-ink">{tx.title}</span>
							<span
								class="hidden shrink-0 rounded-full border border-ink/15 px-2 py-px text-[11px] text-ink/55 sm:inline"
							>
								{tx.category}
							</span>
						</div>
						<span
							class="shrink-0 text-sm tabular-nums {tx.amount > 0
								? 'font-semibold text-teal'
								: 'text-ink'}"
						>
							{fmtSigned(tx.amount)}
						</span>
					</div>
				{/each}
			{/if}
		</div>

		<!-- Budget + debt -->
		<div>
			<p class="mb-4 text-[13px] font-semibold text-ink">Anggaran bulan ini</p>

			{#if !loading && topBudgets.length > 0}
				{#each topBudgets as b (b.id)}
					{@const pct = Math.round(b.progress_percent)}
					<div class="mb-5">
						<div class="mb-2 flex items-baseline justify-between text-[13px]">
							<span class="font-medium text-ink">{b.name}</span>
							<span class="font-semibold {b.is_over_budget ? 'text-clay' : 'text-ink/60'}"
								>{pct}%</span
							>
						</div>
						<div class="h-[3px] bg-ink/10">
							<div
								class="h-full {b.is_over_budget
									? 'bg-clay'
									: b.progress_percent > 75
										? 'bg-gold'
										: 'bg-ink'}"
								style="width: {Math.min(100, b.progress_percent)}%"
							></div>
						</div>
					</div>
				{/each}
			{:else if !loading}
				<p class="text-sm text-ink/45">Belum ada anggaran bulan ini.</p>
			{/if}

			<div class="mt-6 flex justify-between border-t border-ink/10 pt-4 text-[13px]">
				<span class="text-ink/55">Piutang aktif</span>
				<span class="font-semibold text-ink tabular-nums">
					{balanceHidden ? 'Rp •••' : formatCurrency(debtSummary.totalReceivable)}
				</span>
			</div>
			<div class="flex justify-between pt-2.5 text-[13px]">
				<span class="text-ink/55">Hutang aktif</span>
				<span class="font-semibold text-clay tabular-nums">
					{balanceHidden ? 'Rp •••' : formatCurrency(debtSummary.totalPayable)}
				</span>
			</div>

			{#if !loading && debtSummary.overdueCount > 0}
				<a
					href={resolve('/debts')}
					class="mt-4 flex items-center gap-2.5 rounded-xl border border-clay/25 bg-clay/5 px-3.5 py-2.5 text-[13px] text-clay transition-colors hover:bg-clay/10"
				>
					<svg
						class="h-4 w-4 shrink-0"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
						/></svg
					>
					<span class="flex-1">{debtSummary.overdueCount} catatan telah jatuh tempo</span>
					<span>→</span>
				</a>
			{/if}
		</div>
	</section>
</div>
