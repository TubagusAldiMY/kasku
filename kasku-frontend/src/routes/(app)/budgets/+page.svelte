<script lang="ts">
	import { onMount } from 'svelte';
	import { fade, fly } from 'svelte/transition';
	import { apiFetch } from '$lib/api/client';

	type PeriodType = 'MONTHLY' | 'WEEKLY' | 'CUSTOM';

	// Shape rendered by this page — mirrors budgetResponse in the transaction-service handler.
	type Budget = {
		id: string;
		name: string;
		limit_idr: number;
		category_id?: string;
		category_name?: string;
		period_type: PeriodType;
		start_date?: string;
		end_date?: string | null;
		alert_threshold: number;
		spent_idr: number;
		remaining_idr: number;
		progress_percent: number;
		is_over_budget: boolean;
		updated_at: string;
		daily_limit_enabled: boolean;
		daily_base_idr?: number;
		carryover_idr?: number;
		daily_allowance_today_idr?: number;
		spent_today_idr?: number;
		daily_remaining_idr?: number;
	};

	type Category = {
		id: string;
		name: string;
		category_type: 'INCOME' | 'EXPENSE' | 'BOTH';
	};

	type BudgetForm = {
		id: string;
		name: string;
		limit_idr: number | string;
		category_id: string;
		period_type: PeriodType;
		start_date: string;
		end_date: string;
		alert_threshold: number | string;
		daily_limit_enabled: boolean;
	};

	let budgets = $state<Budget[]>([]);
	let categories = $state<Category[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let errorMessage = $state('');
	let showModal = $state(false);

	const defaultForm: BudgetForm = {
		id: '',
		name: '',
		limit_idr: '',
		category_id: '',
		period_type: 'MONTHLY',
		start_date: new Date().toISOString().substring(0, 10),
		end_date: '',
		alert_threshold: 80,
		daily_limit_enabled: false
	};

	let form = $state<BudgetForm>({ ...defaultForm });

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(val);
	}

	function periodLabel(p: string) {
		if (p === 'MONTHLY') return 'Bulanan';
		if (p === 'WEEKLY') return 'Mingguan';
		return 'Kustom';
	}

	function barColor(b: Budget) {
		if (b.is_over_budget) return 'bg-clay';
		if (b.progress_percent > 75) return 'bg-gold';
		return 'bg-teal';
	}

	function textColor(b: Budget) {
		if (b.is_over_budget) return 'text-clay';
		if (b.progress_percent > 75) return 'text-gold';
		return 'text-ink';
	}

	async function readApiResult(res: Response) {
		try {
			return await res.json();
		} catch {
			return {
				success: false,
				error: { message: `Response API tidak valid (HTTP ${res.status})` }
			};
		}
	}

	// Defensive: backend serializes snake_case, but tolerate PascalCase (Go struct) fallbacks.
	function pick(item: Record<string, unknown>, snake: string, pascal: string): unknown {
		return item[snake] ?? item[pascal];
	}
	function numOrUndef(v: unknown): number | undefined {
		return v == null ? undefined : Number(v);
	}

	function normalizeBudget(raw: Record<string, unknown>): Budget | null {
		const id = pick(raw, 'id', 'ID');
		const name = pick(raw, 'name', 'Name');
		const periodType = pick(raw, 'period_type', 'PeriodType');
		if (typeof id !== 'string' || typeof name !== 'string' || typeof periodType !== 'string') {
			return null;
		}
		const catId = pick(raw, 'category_id', 'CategoryID');
		const catName = pick(raw, 'category_name', 'CategoryName');
		const startDate = pick(raw, 'start_date', 'StartDate');
		const endDate = pick(raw, 'end_date', 'EndDate');
		return {
			id,
			name,
			limit_idr: Number(pick(raw, 'limit_idr', 'LimitIDR') ?? 0),
			category_id: typeof catId === 'string' && catId ? catId : undefined,
			category_name: typeof catName === 'string' ? catName : undefined,
			period_type: periodType as PeriodType,
			start_date: typeof startDate === 'string' ? startDate : undefined,
			end_date: typeof endDate === 'string' ? endDate : null,
			alert_threshold: Number(pick(raw, 'alert_threshold', 'AlertThreshold') ?? 80),
			spent_idr: Number(pick(raw, 'spent_idr', 'SpentIDR') ?? 0),
			remaining_idr: Number(pick(raw, 'remaining_idr', 'RemainingIDR') ?? 0),
			progress_percent: Number(pick(raw, 'progress_percent', 'ProgressPercent') ?? 0),
			is_over_budget: Boolean(pick(raw, 'is_over_budget', 'IsOverBudget')),
			updated_at: String(pick(raw, 'updated_at', 'UpdatedAt') ?? ''),
			daily_limit_enabled: Boolean(pick(raw, 'daily_limit_enabled', 'DailyLimitEnabled')),
			daily_base_idr: numOrUndef(pick(raw, 'daily_base_idr', 'DailyBaseIDR')),
			carryover_idr: numOrUndef(pick(raw, 'carryover_idr', 'CarryoverIDR')),
			daily_allowance_today_idr: numOrUndef(
				pick(raw, 'daily_allowance_today_idr', 'DailyAllowanceTodayIDR')
			),
			spent_today_idr: numOrUndef(pick(raw, 'spent_today_idr', 'SpentTodayIDR')),
			daily_remaining_idr: numOrUndef(pick(raw, 'daily_remaining_idr', 'DailyRemainingIDR'))
		};
	}

	function normalizeBudgets(data: unknown): Budget[] {
		if (!Array.isArray(data)) return [];
		return data
			.map((item) => normalizeBudget(item as Record<string, unknown>))
			.filter((b): b is Budget => b !== null);
	}

	async function fetchBudgets() {
		loading = true;
		errorMessage = '';
		try {
			const res = await apiFetch('/budgets');
			const result = await readApiResult(res);
			if (res.ok && result.success) {
				budgets = normalizeBudgets(result.data);
			} else {
				errorMessage = result.error?.message || 'Gagal memuat anggaran.';
			}
		} catch (err) {
			console.error(err);
			errorMessage = 'Gagal memuat anggaran. Periksa koneksi atau service backend.';
		} finally {
			loading = false;
		}
	}

	async function fetchCategories() {
		try {
			const res = await apiFetch('/categories');
			const result = await readApiResult(res);
			if (res.ok && result.success && Array.isArray(result.data)) {
				categories = (result.data as Record<string, unknown>[])
					.map((c) => ({
						id: String(c.id ?? c.ID ?? ''),
						name: String(c.name ?? c.Name ?? ''),
						category_type: (c.category_type ?? c.CategoryType) as Category['category_type']
					}))
					.filter((c) => c.id && (c.category_type === 'EXPENSE' || c.category_type === 'BOTH'));
			}
		} catch (err) {
			console.error(err);
		}
	}

	async function handleSave(e: SubmitEvent) {
		e.preventDefault();
		errorMessage = '';
		saving = true;
		try {
			const isEdit = !!form.id;
			const url = isEdit ? `/budgets/${form.id}` : '/budgets';
			const method = isEdit ? 'PUT' : 'POST';
			const body: Record<string, unknown> = {
				name: form.name.trim(),
				limit_idr: Number(form.limit_idr),
				period_type: form.period_type,
				alert_threshold: Number(form.alert_threshold),
				daily_limit_enabled: form.daily_limit_enabled
			};
			if (form.category_id) body.category_id = form.category_id;
			if (form.start_date) body.start_date = form.start_date;
			if (form.end_date) body.end_date = form.end_date;

			const res = await apiFetch(url, { method, body: JSON.stringify(body) });
			const result = await readApiResult(res);
			if (res.ok && result.success) {
				showModal = false;
				await fetchBudgets();
			} else {
				errorMessage = result.error?.message || 'Gagal menyimpan anggaran.';
			}
		} catch (err) {
			console.error(err);
			errorMessage = 'Gagal menyimpan anggaran. Periksa koneksi atau service backend.';
		} finally {
			saving = false;
		}
	}

	async function handleDelete(id: string) {
		if (!confirm('Hapus anggaran ini?')) return;
		saving = true;
		try {
			const res = await apiFetch(`/budgets/${id}`, { method: 'DELETE' });
			const result = await readApiResult(res);
			if (res.ok && result.success) {
				showModal = false;
				await fetchBudgets();
			} else {
				errorMessage = result.error?.message || 'Gagal menghapus anggaran.';
			}
		} catch (err) {
			console.error(err);
			errorMessage = 'Gagal menghapus anggaran. Periksa koneksi atau service backend.';
		} finally {
			saving = false;
		}
	}

	function openAddModal() {
		errorMessage = '';
		form = { ...defaultForm, start_date: new Date().toISOString().substring(0, 10) };
		showModal = true;
	}

	function openEditModal(b: Budget) {
		errorMessage = '';
		form = {
			id: b.id,
			name: b.name,
			limit_idr: b.limit_idr,
			category_id: b.category_id ?? '',
			period_type: b.period_type,
			start_date: b.start_date ?? new Date().toISOString().substring(0, 10),
			end_date: b.end_date ?? '',
			alert_threshold: b.alert_threshold,
			daily_limit_enabled: b.daily_limit_enabled ?? false
		};
		showModal = true;
	}

	function spentTodayPercent(b: Budget): number {
		const allowance = b.daily_allowance_today_idr ?? 0;
		if (allowance <= 0) return 100;
		return Math.round(((b.spent_today_idr ?? 0) / allowance) * 100);
	}

	function dailyCarryoverLabel(carryover: number): string {
		if (carryover > 0) return `+${formatCurrency(carryover)}`;
		if (carryover < 0) return formatCurrency(carryover);
		return formatCurrency(0);
	}

	onMount(async () => {
		void fetchCategories();
		await fetchBudgets();
	});

	// ── Editorial presentation helper (additive, no logic change) ──
	const totals = $derived({
		spent: budgets.reduce((s, b) => s + b.spent_idr, 0),
		limit: budgets.reduce((s, b) => s + b.limit_idr, 0)
	});
</script>

<div>
	<!-- ═══════════ Header ═══════════ -->
	<div class="flex flex-wrap items-end justify-between gap-4 border-b border-ink/10 pb-8">
		<div>
			<h1 class="font-serif text-4xl tracking-tight text-ink">Anggaran</h1>
			<p class="mt-2.5 text-sm text-ink/60">
				{#if !loading && budgets.length > 0}
					Terpakai <span class="font-semibold text-ink">{formatCurrency(totals.spent)}</span> dari
					<span class="font-semibold text-ink">{formatCurrency(totals.limit)}</span>.
				{:else}
					Tetapkan batas pengeluaran dan pantau realisasinya.
				{/if}
			</p>
		</div>
		<button
			onclick={openAddModal}
			class="rounded-full bg-teal px-5 py-2.5 text-[13px] font-semibold text-card transition-colors hover:bg-ink"
		>
			+ Buat anggaran
		</button>
	</div>

	{#if errorMessage && !showModal}
		<div class="mt-6 rounded-[10px] border border-clay/25 bg-clay/5 px-4 py-3 text-sm text-clay">
			{errorMessage}
		</div>
	{/if}

	{#if loading}
		<div class="pt-2">
			{#each [0, 1, 2] as i (i)}
				<div class="border-t border-ink/10 py-7">
					<div class="mb-3 h-6 w-40 animate-pulse rounded bg-ink/5"></div>
					<div class="h-1 animate-pulse rounded bg-ink/10"></div>
				</div>
			{/each}
		</div>
	{:else if budgets.length === 0}
		<!-- Empty State -->
		<div
			class="flex flex-col items-center justify-center gap-5 border-y border-dashed border-ink/15 py-24 text-center"
		>
			<div
				class="flex h-16 w-16 items-center justify-center rounded-full border border-ink/12 text-teal"
			>
				<svg
					class="h-8 w-8"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="1.4"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 11h6m-3-8a9 9 0 100 18 9 9 0 000-18z"
					/>
				</svg>
			</div>
			<div class="space-y-1.5">
				<h3 class="font-serif text-2xl text-ink">Belum ada anggaran</h3>
				<p class="max-w-xs text-sm text-ink/55">
					Buat anggaran pertama Anda untuk mulai memantau pengeluaran per kategori.
				</p>
			</div>
			<button
				onclick={openAddModal}
				class="rounded-full bg-teal px-5 py-2.5 text-[13px] font-semibold text-card transition-colors hover:bg-ink"
			>
				Buat anggaran pertama
			</button>
		</div>
	{:else}
		<div class="pt-2">
			{#each budgets as b (b.id)}
				{@const pct = Math.round(b.progress_percent)}
				<div class="group border-t border-ink/10 py-7 last:border-b">
					<!-- Row header: name + tag · percent -->
					<div class="mb-2.5 flex items-baseline justify-between gap-4">
						<div class="flex min-w-0 items-baseline gap-3">
							<span class="truncate font-serif text-[22px] text-ink">{b.name}</span>
							{#if b.category_name}
								<span
									class="shrink-0 rounded-full border border-gold/30 px-2 py-px text-[11px] text-gold"
								>
									{b.category_name}
								</span>
							{:else}
								<span
									class="shrink-0 rounded-full border border-ink/15 px-2 py-px text-[11px] text-ink/55"
								>
									Semua pengeluaran
								</span>
							{/if}
							<span
								class="hidden shrink-0 rounded-full border border-ink/15 px-2 py-px text-[11px] text-ink/45 sm:inline"
							>
								{periodLabel(b.period_type)}
							</span>
						</div>
						<div class="flex shrink-0 items-baseline gap-3">
							<span class="font-serif text-[22px] tabular-nums {textColor(b)}">{pct}%</span>
							<!-- Actions (visible on hover) -->
							<div class="flex gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
								<button
									onclick={() => openEditModal(b)}
									aria-label="Edit anggaran {b.name}"
									class="rounded-full p-1.5 text-ink/40 transition-colors hover:bg-ink/5 hover:text-ink"
								>
									<svg
										class="h-4 w-4"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="1.8"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
										/>
									</svg>
								</button>
								<button
									onclick={() => handleDelete(b.id)}
									aria-label="Hapus anggaran {b.name}"
									class="rounded-full p-1.5 text-ink/40 transition-colors hover:bg-clay/10 hover:text-clay"
								>
									<svg
										class="h-4 w-4"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="1.8"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
										/>
									</svg>
								</button>
							</div>
						</div>
					</div>

					<!-- Track -->
					<div class="mb-2.5 h-1 bg-ink/10">
						<div
							class="h-full transition-all duration-700 {barColor(b)}"
							style="width: {Math.min(100, b.progress_percent)}%"
						></div>
					</div>

					<!-- Caption line -->
					<div
						class="flex flex-wrap items-baseline justify-between gap-x-3 text-[12.5px] text-ink/55"
					>
						<span class="tabular-nums">{formatCurrency(b.spent_idr)} terpakai</span>
						<span class="tabular-nums">
							limit {formatCurrency(b.limit_idr)} ·
							{#if b.is_over_budget}
								<span class="font-semibold text-clay">lewat {formatCurrency(-b.remaining_idr)}</span
								>
							{:else}
								sisa {formatCurrency(b.remaining_idr)}
							{/if}
						</span>
					</div>

					<!-- Warn caption when approaching threshold (not yet over) -->
					{#if !b.is_over_budget && b.progress_percent >= (b.alert_threshold ?? 80)}
						<p class="mt-2 text-[11px] font-semibold tracking-[0.08em] text-gold uppercase">
							Mendekati batas
						</p>
					{/if}

					<!-- Daily allowance section -->
					{#if b.daily_limit_enabled}
						<div
							class="mt-4 grid gap-x-8 gap-y-2 border-t border-ink/8 pt-4 text-[12.5px] sm:grid-cols-2"
						>
							<div class="flex items-baseline justify-between">
								<span class="text-ink/50">Jatah dasar</span>
								<span class="text-ink/70 tabular-nums"
									>{formatCurrency(b.daily_base_idr ?? 0)}/hari</span
								>
							</div>
							<div class="flex items-baseline justify-between">
								<span class="text-ink/50">Sisa kemarin</span>
								<span
									class="font-semibold tabular-nums {(b.carryover_idr ?? 0) >= 0
										? 'text-teal'
										: 'text-clay'}"
								>
									{dailyCarryoverLabel(b.carryover_idr ?? 0)}
								</span>
							</div>
							<div class="flex items-baseline justify-between">
								<span class="font-medium text-ink">Jatah hari ini</span>
								<span class="font-semibold text-ink tabular-nums"
									>{formatCurrency(b.daily_allowance_today_idr ?? 0)}</span
								>
							</div>
							<div class="flex items-baseline justify-between">
								<span class="text-ink/50">Terpakai hari ini</span>
								<span
									class="font-semibold tabular-nums {(b.daily_remaining_idr ?? 0) < 0
										? 'text-clay'
										: 'text-ink/70'}"
								>
									{formatCurrency(b.spent_today_idr ?? 0)}
								</span>
							</div>
							<!-- Mini progress bar for today -->
							<div class="h-1 bg-ink/10 sm:col-span-2">
								<div
									class="h-full transition-all duration-500 {spentTodayPercent(b) >= 100
										? 'bg-clay'
										: spentTodayPercent(b) >= 75
											? 'bg-gold'
											: 'bg-teal'}"
									style="width: {Math.min(100, spentTodayPercent(b))}%"
								></div>
							</div>
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Modal Buat/Edit Anggaran -->
{#if showModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-4 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
	>
		<div
			class="w-full max-w-sm overflow-hidden rounded-2xl border border-ink/10 bg-card shadow-2xl"
			in:fly={{ y: 20, duration: 400 }}
		>
			<div class="max-h-[90vh] space-y-6 overflow-y-auto p-8">
				<div class="flex items-center justify-between">
					<h2 class="font-serif text-2xl text-ink">
						{form.id ? 'Edit anggaran' : 'Anggaran baru'}
					</h2>
					<button
						aria-label="Tutup modal"
						onclick={() => (showModal = false)}
						class="rounded-full p-1.5 text-ink/40 transition-colors hover:bg-ink/5 hover:text-ink"
					>
						<svg
							class="h-5 w-5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"
						>
							<path d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<form onsubmit={handleSave} class="space-y-5">
					<!-- Nama -->
					<div class="space-y-1.5">
						<label
							for="bname"
							class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
						>
							Nama anggaran
						</label>
						<input
							id="bname"
							type="text"
							required
							bind:value={form.name}
							maxlength="100"
							placeholder="Misal: Makan & Minum"
							class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none focus:border-teal"
						/>
					</div>

					<!-- Batas (IDR) -->
					<div class="space-y-1.5">
						<label
							for="blimit"
							class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
						>
							Batas pengeluaran (IDR)
						</label>
						<input
							id="blimit"
							type="number"
							required
							min="1"
							bind:value={form.limit_idr}
							placeholder="500000"
							class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none focus:border-teal"
						/>
					</div>

					<!-- Kategori -->
					<div class="space-y-1.5">
						<label
							for="bcat"
							class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
						>
							Kategori (opsional)
						</label>
						<select
							id="bcat"
							bind:value={form.category_id}
							class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none focus:border-teal"
						>
							<option value="">Semua Pengeluaran</option>
							{#each categories as cat (cat.id)}
								<option value={cat.id}>{cat.name}</option>
							{/each}
						</select>
					</div>

					<!-- Periode -->
					<div class="space-y-1.5">
						<label
							for="bperiod"
							class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
						>
							Periode
						</label>
						<select
							id="bperiod"
							bind:value={form.period_type}
							class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none focus:border-teal"
						>
							<option value="MONTHLY">Bulanan</option>
							<option value="WEEKLY">Mingguan</option>
							<option value="CUSTOM">Kustom (Manual)</option>
						</select>
					</div>

					<!-- Jatah Harian (hanya untuk MONTHLY dan WEEKLY) -->
					{#if form.period_type === 'MONTHLY' || form.period_type === 'WEEKLY'}
						<div class="space-y-1.5">
							<label
								class="flex cursor-pointer items-center gap-3 rounded-[10px] border border-ink/15 bg-field px-4 py-3 transition-colors hover:border-teal/40"
							>
								<input
									type="checkbox"
									bind:checked={form.daily_limit_enabled}
									class="h-4 w-4 rounded accent-teal"
								/>
								<div>
									<p class="text-sm font-semibold text-ink">Aktifkan jatah harian</p>
									<p class="text-[11px] text-ink/50">
										Anggaran dibagi rata per hari. Sisa/kelebihan hari ini terbawa ke esok.
									</p>
								</div>
							</label>
						</div>
					{/if}

					<!-- Tanggal Mulai (hanya untuk CUSTOM) -->
					{#if form.period_type === 'CUSTOM'}
						<div class="grid grid-cols-2 gap-4">
							<div class="space-y-1.5">
								<label
									for="bstart"
									class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>
									Mulai
								</label>
								<input
									id="bstart"
									type="date"
									bind:value={form.start_date}
									class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none focus:border-teal"
								/>
							</div>
							<div class="space-y-1.5">
								<label
									for="bend"
									class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>
									Selesai
								</label>
								<input
									id="bend"
									type="date"
									bind:value={form.end_date}
									class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none focus:border-teal"
								/>
							</div>
						</div>
					{/if}

					<!-- Alert Threshold -->
					<div class="space-y-1.5">
						<label
							for="bthreshold"
							class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
						>
							Peringatan saat mencapai (%)
						</label>
						<input
							id="bthreshold"
							type="number"
							min="0"
							max="100"
							bind:value={form.alert_threshold}
							class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none focus:border-teal"
						/>
						<p class="text-[11px] text-ink/45">
							Tampilkan peringatan saat pengeluaran mencapai {form.alert_threshold}% dari batas.
						</p>
					</div>

					{#if errorMessage}
						<div class="rounded-[10px] border border-clay/25 bg-clay/5 px-4 py-3 text-xs text-clay">
							{errorMessage}
						</div>
					{/if}

					<div class="flex gap-3 pt-4">
						{#if form.id}
							<button
								type="button"
								onclick={() => handleDelete(form.id)}
								disabled={saving}
								class="rounded-full border border-ink/15 px-5 py-2.5 text-[13px] font-semibold text-clay transition-colors hover:border-clay/30 hover:bg-clay/5 disabled:opacity-60"
							>
								Hapus
							</button>
						{/if}
						<button
							type="submit"
							disabled={saving}
							class="flex-1 rounded-full bg-teal py-2.5 text-[13px] font-semibold text-card transition-colors hover:bg-ink disabled:cursor-not-allowed disabled:opacity-60"
						>
							{saving ? 'Menyimpan…' : form.id ? 'Simpan perubahan' : 'Buat anggaran'}
						</button>
					</div>
				</form>
			</div>
		</div>
	</div>
{/if}
