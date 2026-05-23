<script lang="ts">
	import { onMount } from 'svelte';
	import { fade, fly } from 'svelte/transition';
	import { apiFetch } from '$lib/api/client';
	import { budgetsRepo, categoriesRepo, type BudgetRow, type CategoryRow } from '$lib/db';

	type BudgetForm = {
		id: string;
		name: string;
		limit_idr: number | string;
		category_id: string;
		period_type: 'MONTHLY' | 'WEEKLY' | 'CUSTOM';
		start_date: string;
		end_date: string;
		alert_threshold: number | string;
		daily_limit_enabled: boolean;
	};

	let budgets = $state<BudgetRow[]>([]);
	let categories = $state<CategoryRow[]>([]);
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

	function barColor(b: BudgetRow) {
		if (b.is_over_budget) return 'bg-red-500';
		if (b.progress_percent > 75) return 'bg-yellow-400';
		return 'bg-[#217b84]';
	}

	function textColor(b: BudgetRow) {
		if (b.is_over_budget) return 'text-red-500';
		if (b.progress_percent > 75) return 'text-yellow-500';
		return 'text-[#217b84]';
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

	async function refreshFromServer() {
		try {
			const res = await apiFetch('/budgets');
			const result = await readApiResult(res);
			if (res.ok && result.success && Array.isArray(result.data)) {
				budgets = result.data as BudgetRow[];
				try {
					await budgetsRepo.clear();
					await budgetsRepo.putMany(result.data as BudgetRow[]);
				} catch {
					// IDB gagal — UI sudah ter-update, biarkan.
				}
			} else if (!res.ok) {
				errorMessage = result.error?.message || 'Gagal memuat anggaran.';
			}
		} catch {
			// Offline — tampilkan cache IDB.
		}
	}

	async function fetchCategories() {
		try {
			const cached = await categoriesRepo.getAll();
			if (cached.length > 0) {
				categories = cached.filter(
					(c) => c.category_type === 'EXPENSE' || c.category_type === 'BOTH'
				);
				return;
			}
			const res = await apiFetch('/categories');
			const result = await readApiResult(res);
			if (res.ok && result.success && Array.isArray(result.data)) {
				categories = (result.data as CategoryRow[]).filter(
					(c) => c.category_type === 'EXPENSE' || c.category_type === 'BOTH'
				);
			}
		} catch {
			// Ignore
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
				await refreshFromServer();
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
				await budgetsRepo.hardDelete(id);
				budgets = budgets.filter((b) => b.id !== id);
				await refreshFromServer();
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

	function openEditModal(b: BudgetRow) {
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

	function spentTodayPercent(b: BudgetRow): number {
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
		try {
			budgets = await budgetsRepo.getAll();
		} catch (err) {
			console.warn('Cache anggaran lokal belum siap, memuat dari server:', err);
		}
		loading = false;
		void fetchCategories();
		await refreshFromServer();
	});
</script>

<div class="animate-in fade-in space-y-8 pb-20 duration-500">
	<div class="flex items-start justify-between">
		<div class="space-y-1">
			<h1 class="text-3xl font-black text-[#0a2e31]">Anggaran</h1>
			<p class="text-sm font-medium text-gray-500">
				Tetapkan batas pengeluaran dan pantau realisasinya.
			</p>
		</div>
		<button
			onclick={openAddModal}
			class="rounded-xl bg-[#0a2e31] px-5 py-3 text-xs font-black tracking-widest text-white uppercase shadow-lg transition-all hover:bg-black active:scale-[0.98]"
		>
			+ Tambah Anggaran
		</button>
	</div>

	{#if errorMessage && !showModal}
		<div
			class="rounded-2xl border border-red-100 bg-red-50 px-4 py-3 text-sm font-bold text-red-600"
		>
			{errorMessage}
		</div>
	{/if}

	{#if loading}
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
			{#each [0, 1, 2] as i (i)}
				<div class="h-48 animate-pulse rounded-[2.5rem] bg-gray-100"></div>
			{/each}
		</div>
	{:else if budgets.length === 0}
		<!-- Empty State -->
		<div
			class="flex flex-col items-center justify-center gap-6 rounded-[2.5rem] border-2 border-dashed border-gray-100 bg-white py-20 text-center"
		>
			<div class="flex h-20 w-20 items-center justify-center rounded-3xl bg-teal-50 text-[#217b84]">
				<svg
					class="h-10 w-10"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="1.5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 11h6m-3-8a9 9 0 100 18 9 9 0 000-18z"
					/>
				</svg>
			</div>
			<div class="space-y-2">
				<h3 class="text-xl font-black text-[#0a2e31]">Belum Ada Anggaran</h3>
				<p class="max-w-xs text-sm font-medium text-gray-400">
					Buat anggaran pertama Anda untuk mulai memantau pengeluaran per kategori.
				</p>
			</div>
			<button
				onclick={openAddModal}
				class="rounded-xl bg-[#0a2e31] px-6 py-3 text-xs font-black tracking-widest text-white uppercase shadow-lg transition-all hover:bg-black"
			>
				Buat Anggaran Pertama
			</button>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
			{#each budgets as b (b.id)}
				<div
					class="group relative flex flex-col gap-5 rounded-[2.5rem] border border-gray-100 bg-white p-8 shadow-sm transition-all hover:shadow-lg"
				>
					<!-- Header -->
					<div class="flex items-start justify-between gap-3">
						<div class="min-w-0 flex-1">
							<h3 class="truncate text-base font-black text-[#0a2e31]">{b.name}</h3>
							<div class="mt-1 flex flex-wrap items-center gap-2">
								<span
									class="rounded-full bg-gray-100 px-2 py-0.5 text-[10px] font-bold tracking-wider text-gray-500 uppercase"
								>
									{periodLabel(b.period_type)}
								</span>
								{#if b.category_name}
									<span
										class="rounded-full bg-teal-50 px-2 py-0.5 text-[10px] font-bold tracking-wider text-[#217b84] uppercase"
									>
										{b.category_name}
									</span>
								{:else}
									<span
										class="rounded-full bg-gray-50 px-2 py-0.5 text-[10px] font-bold tracking-wider text-gray-400 uppercase"
									>
										Semua Pengeluaran
									</span>
								{/if}
							</div>
						</div>

						<!-- Actions (visible on hover) -->
						<div class="flex shrink-0 gap-1 opacity-0 transition-opacity group-hover:opacity-100">
							<button
								onclick={() => openEditModal(b)}
								aria-label="Edit anggaran {b.name}"
								class="rounded-xl p-2 text-gray-400 transition-colors hover:bg-gray-50 hover:text-[#0a2e31]"
							>
								<svg
									class="h-4 w-4"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="2"
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
								class="rounded-xl p-2 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-500"
							>
								<svg
									class="h-4 w-4"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="2"
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

					<!-- Progress -->
					<div class="space-y-2">
						<div class="flex items-center justify-between">
							<span class="text-xs font-bold text-gray-400">Terpakai</span>
							<span class="text-xs font-black {textColor(b)}">
								{Math.round(b.progress_percent)}%
							</span>
						</div>
						<div class="h-2.5 w-full overflow-hidden rounded-full bg-gray-100">
							<div
								class="h-full rounded-full transition-all duration-700 {barColor(b)}"
								style="width: {Math.min(100, b.progress_percent)}%"
							></div>
						</div>
						<div class="flex items-center justify-between text-[11px] font-bold">
							<span class="text-gray-500">{formatCurrency(b.spent_idr)}</span>
							<span class="text-gray-300">dari {formatCurrency(b.limit_idr)}</span>
						</div>
					</div>

					<!-- Over Budget Badge -->
					{#if b.is_over_budget}
						<div
							class="flex items-center gap-1.5 rounded-2xl border border-red-100 bg-red-50 px-3 py-2"
						>
							<svg
								class="h-3.5 w-3.5 shrink-0 text-red-500"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="2.5"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
								/>
							</svg>
							<span class="text-[10px] font-black tracking-wider text-red-600 uppercase"
								>Melebihi Anggaran</span
							>
						</div>
					{:else if b.progress_percent >= (b.alert_threshold ?? 80)}
						<div
							class="flex items-center gap-1.5 rounded-2xl border border-yellow-100 bg-yellow-50 px-3 py-2"
						>
							<svg
								class="h-3.5 w-3.5 shrink-0 text-yellow-500"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="2.5"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
								/>
							</svg>
							<span class="text-[10px] font-black tracking-wider text-yellow-600 uppercase"
								>Mendekati Batas</span
							>
						</div>
					{/if}

					<!-- Remaining -->
					{#if !b.is_over_budget}
						<p class="text-[11px] font-bold text-gray-400">
							Sisa: <span class="text-[#0a2e31]">{formatCurrency(b.remaining_idr)}</span>
						</p>
					{:else}
						<p class="text-[11px] font-bold text-red-400">
							Kelebihan: <span class="text-red-600">{formatCurrency(-b.remaining_idr)}</span>
						</p>
					{/if}

					<!-- Daily allowance section -->
					{#if b.daily_limit_enabled}
						<div class="mt-2 space-y-2 border-t border-gray-100 pt-3">
							<div class="flex items-center justify-between text-[11px]">
								<span class="font-bold text-gray-400">Jatah dasar</span>
								<span class="font-black text-gray-500"
									>{formatCurrency(b.daily_base_idr ?? 0)}/hari</span
								>
							</div>
							<div class="flex items-center justify-between text-[11px]">
								<span class="font-bold text-gray-400">Sisa kemarin</span>
								<span
									class="font-black {(b.carryover_idr ?? 0) >= 0
										? 'text-[#217b84]'
										: 'text-red-500'}"
								>
									{dailyCarryoverLabel(b.carryover_idr ?? 0)}
								</span>
							</div>
							<div class="flex items-center justify-between text-[11px]">
								<span class="font-black text-[#0a2e31]">Jatah hari ini</span>
								<span class="font-black text-[#0a2e31]"
									>{formatCurrency(b.daily_allowance_today_idr ?? 0)}</span
								>
							</div>
							<!-- Mini progress bar for today -->
							<div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-100">
								<div
									class="h-full rounded-full transition-all duration-500 {spentTodayPercent(b) >=
									100
										? 'bg-red-400'
										: spentTodayPercent(b) >= 75
											? 'bg-yellow-400'
											: 'bg-[#217b84]'}"
									style="width: {Math.min(100, spentTodayPercent(b))}%"
								></div>
							</div>
							<div class="flex items-center justify-between text-[11px]">
								<span class="font-bold text-gray-400">Terpakai hari ini</span>
								<span
									class="font-black {(b.daily_remaining_idr ?? 0) < 0
										? 'text-red-500'
										: 'text-gray-500'}">{formatCurrency(b.spent_today_idr ?? 0)}</span
								>
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
		class="fixed inset-0 z-50 flex items-center justify-center bg-[#0a2e31]/40 p-4 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
	>
		<div
			class="w-full max-w-sm overflow-hidden rounded-[2.5rem] bg-white shadow-2xl"
			in:fly={{ y: 20, duration: 400 }}
		>
			<div class="max-h-[90vh] space-y-6 overflow-y-auto p-8">
				<div class="flex items-center justify-between">
					<h2 class="text-xl font-black text-[#0a2e31]">
						{form.id ? 'Edit Anggaran' : 'Anggaran Baru'}
					</h2>
					<button
						aria-label="Tutup modal"
						onclick={() => (showModal = false)}
						class="text-gray-300 transition-colors hover:text-gray-500"
					>
						<svg
							class="h-6 w-6"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2.5"
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
							class="block px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
						>
							Nama Anggaran
						</label>
						<input
							id="bname"
							type="text"
							required
							bind:value={form.name}
							maxlength="100"
							placeholder="Misal: Makan & Minum"
							class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] transition-all outline-none focus:ring-2 focus:ring-teal-500"
						/>
					</div>

					<!-- Batas (IDR) -->
					<div class="space-y-1.5">
						<label
							for="blimit"
							class="block px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
						>
							Batas Pengeluaran (IDR)
						</label>
						<input
							id="blimit"
							type="number"
							required
							min="1"
							bind:value={form.limit_idr}
							placeholder="500000"
							class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] transition-all outline-none focus:ring-2 focus:ring-teal-500"
						/>
					</div>

					<!-- Kategori -->
					<div class="space-y-1.5">
						<label
							for="bcat"
							class="block px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
						>
							Kategori (Opsional)
						</label>
						<select
							id="bcat"
							bind:value={form.category_id}
							class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] transition-all outline-none focus:ring-2 focus:ring-teal-500"
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
							class="block px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
						>
							Periode
						</label>
						<select
							id="bperiod"
							bind:value={form.period_type}
							class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] transition-all outline-none focus:ring-2 focus:ring-teal-500"
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
								class="flex cursor-pointer items-center gap-3 rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 transition-all hover:bg-teal-50/40"
							>
								<input
									type="checkbox"
									bind:checked={form.daily_limit_enabled}
									class="h-4 w-4 rounded accent-[#217b84]"
								/>
								<div>
									<p class="text-sm font-black text-[#0a2e31]">Aktifkan Jatah Harian</p>
									<p class="text-[10px] font-bold text-gray-400">
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
									class="block px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
								>
									Mulai
								</label>
								<input
									id="bstart"
									type="date"
									bind:value={form.start_date}
									class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] transition-all outline-none focus:ring-2 focus:ring-teal-500"
								/>
							</div>
							<div class="space-y-1.5">
								<label
									for="bend"
									class="block px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
								>
									Selesai
								</label>
								<input
									id="bend"
									type="date"
									bind:value={form.end_date}
									class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] transition-all outline-none focus:ring-2 focus:ring-teal-500"
								/>
							</div>
						</div>
					{/if}

					<!-- Alert Threshold -->
					<div class="space-y-1.5">
						<label
							for="bthreshold"
							class="block px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
						>
							Peringatan saat mencapai (%)
						</label>
						<input
							id="bthreshold"
							type="number"
							min="0"
							max="100"
							bind:value={form.alert_threshold}
							class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] transition-all outline-none focus:ring-2 focus:ring-teal-500"
						/>
						<p class="px-1 text-[10px] font-bold text-gray-300">
							Tampilkan peringatan kuning saat pengeluaran mencapai {form.alert_threshold}% dari
							batas.
						</p>
					</div>

					{#if errorMessage}
						<div
							class="rounded-2xl border border-red-100 bg-red-50 px-4 py-3 text-xs font-bold text-red-600"
						>
							{errorMessage}
						</div>
					{/if}

					<div class="flex gap-3 pt-4">
						{#if form.id}
							<button
								type="button"
								onclick={() => handleDelete(form.id)}
								disabled={saving}
								class="rounded-xl border border-red-100 px-5 py-3 text-[10px] font-black tracking-widest text-red-500 uppercase transition-all hover:bg-red-50"
							>
								Hapus
							</button>
						{/if}
						<button
							type="submit"
							disabled={saving}
							class="flex-1 rounded-xl bg-[#0a2e31] py-3 text-[10px] font-black tracking-widest text-white uppercase shadow-lg transition-all hover:bg-black active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-60"
						>
							{saving ? 'Menyimpan...' : form.id ? 'Simpan Perubahan' : 'Buat Anggaran'}
						</button>
					</div>
				</form>
			</div>
		</div>
	</div>
{/if}
