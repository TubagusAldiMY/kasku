<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { SvelteDate } from 'svelte/reactivity';

	type MonthlyPoint = { month: string; income: number; expense: number };
	type CategoryItem = {
		category_id: string;
		category_name: string;
		total_amount: number;
		count: number;
		percentage: number;
	};
	type ReportData = {
		period_from: string;
		period_to: string;
		total_income: number;
		total_expense: number;
		net_amount: number;
		category_breakdown: CategoryItem[];
		monthly_trend: MonthlyPoint[];
	};

	const PRESETS = [
		{ label: '1B', months: 1 },
		{ label: '3B', months: 3 },
		{ label: '6B', months: 6 },
		{ label: '12B', months: 12 }
	] as const;

	// Editorial category palette (mockup "Pengeluaran per kategori" bars).
	const CAT_COLORS = ['#2dd4bf', '#85aee7', '#e0b357', '#f4795b', '#8fd18a', '#b8a7e9', '#94a3b8'];

	let loading = $state(true);
	let errorMsg = $state<string | null>(null);
	let report = $state<ReportData | null>(null);
	let activePreset = $state<string>('6B');
	let csvExporting = $state(false);

	// ── SVG Bar Chart constants ─────────────────────────────────────────────────
	const SVG_W = 520,
		SVG_H = 240;
	const PAD_L = 8,
		PAD_R = 8,
		PAD_T = 20,
		PAD_B = 40;
	const CHART_W = SVG_W - PAD_L - PAD_R;
	const CHART_H = SVG_H - PAD_T - PAD_B;

	// ── Derived bar chart data ──────────────────────────────────────────────────
	type BarSegment = { x: number; y: number; h: number; fill: string; w: number };
	type BarMonth = { label: string; bars: BarSegment[]; labelX: number };

	let barMonths = $derived.by((): BarMonth[] => {
		const trend = report?.monthly_trend;
		if (!trend?.length) return [];
		const maxVal = Math.max(...trend.map((p) => Math.max(p.income, p.expense)), 1);
		const n = trend.length;
		const slotW = CHART_W / n;
		const barW = Math.min(slotW * 0.28, 22);
		const gap = 4;

		return trend.map((p, i) => {
			const slotX = PAD_L + i * slotW;
			const center = slotX + slotW / 2;
			const incH = Math.max((p.income / maxVal) * CHART_H, p.income > 0 ? 2 : 0);
			const expH = Math.max((p.expense / maxVal) * CHART_H, p.expense > 0 ? 2 : 0);
			return {
				label: fmtMonth(p.month),
				labelX: center,
				bars: [
					{
						x: center - barW - gap / 2,
						y: PAD_T + CHART_H - incH,
						h: incH,
						fill: '#2dd4bf',
						w: barW
					},
					{
						x: center + gap / 2,
						y: PAD_T + CHART_H - expH,
						h: expH,
						fill: 'rgba(244,121,91,0.75)',
						w: barW
					}
				]
			};
		});
	});

	let yGrid = $derived.by((): { y: number; label: string }[] => {
		const trend = report?.monthly_trend;
		if (!trend?.length) return [];
		const maxVal = Math.max(...trend.map((p) => Math.max(p.income, p.expense)), 1);
		return [0.5, 1].map((f) => ({
			y: PAD_T + CHART_H * (1 - f),
			label: fmtShort(Math.round(maxVal * f))
		}));
	});

	// ── Helpers ─────────────────────────────────────────────────────────────────
	function fmtMonth(m: string): string {
		const [year, month] = m.split('-');
		return new Date(+year, +month - 1, 1).toLocaleDateString('id-ID', { month: 'short' });
	}

	function fmtShort(v: number): string {
		if (v >= 1_000_000_000) return `${(v / 1_000_000_000).toFixed(1)}M`;
		if (v >= 1_000_000) return `${(v / 1_000_000).toFixed(1)}jt`;
		if (v >= 1_000) return `${(v / 1_000).toFixed(0)}rb`;
		return v.toLocaleString('id-ID');
	}

	function fmtIDR(v: number): string {
		return `Rp ${Math.abs(v).toLocaleString('id-ID')}`;
	}

	function getDateRange(months: number): { from: string; to: string } {
		const to = new Date();
		const from = new SvelteDate();
		from.setMonth(from.getMonth() - months + 1);
		from.setDate(1);
		return { from: from.toISOString().split('T')[0], to: to.toISOString().split('T')[0] };
	}

	// ── API calls ───────────────────────────────────────────────────────────────
	async function fetchReport(months: number) {
		loading = true;
		errorMsg = null;
		const { from, to } = getDateRange(months);
		try {
			const res = await apiFetch(`/transactions/reports?from=${from}&to=${to}&months=${months}`);
			const json = await res.json();
			if (!res.ok || !json.success) throw new Error(json.error?.message ?? 'Gagal memuat laporan');
			report = json.data as ReportData;
		} catch (e) {
			errorMsg = e instanceof Error ? e.message : 'Gagal memuat laporan';
		} finally {
			loading = false;
		}
	}

	function setPreset(p: (typeof PRESETS)[number]) {
		activePreset = p.label;
		fetchReport(p.months);
	}

	async function exportCSV() {
		csvExporting = true;
		try {
			const res = await apiFetch('/transactions/export');
			if (!res.ok) throw new Error('Gagal ekspor');
			const blob = await res.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `kasku_export_${new Date().toISOString().split('T')[0]}.csv`;
			document.body.appendChild(a);
			a.click();
			URL.revokeObjectURL(url);
			document.body.removeChild(a);
		} finally {
			csvExporting = false;
		}
	}

	onMount(() => fetchReport(6));
</script>

<div>
	<!-- Header ─────────────────────────────────────────────────────────────── -->
	<div class="flex flex-wrap items-end justify-between gap-4 border-b border-ink/10 pb-8">
		<div>
			<h1 class="font-serif text-4xl tracking-tight text-ink">Laporan</h1>
			<p class="mt-2.5 text-sm text-ink/60">Arus kas dan rincian kategori periode ini.</p>
		</div>
		<div class="flex items-center gap-2.5">
			<div class="flex gap-1 rounded-full border border-ink/15 p-1">
				{#each PRESETS as p (p.label)}
					<button
						onclick={() => setPreset(p)}
						class="rounded-full px-3.5 py-1.5 text-[12.5px] font-semibold transition-colors {activePreset ===
						p.label
							? 'bg-ink text-card'
							: 'text-ink/50 hover:text-ink'}">{p.label}</button
					>
				{/each}
			</div>
			<button
				onclick={exportCSV}
				disabled={csvExporting}
				class="flex items-center gap-2 rounded-full bg-teal px-4 py-2 text-[12.5px] font-semibold text-card transition-colors hover:bg-ink disabled:opacity-60"
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
					/>
				</svg>
				{csvExporting ? 'Mengunduh…' : 'Ekspor CSV'}
			</button>
		</div>
	</div>

	<!-- Loading skeleton ────────────────────────────────────────────────────── -->
	{#if loading}
		<div class="grid grid-cols-1 border-b border-ink/10 py-8 sm:grid-cols-3">
			{#each [1, 2, 3] as i (i)}
				<div class="px-7 first:pl-0">
					<div class="mb-2 h-3 w-20 animate-pulse rounded bg-ink/5"></div>
					<div class="h-8 w-32 animate-pulse rounded bg-ink/5"></div>
				</div>
			{/each}
		</div>
		<div class="grid grid-cols-1 gap-16 py-10 lg:grid-cols-2">
			<div class="h-56 animate-pulse rounded bg-ink/5"></div>
			<div class="h-56 animate-pulse rounded bg-ink/5"></div>
		</div>

		<!-- Error ──────────────────────────────────────────────────────────────── -->
	{:else if errorMsg}
		<div class="mt-8 border-y border-clay/20 bg-clay/5 py-16 text-center">
			<p class="text-clay">{errorMsg}</p>
			<button
				onclick={() => fetchReport(PRESETS.find((p) => p.label === activePreset)?.months ?? 6)}
				class="mt-4 rounded-full border border-clay/25 px-5 py-2 text-sm font-semibold text-clay transition-colors hover:bg-clay/10"
				>Coba lagi</button
			>
		</div>

		<!-- Data ───────────────────────────────────────────────────────────────── -->
	{:else if report}
		<!-- Tri-stat header -->
		<div class="grid grid-cols-1 border-b border-ink/10 py-9 sm:grid-cols-3">
			<div class="pr-7 pb-6 sm:pb-0">
				<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">Pemasukan</p>
				<p class="mt-2 font-serif text-4xl text-teal tabular-nums">{fmtIDR(report.total_income)}</p>
			</div>
			<div class="border-t border-ink/12 pt-6 pr-7 sm:border-t-0 sm:border-l sm:pt-0 sm:pl-7">
				<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">Pengeluaran</p>
				<p class="mt-2 font-serif text-4xl text-clay tabular-nums">
					{fmtIDR(report.total_expense)}
				</p>
			</div>
			<div class="border-t border-ink/12 pt-6 sm:border-t-0 sm:border-l sm:pt-0 sm:pl-7">
				<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
					Selisih (ditabung)
				</p>
				<p
					class="mt-2 font-serif text-4xl tabular-nums {report.net_amount >= 0
						? 'text-ink'
						: 'text-clay'}"
				>
					{report.net_amount >= 0 ? '+' : '−'}{fmtIDR(report.net_amount)}
				</p>
				<p class="mt-1.5 text-xs text-ink/45">
					{report.net_amount >= 0 ? 'Surplus' : 'Defisit'} · {report.period_from} — {report.period_to}
				</p>
			</div>
		</div>

		<!-- Charts Row -->
		<div class="grid grid-cols-1 gap-16 py-10 lg:grid-cols-2">
			<!-- ── Per-category horizontal bars ───────────────────────────────── -->
			<div>
				<p class="mb-5 text-[13px] font-semibold text-ink">Pengeluaran per kategori</p>
				{#if !report.category_breakdown?.length}
					<p class="py-16 text-sm text-ink/45">Belum ada pengeluaran pada periode ini.</p>
				{:else}
					{#each report.category_breakdown.slice(0, 8) as cat, i (cat.category_id)}
						<div class="mb-4 last:mb-0">
							<div class="mb-1.5 flex items-baseline justify-between text-[13px]">
								<span class="truncate pr-2 font-medium text-ink">{cat.category_name}</span>
								<span class="shrink-0 text-ink/60 tabular-nums">
									{fmtShort(cat.total_amount)} · {cat.percentage.toFixed(0)}%
								</span>
							</div>
							<div class="h-3.5 bg-ink/7">
								<div
									class="h-full"
									style="width: {Math.min(100, cat.percentage)}%; background: {CAT_COLORS[
										i % CAT_COLORS.length
									]}"
								></div>
							</div>
						</div>
					{/each}
					{#if report.category_breakdown.length > 8}
						<p class="mt-3 text-[11px] text-ink/45">
							+{report.category_breakdown.length - 8} kategori lainnya
						</p>
					{/if}
				{/if}
			</div>

			<!-- ── Bar Chart (monthly trend) ──────────────────────────────────── -->
			<div>
				<p class="mb-5 text-[13px] font-semibold text-ink">
					Arus kas {PRESETS.find((p) => p.label === activePreset)?.months ?? 6} bulan terakhir
				</p>
				{#if !barMonths.length}
					<p class="py-16 text-sm text-ink/45">Belum ada data transaksi untuk periode ini.</p>
				{:else}
					<svg
						viewBox="0 0 {SVG_W} {SVG_H}"
						class="block h-60 w-full"
						aria-label="Grafik arus kas bulanan pemasukan dan pengeluaran"
					>
						<!-- Y-axis grid lines -->
						{#each yGrid as g (g.y)}
							<line
								x1={PAD_L}
								y1={g.y}
								x2={SVG_W - PAD_R}
								y2={g.y}
								stroke="rgba(233,237,244,0.06)"
								stroke-width="1"
							/>
						{/each}
						<!-- Baseline -->
						<line
							x1={PAD_L}
							y1={PAD_T + CHART_H}
							x2={SVG_W - PAD_R}
							y2={PAD_T + CHART_H}
							stroke="rgba(233,237,244,0.15)"
							stroke-width="1"
						/>
						<!-- Bars & labels -->
						{#each barMonths as bm, i (i)}
							{#each bm.bars as bar, j (j)}
								{#if bar.h > 0}
									<rect x={bar.x} y={bar.y} width={bar.w} height={bar.h} fill={bar.fill} />
								{/if}
							{/each}
							<text
								x={bm.labelX}
								y={SVG_H - PAD_B + 20}
								text-anchor="middle"
								font-size="11"
								fill="rgba(233,237,244,0.45)"
								font-weight="500"
								font-family="sans-serif">{bm.label}</text
							>
						{/each}
					</svg>
					<div class="mt-4 flex gap-5 text-xs text-ink/60">
						<span class="flex items-center gap-1.5"
							><span class="inline-block h-2.5 w-2.5 bg-teal"></span>Pemasukan</span
						>
						<span class="flex items-center gap-1.5"
							><span class="inline-block h-2.5 w-2.5 bg-clay/75"></span>Pengeluaran</span
						>
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>
