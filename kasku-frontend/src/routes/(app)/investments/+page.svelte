<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { fade, fly } from 'svelte/transition';
	import { investmentsRepo, type InvestmentRow } from '$lib/db';
	import {
		enqueueCreate,
		enqueueUpdate,
		enqueueDelete,
		syncStatus,
		triggerManualSync
	} from '$lib/sync';

	type AssetType = InvestmentRow['asset_type'];

	type HistoryEntry = {
		id: string;
		transaction_type: 'BUY' | 'SELL';
		quantity_change: number;
		price_per_unit: number;
		transaction_date: string;
		notes?: string;
	};

	let assets = $state<InvestmentRow[]>([]);
	let loading = $state(true);
	let showAssetModal = $state(false);
	let showHistoryModal = $state(false);
	let selectedAsset = $state<InvestmentRow | null>(null);
	let history = $state<HistoryEntry[]>([]);
	let historyLoading = $state(false);

	// Price runtime (tidak di-persist — out of sync engine scope).
	let priceLoading = $state<Record<string, boolean>>({});
	let priceCache = $state<Record<string, number>>({});

	async function handleFetchPrice(asset: InvestmentRow) {
		if (!asset.symbol) return;
		priceLoading[asset.id] = true;
		try {
			// TODO(sync): /prices/:symbol real-time fetch — tetap online-only, bukan SyncableResource.
			const res = await apiFetch(`/prices/${asset.symbol}`);
			const result = await res.json();
			if (result.success && result.data) {
				priceCache[asset.id] = Number(result.data.price_idr);
			}
		} catch (err) {
			console.error('Gagal mengambil harga:', err);
		} finally {
			priceLoading[asset.id] = false;
		}
	}

	type AssetFormState = {
		id: string;
		name: string;
		asset_type: AssetType;
		symbol: string;
		units: number;
		avg_buy_price_idr: number;
	};

	let assetForm = $state<AssetFormState>({
		id: '',
		name: '',
		asset_type: 'STOCK',
		symbol: '',
		units: 0,
		avg_buy_price_idr: 0
	});

	let historyForm = $state({
		transaction_type: 'BUY' as 'BUY' | 'SELL',
		quantity_change: 0,
		price_per_unit: 0,
		transaction_date: new Date().toISOString().split('T')[0],
		notes: ''
	});

	const assetTypes: { value: AssetType; label: string }[] = [
		{ value: 'STOCK', label: 'Saham' },
		{ value: 'CRYPTO', label: 'Kripto' },
		{ value: 'MUTUAL_FUND', label: 'Reksa Dana' },
		{ value: 'GOLD', label: 'Emas' }
	];

	async function reloadFromLocal() {
		try {
			assets = await investmentsRepo.getAll();
		} catch (err) {
			console.error('Gagal membaca instrumen dari penyimpanan lokal:', err);
		}
	}

	$effect(() => {
		void syncStatus.dataVersion;
		void reloadFromLocal();
	});

	async function fetchHistory(assetId: string) {
		historyLoading = true;
		try {
			// TODO(sync): unit_history sub-resource belum ikut sync engine — online-only.
			const res = await apiFetch(`/investments/${assetId}/history`);
			const result = await res.json();
			if (result.success) history = (result.data ?? []) as HistoryEntry[];
		} catch (err) {
			console.error('Gagal mengambil riwayat instrumen:', err);
		} finally {
			historyLoading = false;
		}
	}

	async function handleSaveAsset(e: SubmitEvent) {
		e.preventDefault();
		try {
			if (assetForm.id) {
				await enqueueUpdate<InvestmentRow>('investments', assetForm.id, {
					name: assetForm.name,
					asset_type: assetForm.asset_type,
					symbol: assetForm.symbol || undefined
				});
			} else {
				await enqueueCreate<InvestmentRow>('investments', {
					name: assetForm.name,
					asset_type: assetForm.asset_type,
					symbol: assetForm.symbol || undefined,
					units: assetForm.units,
					avg_buy_price_idr: assetForm.avg_buy_price_idr
				});
			}
			showAssetModal = false;
		} catch (err) {
			console.error('Gagal menyimpan instrumen:', err);
		}
	}

	async function handleDeleteAsset(id: string) {
		if (!confirm('Hapus instrumen ini? Riwayat transaksi juga akan terhapus.')) return;
		try {
			await enqueueDelete('investments', id);
			showAssetModal = false;
		} catch (err) {
			console.error('Gagal menghapus instrumen:', err);
		}
	}

	async function handleRecordHistory(e: SubmitEvent) {
		e.preventDefault();
		if (!selectedAsset) return;
		try {
			// TODO(sync): record history endpoint online-only; sync engine akan refresh investment row via pull.
			const res = await apiFetch(`/investments/${selectedAsset.id}/history`, {
				method: 'POST',
				body: JSON.stringify(historyForm)
			});
			const result = await res.json();
			if (result.success) {
				await fetchHistory(selectedAsset.id);
				// Trigger sync supaya server-side units change ke-pull ke IDB lokal.
				void triggerManualSync();
			}
		} catch (err) {
			console.error('Gagal mencatat transaksi instrumen:', err);
		}
	}

	function openAddModal() {
		assetForm = {
			id: '',
			name: '',
			asset_type: 'STOCK',
			symbol: '',
			units: 0,
			avg_buy_price_idr: 0
		};
		showAssetModal = true;
	}

	function openEditModal(asset: InvestmentRow) {
		assetForm = {
			id: asset.id,
			name: asset.name,
			asset_type: asset.asset_type,
			symbol: asset.symbol ?? '',
			units: asset.units,
			avg_buy_price_idr: asset.avg_buy_price_idr
		};
		showAssetModal = true;
	}

	function openHistoryModal(asset: InvestmentRow) {
		selectedAsset = asset;
		history = [];
		void fetchHistory(asset.id);
		showHistoryModal = true;
	}

	onMount(async () => {
		await reloadFromLocal();
		loading = false;
		void triggerManualSync();
	});

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(val);
	}

	function formatNumber(val: number) {
		return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 8 }).format(val);
	}

	const totalValue = $derived(assets.reduce((acc, a) => acc + a.units * a.avg_buy_price_idr, 0));
</script>

<div class="animate-in fade-in space-y-8 pb-20 duration-500">
	<!-- Header & Summary -->
	<div class="flex flex-col justify-between gap-6 md:flex-row md:items-end">
		<div class="space-y-1">
			<h1 class="text-3xl font-black text-[#0a2e31]">Portofolio Investasi</h1>
			<p class="text-sm font-medium text-gray-500">
				Kelola instrumen aset dan pantau pertumbuhan unit Anda.
			</p>
		</div>

		<div
			class="flex flex-col items-end rounded-[2rem] bg-[#0a2e31] px-8 py-5 text-white shadow-xl shadow-teal-950/20"
		>
			<span class="mb-1 text-[10px] font-black tracking-[0.2em] text-teal-400/80 uppercase"
				>Total Estimasi Aset</span
			>
			<span class="text-2xl font-black tabular-nums">{formatCurrency(totalValue)}</span>
		</div>
	</div>

	<!-- Action Bar -->
	<div class="flex justify-end">
		<button
			onclick={openAddModal}
			class="inline-flex items-center gap-2 rounded-2xl bg-[#217b84] px-6 py-3.5 text-xs font-black tracking-widest text-white uppercase shadow-lg transition-all hover:bg-[#1a5f66] active:scale-95"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3"
				><path d="M12 4v16m8-8H4" /></svg
			>
			Tambah Instrumen
		</button>
	</div>

	<!-- Assets List -->
	<div class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
		{#if loading}
			{#each [0, 1, 2] as i (i)}
				<div class="h-48 animate-pulse rounded-[2.5rem] border border-gray-100 bg-white"></div>
			{/each}
		{:else if assets.length === 0}
			<div
				class="col-span-full space-y-4 rounded-[2.5rem] border border-dashed border-gray-200 bg-white py-20 text-center"
			>
				<div
					class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-gray-50 text-gray-300"
				>
					<svg class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6"
						/></svg
					>
				</div>
				<div class="space-y-1">
					<p class="font-bold text-[#0a2e31]">Belum ada instrumen investasi</p>
					<p class="text-xs text-gray-400">Klik tombol di atas untuk mulai mencatat aset Anda.</p>
				</div>
			</div>
		{:else}
			{#each assets as asset (asset.id)}
				<div
					class="group relative overflow-hidden rounded-[2.5rem] border border-gray-100 bg-white p-8 shadow-sm transition-all hover:-translate-y-1 hover:shadow-xl"
				>
					<div
						class="absolute -top-4 -right-4 h-24 w-24 rounded-full bg-teal-50/50 transition-transform duration-700 group-hover:scale-125"
					></div>

					<div class="relative z-10 flex h-full flex-col">
						<div class="mb-6 flex items-start justify-between">
							<div>
								<span
									class="mb-2 inline-block rounded-full bg-teal-50 px-3 py-1 text-[10px] font-black tracking-widest text-teal-600 uppercase"
								>
									{asset.asset_type}
								</span>
								<h3 class="text-xl font-black text-[#0a2e31]">{asset.name}</h3>
								<p class="text-xs font-bold text-gray-400">{asset.symbol ?? '—'}</p>
							</div>
							<button
								aria-label="Edit {asset.name}"
								onclick={() => openEditModal(asset)}
								class="p-2 text-gray-300 transition-colors hover:text-[#0a2e31]"
							>
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"
									><path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
									/></svg
								>
							</button>
						</div>

						<div class="mt-auto space-y-4">
							<div class="flex items-end justify-between">
								<div class="space-y-0.5">
									<p class="text-[10px] font-bold tracking-widest text-gray-400 uppercase">
										Kepemilikan
									</p>
									<p class="text-lg font-black text-[#0a2e31]">
										{formatNumber(asset.units)}
										<span class="ml-1 text-xs font-bold text-gray-400">Unit</span>
									</p>
								</div>
								<div class="space-y-0.5 text-right">
									<p class="text-[10px] font-bold tracking-widest text-gray-400 uppercase">
										Avg Price
									</p>
									<p class="text-sm font-bold text-[#0a2e31]">
										{formatCurrency(asset.avg_buy_price_idr)}
									</p>
								</div>
							</div>

							{#if priceCache[asset.id] !== undefined}
								<div
									class="flex items-center justify-between rounded-xl border border-teal-100/50 bg-teal-50/50 p-3"
								>
									<span class="text-[10px] font-black tracking-widest text-teal-700 uppercase">
										Harga Terkini
									</span>
									<span class="text-sm font-black text-teal-800"
										>{formatCurrency(priceCache[asset.id])}</span
									>
								</div>
							{/if}

							<div class="flex gap-2">
								<button
									onclick={() => handleFetchPrice(asset)}
									disabled={priceLoading[asset.id] || !asset.symbol}
									class="rounded-xl bg-gray-50 p-3 text-[#0a2e31] transition-all hover:bg-gray-100 disabled:opacity-50"
									title="Perbarui Harga"
									aria-label="Perbarui harga {asset.name}"
								>
									<svg
										class="h-4 w-4 {priceLoading[asset.id] ? 'animate-spin' : ''}"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="3"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
										/>
									</svg>
								</button>
								<button
									onclick={() => openHistoryModal(asset)}
									class="flex-1 rounded-xl bg-gray-50 py-3 text-[11px] font-black tracking-[0.1em] text-[#0a2e31] uppercase transition-all hover:bg-teal-50 hover:text-teal-700"
								>
									Lihat Riwayat
								</button>
							</div>
						</div>
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>

<!-- Modal Asset (Add/Edit) -->
{#if showAssetModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-[#0a2e31]/40 p-4 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
	>
		<div
			class="w-full max-w-lg overflow-hidden rounded-[2.5rem] bg-white shadow-2xl"
			in:fly={{ y: 20, duration: 400 }}
		>
			<div class="space-y-8 p-10">
				<div class="flex items-center justify-between">
					<h2 class="text-2xl font-black text-[#0a2e31]">
						{assetForm.id ? 'Edit Instrumen' : 'Tambah Instrumen'}
					</h2>
					<button
						aria-label="Tutup modal"
						onclick={() => (showAssetModal = false)}
						class="text-gray-300 transition-colors hover:text-gray-500"
					>
						<svg
							class="h-6 w-6"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2.5"><path d="M6 18L18 6M6 6l12 12" /></svg
						>
					</button>
				</div>

				<form onsubmit={handleSaveAsset} class="space-y-6">
					<div class="grid grid-cols-1 gap-5 md:grid-cols-2">
						<div class="space-y-2 md:col-span-2">
							<label
								for="name"
								class="block px-1 text-[11px] font-black tracking-widest text-gray-400 uppercase"
								>Nama Aset</label
							>
							<input
								id="name"
								type="text"
								required
								bind:value={assetForm.name}
								placeholder="Contoh: Saham BBCA"
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-bold text-[#0a2e31] transition-all outline-none focus:ring-4 focus:ring-teal-50"
							/>
						</div>

						<div class="space-y-2">
							<label
								for="type"
								class="block px-1 text-[11px] font-black tracking-widest text-gray-400 uppercase"
								>Tipe Aset</label
							>
							<select
								id="type"
								bind:value={assetForm.asset_type}
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-bold text-[#0a2e31] transition-all outline-none focus:ring-4 focus:ring-teal-50"
							>
								{#each assetTypes as type (type.value)}
									<option value={type.value}>{type.label}</option>
								{/each}
							</select>
						</div>

						<div class="space-y-2">
							<label
								for="symbol"
								class="block px-1 text-[11px] font-black tracking-widest text-gray-400 uppercase"
								>Ticker / Simbol</label
							>
							<input
								id="symbol"
								type="text"
								bind:value={assetForm.symbol}
								placeholder="Contoh: BBCA.JK"
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-bold text-[#0a2e31] transition-all outline-none focus:ring-4 focus:ring-teal-50"
							/>
						</div>

						{#if !assetForm.id}
							<div class="space-y-2">
								<label
									for="qty"
									class="block px-1 text-[11px] font-black tracking-widest text-gray-400 uppercase"
									>Kuantitas Awal</label
								>
								<input
									id="qty"
									type="number"
									step="any"
									bind:value={assetForm.units}
									class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-bold text-[#0a2e31] transition-all outline-none focus:ring-4 focus:ring-teal-50"
								/>
							</div>

							<div class="space-y-2">
								<label
									for="price"
									class="block px-1 text-[11px] font-black tracking-widest text-gray-400 uppercase"
									>Harga Beli Rata-Rata (IDR)</label
								>
								<input
									id="price"
									type="number"
									step="any"
									bind:value={assetForm.avg_buy_price_idr}
									class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-bold text-[#0a2e31] transition-all outline-none focus:ring-4 focus:ring-teal-50"
								/>
							</div>
						{/if}
					</div>

					<div class="flex gap-4 pt-4">
						{#if assetForm.id}
							<button
								type="button"
								onclick={() => handleDeleteAsset(assetForm.id)}
								class="rounded-2xl border-2 border-red-50 px-6 py-4 text-xs font-black tracking-widest text-red-500 uppercase transition-all hover:bg-red-50"
							>
								Hapus
							</button>
						{/if}
						<button
							type="submit"
							class="flex-1 rounded-2xl bg-[#0a2e31] py-4 text-xs font-black tracking-widest text-white uppercase shadow-xl transition-all hover:bg-black active:scale-[0.98]"
						>
							Simpan Perubahan
						</button>
					</div>
				</form>
			</div>
		</div>
	</div>
{/if}

<!-- Modal History & Transaction -->
{#if showHistoryModal && selectedAsset}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-[#0a2e31]/40 p-4 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
	>
		<div
			class="flex max-h-[90vh] w-full max-w-4xl flex-col overflow-hidden rounded-[3rem] bg-gray-50 shadow-2xl"
			in:fly={{ y: 20, duration: 400 }}
		>
			<!-- Modal Header -->
			<div class="flex items-center justify-between border-b border-gray-100 bg-white p-8">
				<div>
					<h2 class="text-2xl font-black text-[#0a2e31]">{selectedAsset.name}</h2>
					<p class="text-xs font-bold text-gray-400">
						{selectedAsset.symbol ?? '—'} — {selectedAsset.asset_type}
					</p>
				</div>
				<button
					aria-label="Tutup riwayat"
					onclick={() => (showHistoryModal = false)}
					class="p-2 text-gray-300 hover:text-gray-500"
				>
					<svg
						class="h-6 w-6"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2.5"><path d="M6 18L18 6M6 6l12 12" /></svg
					>
				</button>
			</div>

			<div class="grid flex-1 grid-cols-1 gap-8 overflow-y-auto p-8 lg:grid-cols-5">
				<!-- History List -->
				<div class="space-y-6 lg:col-span-3">
					<h3 class="text-sm font-black tracking-widest text-[#0a2e31] uppercase">
						Riwayat Perubahan Unit
					</h3>

					{#if historyLoading}
						<div class="space-y-4">
							{#each [0, 1, 2] as i (i)}
								<div class="h-20 animate-pulse rounded-2xl bg-white"></div>
							{/each}
						</div>
					{:else if history.length === 0}
						<div class="rounded-3xl border border-gray-100 bg-white p-12 text-center">
							<p class="text-sm font-bold text-gray-400">Belum ada riwayat transaksi.</p>
						</div>
					{:else}
						<div class="space-y-3">
							{#each history as entry (entry.id)}
								<div
									class="group flex items-center justify-between rounded-2xl border border-gray-100 bg-white p-5 transition-colors hover:border-teal-200"
								>
									<div class="flex items-center gap-4">
										<div
											class="flex h-10 w-10 items-center justify-center rounded-xl text-[10px] font-black {entry.transaction_type ===
											'BUY'
												? 'bg-green-50 text-green-600'
												: 'bg-red-50 text-red-500'}"
										>
											{entry.transaction_type}
										</div>
										<div>
											<p class="text-sm font-black text-[#0a2e31]">
												{formatNumber(entry.quantity_change)} Unit
											</p>
											<p class="text-[10px] font-bold text-gray-400">
												{entry.transaction_date}
											</p>
										</div>
									</div>
									<div class="text-right">
										<p class="text-sm font-bold text-[#0a2e31]">
											{formatCurrency(entry.price_per_unit)}
										</p>
										<p class="max-w-[100px] truncate text-[10px] font-medium text-gray-400 italic">
											{entry.notes ?? '-'}
										</p>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>

				<!-- Record Form -->
				<div class="lg:col-span-2">
					<div
						class="sticky top-0 space-y-6 rounded-3xl border border-gray-100 bg-white p-8 shadow-sm"
					>
						<h3 class="text-sm font-black tracking-widest text-[#0a2e31] uppercase">
							Catat Transaksi Baru
						</h3>

						<form onsubmit={handleRecordHistory} class="space-y-4">
							<div class="flex rounded-xl bg-gray-50 p-1">
								<button
									type="button"
									onclick={() => (historyForm.transaction_type = 'BUY')}
									class="flex-1 rounded-lg py-2 text-[10px] font-black tracking-widest uppercase transition-all {historyForm.transaction_type ===
									'BUY'
										? 'bg-white text-green-600 shadow-sm'
										: 'text-gray-400'}"
								>
									Beli
								</button>
								<button
									type="button"
									onclick={() => (historyForm.transaction_type = 'SELL')}
									class="flex-1 rounded-lg py-2 text-[10px] font-black tracking-widest uppercase transition-all {historyForm.transaction_type ===
									'SELL'
										? 'bg-white text-red-500 shadow-sm'
										: 'text-gray-400'}"
								>
									Jual
								</button>
							</div>

							<div class="space-y-1.5">
								<label
									for="h-qty"
									class="px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
									>Perubahan Kuantitas</label
								>
								<input
									id="h-qty"
									type="number"
									step="any"
									required
									bind:value={historyForm.quantity_change}
									class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-2.5 text-sm font-bold transition-all outline-none focus:ring-2 focus:ring-teal-500"
								/>
							</div>

							<div class="space-y-1.5">
								<label
									for="h-price"
									class="px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
									>Harga per Unit</label
								>
								<input
									id="h-price"
									type="number"
									step="any"
									required
									bind:value={historyForm.price_per_unit}
									class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-2.5 text-sm font-bold transition-all outline-none focus:ring-2 focus:ring-teal-500"
								/>
							</div>

							<div class="space-y-1.5">
								<label
									for="h-date"
									class="px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
									>Tanggal</label
								>
								<input
									id="h-date"
									type="date"
									required
									bind:value={historyForm.transaction_date}
									class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-2.5 text-sm font-bold transition-all outline-none focus:ring-2 focus:ring-teal-500"
								/>
							</div>

							<div class="space-y-1.5">
								<label
									for="h-notes"
									class="px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
									>Catatan</label
								>
								<textarea
									id="h-notes"
									bind:value={historyForm.notes}
									rows="2"
									class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-2.5 text-sm font-bold transition-all outline-none focus:ring-2 focus:ring-teal-500"
								></textarea>
							</div>

							<button
								type="submit"
								class="w-full rounded-xl bg-[#0a2e31] py-3.5 text-[10px] font-black tracking-widest text-white uppercase shadow-lg transition-all hover:bg-black active:scale-[0.98]"
							>
								Simpan Transaksi
							</button>
						</form>
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	::-webkit-scrollbar {
		width: 6px;
	}
	::-webkit-scrollbar-track {
		background: transparent;
	}
	::-webkit-scrollbar-thumb {
		background: #e5e7eb;
		border-radius: 10px;
	}
	::-webkit-scrollbar-thumb:hover {
		background: #d1d5db;
	}
</style>
