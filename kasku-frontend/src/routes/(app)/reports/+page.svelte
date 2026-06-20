<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';

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

	const CAT_COLORS = [
		'#217b84',
		'#f59e0b',
		'#8b5cf6',
		'#ef4444',
		'#06b6d4',
		'#10b981',
		'#f97316',
		'#6366f1',
		'#ec4899',
		'#14b8a6'
	];

	let loading = $state(true);
	let errorMsg = $state<string | null>(null);
	let report = $state<ReportData | null>(null);
	let activePreset = $state<string>('6B');
	let csvExporting = $state(false);

	// ── SVG Bar Chart constants ─────────────────────────────────────────────────
	const SVG_W = 500,
		SVG_H = 220;
	const PAD_L = 54,
		PAD_R = 16,
		PAD_T = 16,
		PAD_B = 44;
	const CHART_W = SVG_W - PAD_L - PAD_R;
	const CHART_H = SVG_H - PAD_T - PAD_B;

	// ── SVG Donut Chart constants ───────────────────────────────────────────────
	const DON_CX = 100,
		DON_CY = 100,
		DON_R = 68,
		DON_SW = 34;
	const DON_CIRC = 2 * Math.PI * DON_R;

	// ── Derived bar chart data ──────────────────────────────────────────────────
	type BarSegment = { x: number; y: number; h: number; fill: string; w: number };
	type BarMonth = { label: string; bars: BarSegment[]; labelX: number };

	let barMonths = $derived.by((): BarMonth[] => {
		const trend = report?.monthly_trend;
		if (!trend?.length) return [];
		const maxVal = Math.max(...trend.map((p) => Math.max(p.income, p.expense)), 1);
		const n = trend.length;
		const slotW = CHART_W / n;
		const barW = Math.min(slotW * 0.36, 22);
		const gap = 2;

		return trend.map((p, i) => {
			const slotX = PAD_L + i * slotW;
			const center = slotX + slotW / 2;
			const incH = Math.max((p.income / maxVal) * CHART_H, p.income > 0 ? 2 : 0);
			const expH = Math.max((p.expense / maxVal) * CHART_H, p.expense > 0 ? 2 : 0);
			return {
				label: fmtMonth(p.month),
				labelX: center,
				bars: [
					{ x: center - barW - gap / 2, y: PAD_T + CHART_H - incH, h: incH, fill: '#217b84', w: barW },
					{ x: center + gap / 2, y: PAD_T + CHART_H - expH, h: expH, fill: '#ef4444', w: barW }
				]
			};
		});
	});

	let yGrid = $derived.by((): { y: number; label: string }[] => {
		const trend = report?.monthly_trend;
		if (!trend?.length) return [];
		const maxVal = Math.max(...trend.map((p) => Math.max(p.income, p.expense)), 1);
		return [0.25, 0.5, 0.75, 1].map((f) => ({
			y: PAD_T + CHART_H * (1 - f),
			label: fmtShort(Math.round(maxVal * f))
		}));
	});

	// ── Derived donut chart data ────────────────────────────────────────────────
	type DonutSeg = { dash: number; offset: number; color: string; name: string; pct: number };

	let donutSegs = $derived.by((): DonutSeg[] => {
		const cats = report?.category_breakdown;
		if (!cats?.length) return [];
		let accumulated = 0;
		return cats.slice(0, 8).map((cat, i) => {
			const dash = (cat.percentage / 100) * DON_CIRC;
			const seg: DonutSeg = {
				dash,
				offset: DON_CIRC - accumulated,
				color: CAT_COLORS[i % CAT_COLORS.length],
				name: cat.category_name,
				pct: cat.percentage
			};
			accumulated += dash;
			return seg;
		});
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
		const from = new Date();
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

<div class="animate-in fade-in space-y-8 pb-20 duration-700">
	<!-- Header ─────────────────────────────────────────────────────────────── -->
	<div class="flex flex-wrap items-center justify-between gap-4">
		<div>
			<h1 class="text-3xl font-black text-[#0a2e31]">Laporan Visual</h1>
			<p class="font-medium text-gray-500">Analisis keuangan dalam grafik interaktif.</p>
		</div>
		<div class="flex items-center gap-3">
			<div class="flex rounded-2xl border border-gray-100 bg-gray-50 p-1">
				{#each PRESETS as p (p.label)}
					<button
						onclick={() => setPreset(p)}
						class="rounded-xl px-4 py-2 text-xs font-bold transition-all {activePreset === p.label
							? 'bg-[#0a2e31] text-white shadow'
							: 'text-gray-400 hover:text-[#0a2e31]'}"
					>{p.label}</button>
				{/each}
			</div>
			<button
				onclick={exportCSV}
				disabled={csvExporting}
				class="flex items-center gap-2 rounded-2xl bg-[#217b84] px-4 py-2.5 text-xs font-bold text-white transition-all hover:bg-[#1a5f66] disabled:opacity-60"
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2.5"
						d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
					/>
				</svg>
				{csvExporting ? 'Mengunduh...' : 'Ekspor CSV'}
			</button>
		</div>
	</div>

	<!-- Loading skeleton ────────────────────────────────────────────────────── -->
	{#if loading}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
			{#each [1, 2, 3] as i (i)}
				<div class="h-28 animate-pulse rounded-3xl bg-gray-100"></div>
			{/each}
		</div>
		<div class="grid grid-cols-1 gap-6 lg:grid-cols-5">
			<div class="h-64 animate-pulse rounded-[2rem] bg-gray-100 lg:col-span-3"></div>
			<div class="h-64 animate-pulse rounded-[2rem] bg-gray-100 lg:col-span-2"></div>
		</div>

	<!-- Error ──────────────────────────────────────────────────────────────── -->
	{:else if errorMsg}
		<div class="rounded-3xl border border-red-100 bg-red-50 p-10 text-center">
			<p class="font-bold text-red-700">{errorMsg}</p>
			<button
				onclick={() => fetchReport(PRESETS.find((p) => p.label === activePreset)?.months ?? 6)}
				class="mt-4 rounded-2xl bg-red-100 px-6 py-2 text-sm font-bold text-red-700 hover:bg-red-200"
			>Coba Lagi</button>
		</div>

	<!-- Data ───────────────────────────────────────────────────────────────── -->
	{:else if report}
		<!-- Summary Cards -->
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
			<div class="rounded-3xl border border-teal-100 bg-teal-50 p-6">
				<p class="mb-1 text-[10px] font-bold tracking-widest text-teal-600 uppercase">Pemasukan</p>
				<p class="text-2xl font-black text-[#0a2e31]">{fmtIDR(report.total_income)}</p>
				<p class="mt-1 text-xs text-teal-500">{report.period_from} — {report.period_to}</p>
			</div>
			<div class="rounded-3xl border border-red-100 bg-red-50 p-6">
				<p class="mb-1 text-[10px] font-bold tracking-widest text-red-400 uppercase">Pengeluaran</p>
				<p class="text-2xl font-black text-[#0a2e31]">{fmtIDR(report.total_expense)}</p>
				<p class="mt-1 text-xs text-red-400">{report.period_from} — {report.period_to}</p>
			</div>
			<div
				class="rounded-3xl border p-6 {report.net_amount >= 0
					? 'border-gray-100 bg-white shadow-sm'
					: 'border-orange-100 bg-orange-50'}"
			>
				<p class="mb-1 text-[10px] font-bold tracking-widest text-gray-400 uppercase">Selisih Bersih</p>
				<p
					class="text-2xl font-black {report.net_amount >= 0 ? 'text-teal-700' : 'text-red-600'}"
				>
					{report.net_amount >= 0 ? '+' : '-'}{fmtIDR(report.net_amount)}
				</p>
				<p class="mt-1 text-xs {report.net_amount >= 0 ? 'text-teal-500' : 'text-orange-500'}">
					{report.net_amount >= 0 ? 'Surplus' : 'Defisit'} periode ini
				</p>
			</div>
		</div>

		<!-- Charts Row -->
		<div class="grid grid-cols-1 gap-6 lg:grid-cols-5">
			<!-- ── Bar Chart (monthly trend) ──────────────────────────────────── -->
			<div class="rounded-[2rem] border border-gray-100 bg-white p-6 shadow-sm lg:col-span-3">
				<div class="mb-5 flex flex-wrap items-center justify-between gap-3">
					<h3 class="font-black text-[#0a2e31]">Tren Bulanan</h3>
					<div class="flex items-center gap-5 text-xs font-bold text-gray-500">
						<span class="flex items-center gap-1.5">
							<span class="inline-block h-2.5 w-2.5 rounded-full bg-[#217b84]"></span>Pemasukan
						</span>
						<span class="flex items-center gap-1.5">
							<span class="inline-block h-2.5 w-2.5 rounded-full bg-red-400"></span>Pengeluaran
						</span>
					</div>
				</div>
				{#if !barMonths.length}
					<div class="flex h-48 items-center justify-center text-sm text-gray-400">
						Belum ada data transaksi untuk periode ini
					</div>
				{:else}
					<svg
						viewBox="0 0 {SVG_W} {SVG_H}"
						class="w-full"
						aria-label="Grafik tren bulanan pemasukan dan pengeluaran"
					>
						<!-- Y-axis grid lines & labels -->
						{#each yGrid as g (g.y)}
							<line
								x1={PAD_L}
								y1={g.y}
								x2={SVG_W - PAD_R}
								y2={g.y}
								stroke="#f3f4f6"
								stroke-width="1"
							/>
							<text
								x={PAD_L - 6}
								y={g.y + 4}
								text-anchor="end"
								font-size="9"
								fill="#9ca3af"
								font-family="sans-serif">{g.label}</text
							>
						{/each}
						<!-- Baseline -->
						<line
							x1={PAD_L}
							y1={PAD_T + CHART_H}
							x2={SVG_W - PAD_R}
							y2={PAD_T + CHART_H}
							stroke="#e5e7eb"
							stroke-width="1"
						/>
						<!-- Bars & labels -->
						{#each barMonths as bm, i (i)}
							{#each bm.bars as bar}
								{#if bar.h > 0}
									<rect
										x={bar.x}
										y={bar.y}
										width={bar.w}
										height={bar.h}
										rx="3"
										fill={bar.fill}
										opacity="0.88"
									/>
								{/if}
							{/each}
							<text
								x={bm.labelX}
								y={SVG_H - PAD_B + 16}
								text-anchor="middle"
								font-size="10"
								fill="#6b7280"
								font-weight="600"
								font-family="sans-serif">{bm.label}</text
							>
						{/each}
					</svg>
				{/if}
			</div>

			<!-- ── Donut Chart (category breakdown) ───────────────────────────── -->
			<div class="rounded-[2rem] border border-gray-100 bg-white p-6 shadow-sm lg:col-span-2">
				<h3 class="mb-5 font-black text-[#0a2e31]">Kategori Pengeluaran</h3>
				{#if !report.category_breakdown?.length}
					<div class="flex h-48 items-center justify-center text-sm text-gray-400">
						Belum ada pengeluaran pada periode ini
					</div>
				{:else}
					<div class="flex flex-col items-center gap-5">
						<svg
							viewBox="0 0 200 200"
							class="w-44"
							aria-label="Grafik donat kategori pengeluaran"
						>
							<!-- Donut segments — group rotated -90° so first segment starts at 12 o'clock -->
							<g transform="rotate(-90 {DON_CX} {DON_CY})">
								{#each donutSegs as seg, i (i)}
									<circle
										cx={DON_CX}
										cy={DON_CY}
										r={DON_R}
										fill="none"
										stroke={seg.color}
										stroke-width={DON_SW}
										stroke-dasharray="{seg.dash} {DON_CIRC - seg.dash}"
										stroke-dashoffset={seg.offset}
									/>
								{/each}
							</g>
							<!-- Center label -->
							<text
								x="100"
								y="95"
								text-anchor="middle"
								font-size="10"
								fill="#9ca3af"
								font-weight="600"
								font-family="sans-serif">Total</text
							>
							<text
								x="100"
								y="112"
								text-anchor="middle"
								font-size="13"
								fill="#0a2e31"
								font-weight="800"
								font-family="sans-serif">{fmtShort(report.total_expense)}</text
							>
						</svg>

						<!-- Category legend -->
						<div class="w-full space-y-2.5">
							{#each report.category_breakdown.slice(0, 7) as cat, i (cat.category_id)}
								<div class="flex items-center justify-between gap-2 text-xs">
									<div class="flex min-w-0 items-center gap-2">
										<span
											class="h-2.5 w-2.5 shrink-0 rounded-full"
											style="background:{CAT_COLORS[i % CAT_COLORS.length]}"
										></span>
										<span class="truncate font-medium text-gray-700">{cat.category_name}</span>
									</div>
									<div class="flex shrink-0 items-center gap-2">
										<span class="text-gray-400">{fmtShort(cat.total_amount)}</span>
										<span class="w-9 text-right font-bold text-gray-600"
											>{cat.percentage.toFixed(1)}%</span
										>
									</div>
								</div>
							{/each}
							{#if report.category_breakdown.length > 7}
								<p class="text-center text-[10px] text-gray-400">
									+{report.category_breakdown.length - 7} kategori lainnya
								</p>
							{/if}
						</div>
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>
