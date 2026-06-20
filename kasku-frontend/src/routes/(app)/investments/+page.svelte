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
		total_value: number;
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

	// Price state
	let priceLoading = $state<Record<string, boolean>>({});
	let priceCache = $state<Record<string, number>>({});
	let priceFresh = $state<Record<string, boolean>>({});

	async function fetchPrice(asset: InvestmentRow) {
		if (!asset.symbol) return;
		priceLoading[asset.id] = true;
		try {
			const res = await apiFetch(`/prices/${asset.symbol}`);
			const result = await res.json();
			if (result.success && result.data) {
				priceCache[asset.id] = Number(result.data.price_idr);
				priceFresh[asset.id] = Boolean(result.data.is_fresh);
			}
		} catch {
			// gagal fetch harga — biarkan card tanpa live price
		} finally {
			priceLoading[asset.id] = false;
		}
	}

	async function fetchAllPrices(list: InvestmentRow[]) {
		await Promise.allSettled(list.filter((a) => a.symbol).map((a) => fetchPrice(a)));
	}

	// Form tambah/edit instrumen
	type AssetFormState = {
		id: string;
		name: string;
		asset_type: AssetType;
		symbol: string;
		units: number;
		nilai_pembelian: number; // Nilai total pembelian awal — avg_buy_price_idr dihitung dari ini
	};

	let assetForm = $state<AssetFormState>({
		id: '',
		name: '',
		asset_type: 'CRYPTO',
		symbol: '',
		units: 0,
		nilai_pembelian: 0
	});

	// Preview avg price di form add
	const previewAvgPrice = $derived(
		assetForm.units > 0 ? assetForm.nilai_pembelian / assetForm.units : 0
	);

	// Form catat transaksi
	let historyForm = $state({
		transaction_type: 'BUY' as 'BUY' | 'SELL',
		quantity_change: 0,
		nilai_total: 0, // Nilai total transaksi — price_per_unit dihitung dari ini
		transaction_date: new Date().toISOString().split('T')[0],
		notes: ''
	});

	// Preview harga per unit di form transaksi
	const previewPricePerUnit = $derived(
		historyForm.quantity_change > 0 ? historyForm.nilai_total / historyForm.quantity_change : 0
	);

	const assetTypes: { value: AssetType; label: string; hint?: string }[] = [
		{ value: 'CRYPTO', label: 'Kripto', hint: 'Gunakan ID CoinGecko (bitcoin, ethereum)' },
		{ value: 'STOCK', label: 'Saham', hint: 'Kode saham (BBCA, TLKM)' },
		{ value: 'MUTUAL_FUND', label: 'Reksa Dana' },
		{ value: 'GOLD', label: 'Emas', hint: 'Simbol: tether-gold · Satuan units = gram · Harga live per gram (otomatis)' }
	];

	const selectedAssetTypeHint = $derived(
		assetTypes.find((t) => t.value === assetForm.asset_type)?.hint ?? ''
	);

	async function reloadFromLocal() {
		try {
			assets = await investmentsRepo.getAll();
		} catch {
			// biarkan assets kosong
		}
	}

	$effect(() => {
		void syncStatus.dataVersion;
		void reloadFromLocal();
	});

	async function fetchHistory(assetId: string) {
		historyLoading = true;
		try {
			const res = await apiFetch(`/investments/${assetId}/history`);
			const result = await res.json();
			if (result.success) history = (result.data ?? []) as HistoryEntry[];
		} catch {
			history = [];
		} finally {
			historyLoading = false;
		}
	}

	async function handleSaveAsset(e: SubmitEvent) {
		e.preventDefault();
		try {
			const avg_buy_price_idr =
				assetForm.units > 0 ? assetForm.nilai_pembelian / assetForm.units : 0;

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
					avg_buy_price_idr
				});
			}
			showAssetModal = false;
		} catch {
			// error ditangani oleh queue
		}
	}

	async function handleDeleteAsset(id: string) {
		if (!confirm('Hapus instrumen ini? Riwayat transaksi juga akan terhapus.')) return;
		try {
			await enqueueDelete('investments', id);
			showAssetModal = false;
		} catch {
			// error ditangani oleh queue
		}
	}

	async function handleRecordHistory(e: SubmitEvent) {
		e.preventDefault();
		if (!selectedAsset) return;
		const price_per_unit =
			historyForm.quantity_change > 0 ? historyForm.nilai_total / historyForm.quantity_change : 0;
		try {
			const res = await apiFetch(`/investments/${selectedAsset.id}/units`, {
				method: 'POST',
				body: JSON.stringify({
					transaction_type: historyForm.transaction_type,
					quantity_change: historyForm.quantity_change,
					price_per_unit,
					transaction_date: historyForm.transaction_date,
					notes: historyForm.notes
				})
			});
			const result = await res.json();
			if (result.success) {
				historyForm = {
					transaction_type: 'BUY',
					quantity_change: 0,
					nilai_total: 0,
					transaction_date: new Date().toISOString().split('T')[0],
					notes: ''
				};
				await fetchHistory(selectedAsset.id);
				void triggerManualSync();
			}
		} catch {
			// error UI handling bisa ditambahkan
		}
	}

	function openAddModal() {
		assetForm = { id: '', name: '', asset_type: 'CRYPTO', symbol: '', units: 0, nilai_pembelian: 0 };
		showAssetModal = true;
	}

	function openEditModal(asset: InvestmentRow) {
		assetForm = {
			id: asset.id,
			name: asset.name,
			asset_type: asset.asset_type,
			symbol: asset.symbol ?? '',
			units: asset.units,
			nilai_pembelian: asset.units * asset.avg_buy_price_idr
		};
		showAssetModal = true;
	}

	function openHistoryModal(asset: InvestmentRow) {
		selectedAsset = asset;
		history = [];
		historyForm = {
			transaction_type: 'BUY',
			quantity_change: 0,
			nilai_total: 0,
			transaction_date: new Date().toISOString().split('T')[0],
			notes: ''
		};
		void fetchHistory(asset.id);
		showHistoryModal = true;
	}

	onMount(async () => {
		await reloadFromLocal();
		loading = false;
		void triggerManualSync();
		void fetchAllPrices(assets);
	});

	// Derived totals
	const totalCurrentValue = $derived(
		assets.reduce((acc, a) => {
			const price = priceCache[a.id] ?? a.avg_buy_price_idr;
			return acc + a.units * price;
		}, 0)
	);

	const totalCostBasis = $derived(assets.reduce((acc, a) => acc + a.units * a.avg_buy_price_idr, 0));
	const totalGainLoss = $derived(totalCurrentValue - totalCostBasis);
	const totalGainLossPct = $derived(
		totalCostBasis > 0 ? (totalGainLoss / totalCostBasis) * 100 : 0
	);
	const assetsWithPrice = $derived(assets.filter((a) => priceCache[a.id] !== undefined).length);

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(val);
	}

	function formatNumber(val: number, decimals = 8) {
		return new Intl.NumberFormat('id-ID', { maximumFractionDigits: decimals }).format(val);
	}

	function formatPct(val: number) {
		return (val >= 0 ? '+' : '') + val.toFixed(2) + '%';
	}
</script>

<div class="animate-in fade-in space-y-8 pb-20 duration-500">
	<!-- Header & Summary -->
	<div class="flex flex-col justify-between gap-6 md:flex-row md:items-start">
		<div class="space-y-1">
			<h1 class="text-3xl font-black text-[#0a2e31]">Portofolio Investasi</h1>
			<p class="text-sm font-medium text-gray-500">
				Pantau nilai aset dan pertumbuhan portofolio Anda secara real-time.
			</p>
		</div>

		<!-- Summary Cards -->
		<div class="flex flex-col gap-3 sm:flex-row">
			<div
				class="flex flex-col items-end rounded-[2rem] bg-[#0a2e31] px-7 py-5 text-white shadow-xl shadow-teal-950/20"
			>
				<span class="mb-1 text-[10px] font-black tracking-[0.2em] text-teal-400/80 uppercase">
					Nilai Portofolio
					{#if assetsWithPrice > 0}
						<span class="ml-1 rounded-full bg-teal-700/40 px-1.5 py-0.5 text-[9px] text-teal-300"
							>Live</span
						>
					{/if}
				</span>
				<span class="text-2xl font-black tabular-nums">{formatCurrency(totalCurrentValue)}</span>
				{#if totalCostBasis > 0}
					<span
						class="mt-1 text-[11px] font-bold {totalGainLoss >= 0
							? 'text-green-400'
							: 'text-red-400'}"
					>
						{formatPct(totalGainLossPct)}
						({totalGainLoss >= 0 ? '+' : ''}{formatCurrency(totalGainLoss)})
					</span>
				{/if}
			</div>

			{#if totalCostBasis > 0}
				<div
					class="flex flex-col items-end rounded-[2rem] border border-gray-100 bg-white px-7 py-5 shadow-sm"
				>
					<span class="mb-1 text-[10px] font-black tracking-[0.2em] text-gray-400 uppercase"
						>Modal Awal</span
					>
					<span class="text-xl font-black tabular-nums text-[#0a2e31]"
						>{formatCurrency(totalCostBasis)}</span
					>
				</div>
			{/if}
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

	<!-- Assets Grid -->
	<div class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
		{#if loading}
			{#each [0, 1, 2] as i (i)}
				<div class="h-56 animate-pulse rounded-[2.5rem] border border-gray-100 bg-white"></div>
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
				{@const livePrice = priceCache[asset.id]}
				{@const currentValue = livePrice !== undefined ? livePrice * asset.units : null}
				{@const costBasis = asset.avg_buy_price_idr * asset.units}
				{@const gainLoss = currentValue !== null ? currentValue - costBasis : null}
				{@const gainLossPct =
					gainLoss !== null && costBasis > 0 ? (gainLoss / costBasis) * 100 : null}

				<div
					class="group relative overflow-hidden rounded-[2.5rem] border border-gray-100 bg-white p-7 shadow-sm transition-all hover:-translate-y-1 hover:shadow-xl"
				>
					<div
						class="absolute -top-4 -right-4 h-24 w-24 rounded-full bg-teal-50/50 transition-transform duration-700 group-hover:scale-125"
					></div>

					<div class="relative z-10 flex h-full flex-col gap-5">
						<!-- Asset Header -->
						<div class="flex items-start justify-between">
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

						<!-- Live Price Block -->
						{#if priceLoading[asset.id]}
							<div class="h-16 animate-pulse rounded-2xl bg-gray-50"></div>
						{:else if livePrice !== undefined}
							<div
								class="rounded-2xl border border-teal-100 bg-gradient-to-br from-teal-50 to-emerald-50 p-4"
							>
								<div class="mb-1 flex items-center justify-between">
									<span class="text-[10px] font-black tracking-widest text-teal-600 uppercase"
										>Harga Live</span
									>
									<span
										class="flex items-center gap-1 text-[10px] font-bold text-teal-500"
										title={priceFresh[asset.id] ? 'Data segar' : 'Data dari cache'}
									>
										<span
											class="inline-block h-1.5 w-1.5 rounded-full {priceFresh[asset.id]
												? 'bg-green-400'
												: 'bg-yellow-400'}"
										></span>
										{priceFresh[asset.id] ? 'Segar' : 'Cache'}
									</span>
								</div>
								<p class="text-lg font-black text-[#0a2e31]">{formatCurrency(livePrice)}</p>
							</div>
						{/if}

						<!-- Stats Row -->
						<div class="grid grid-cols-2 gap-3">
							<div class="space-y-0.5">
								<p class="text-[10px] font-bold tracking-widest text-gray-400 uppercase">
									Kepemilikan
								</p>
								<p class="text-base font-black text-[#0a2e31]">
									{formatNumber(asset.units)}
									<span class="ml-0.5 text-xs font-bold text-gray-400">{asset.asset_type === 'GOLD' ? 'gram' : 'Unit'}</span>
								</p>
							</div>

							{#if currentValue !== null}
								<div class="space-y-0.5 text-right">
									<p class="text-[10px] font-bold tracking-widest text-gray-400 uppercase">
										Nilai Sekarang
									</p>
									<p class="text-base font-black text-[#0a2e31]">{formatCurrency(currentValue)}</p>
								</div>
							{:else}
								<div class="space-y-0.5 text-right">
									<p class="text-[10px] font-bold tracking-widest text-gray-400 uppercase">
										Avg Beli
									</p>
									<p class="text-sm font-bold text-[#0a2e31]">
										{formatCurrency(asset.avg_buy_price_idr)}
									</p>
								</div>
							{/if}
						</div>

						<!-- Gain/Loss Badge -->
						{#if gainLoss !== null && gainLossPct !== null && costBasis > 0}
							<div
								class="flex items-center justify-between rounded-xl px-3 py-2 {gainLoss >= 0
									? 'bg-green-50 text-green-700'
									: 'bg-red-50 text-red-600'}"
							>
								<span class="text-[10px] font-black tracking-widest uppercase">
									{gainLoss >= 0 ? 'Untung' : 'Rugi'}
								</span>
								<div class="text-right">
									<span class="text-sm font-black">{formatPct(gainLossPct)}</span>
									<span class="ml-2 text-xs font-bold opacity-70"
										>{gainLoss >= 0 ? '+' : ''}{formatCurrency(gainLoss)}</span
									>
								</div>
							</div>
						{/if}

						<!-- Actions -->
						<div class="mt-auto flex gap-2">
							<button
								onclick={() => fetchPrice(asset)}
								disabled={priceLoading[asset.id] || !asset.symbol}
								class="rounded-xl bg-gray-50 p-3 text-[#0a2e31] transition-all hover:bg-gray-100 disabled:opacity-40"
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
								Riwayat & Transaksi
							</button>
						</div>
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>

<!-- Modal Tambah / Edit Instrumen -->
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

				<form onsubmit={handleSaveAsset} class="space-y-5">
					<!-- Nama -->
					<div class="space-y-2">
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
							placeholder="Contoh: Bitcoin, BBCA, Emas"
							class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-bold text-[#0a2e31] outline-none transition-all focus:ring-4 focus:ring-teal-50"
						/>
					</div>

					<div class="grid grid-cols-2 gap-4">
						<!-- Tipe -->
						<div class="space-y-2">
							<label
								for="type"
								class="block px-1 text-[11px] font-black tracking-widest text-gray-400 uppercase"
								>Tipe Aset</label
							>
							<select
								id="type"
								bind:value={assetForm.asset_type}
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-bold text-[#0a2e31] outline-none transition-all focus:ring-4 focus:ring-teal-50"
							>
								{#each assetTypes as type (type.value)}
									<option value={type.value}>{type.label}</option>
								{/each}
							</select>
						</div>

						<!-- Simbol -->
						<div class="space-y-2">
							<label
								for="symbol"
								class="block px-1 text-[11px] font-black tracking-widest text-gray-400 uppercase"
								>Simbol</label
							>
							<input
								id="symbol"
								type="text"
								bind:value={assetForm.symbol}
								placeholder={assetForm.asset_type === 'CRYPTO'
									? 'bitcoin'
									: assetForm.asset_type === 'GOLD'
										? 'tether-gold'
										: 'BBCA'}
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-bold text-[#0a2e31] outline-none transition-all focus:ring-4 focus:ring-teal-50"
							/>
						</div>
					</div>

					{#if selectedAssetTypeHint}
						<p class="px-1 text-[11px] font-medium text-teal-600">
							💡 {selectedAssetTypeHint}
						</p>
					{/if}

					<!-- Pembelian awal (hanya saat tambah) -->
					{#if !assetForm.id}
						<div class="space-y-4 rounded-2xl border border-gray-100 bg-gray-50 p-5">
							<p class="text-[11px] font-black tracking-widest text-gray-400 uppercase">
								Pembelian Awal
							</p>

							<div class="grid grid-cols-2 gap-4">
								<div class="space-y-2">
									<label
										for="qty"
										class="block text-[11px] font-black tracking-widest text-gray-400 uppercase"
										>{assetForm.asset_type === 'GOLD' ? 'Jumlah (gram)' : 'Jumlah Unit'}</label
									>
									<input
										id="qty"
										type="number"
										step="any"
										min="0"
										bind:value={assetForm.units}
										placeholder={assetForm.asset_type === 'GOLD' ? '10' : '0.00004408'}
										class="w-full rounded-xl border border-gray-100 bg-white px-4 py-3 font-bold text-[#0a2e31] outline-none transition-all focus:ring-2 focus:ring-teal-400"
									/>
								</div>

								<div class="space-y-2">
									<label
										for="nilai"
										class="block text-[11px] font-black tracking-widest text-gray-400 uppercase"
										>Nilai Pembelian (Rp)</label
									>
									<input
										id="nilai"
										type="number"
										step="any"
										min="0"
										bind:value={assetForm.nilai_pembelian}
										placeholder="49998"
										class="w-full rounded-xl border border-gray-100 bg-white px-4 py-3 font-bold text-[#0a2e31] outline-none transition-all focus:ring-2 focus:ring-teal-400"
									/>
								</div>
							</div>

							{#if previewAvgPrice > 0}
								<div class="flex items-center justify-between rounded-xl bg-teal-50 px-4 py-3">
									<span class="text-[10px] font-black tracking-widest text-teal-600 uppercase"
										>Harga Rata-Rata</span
									>
									<span class="text-sm font-black text-teal-800"
										>{formatCurrency(previewAvgPrice)}/Unit</span
									>
								</div>
							{/if}
						</div>
					{/if}

					<div class="flex gap-4 pt-2">
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
							{assetForm.id ? 'Simpan Perubahan' : 'Tambah Aset'}
						</button>
					</div>
				</form>
			</div>
		</div>
	</div>
{/if}

<!-- Modal Riwayat & Transaksi -->
{#if showHistoryModal && selectedAsset}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-[#0a2e31]/40 p-4 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
	>
		<div
			class="flex max-h-[90vh] w-full max-w-4xl flex-col overflow-hidden rounded-[3rem] bg-gray-50 shadow-2xl"
			in:fly={{ y: 20, duration: 400 }}
		>
			<!-- Header -->
			<div class="flex items-center justify-between border-b border-gray-100 bg-white p-8">
				<div>
					<h2 class="text-2xl font-black text-[#0a2e31]">{selectedAsset.name}</h2>
					<p class="text-xs font-bold text-gray-400">
						{selectedAsset.symbol ?? '—'} &middot; {selectedAsset.asset_type} &middot;
						{formatNumber(selectedAsset.units)} Unit
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
				<div class="space-y-4 lg:col-span-3">
					<h3 class="text-sm font-black tracking-widest text-[#0a2e31] uppercase">
						Riwayat Transaksi
					</h3>

					{#if historyLoading}
						<div class="space-y-3">
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
									class="flex items-center justify-between rounded-2xl border border-gray-100 bg-white p-5 transition-colors hover:border-teal-100"
								>
									<div class="flex items-center gap-4">
										<div
											class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl text-[10px] font-black {entry.transaction_type ===
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
												{new Date(entry.transaction_date).toLocaleDateString('id-ID', {
													day: 'numeric',
													month: 'short',
													year: 'numeric'
												})}
											</p>
										</div>
									</div>
									<div class="text-right">
										<p class="text-sm font-black text-[#0a2e31]">
											{formatCurrency(entry.total_value)}
										</p>
										<p class="text-[10px] font-bold text-gray-400">
											@{formatCurrency(entry.price_per_unit)}/unit
										</p>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>

				<!-- Form Catat Transaksi -->
				<div class="lg:col-span-2">
					<div
						class="sticky top-0 space-y-5 rounded-3xl border border-gray-100 bg-white p-7 shadow-sm"
					>
						<h3 class="text-sm font-black tracking-widest text-[#0a2e31] uppercase">
							Catat Transaksi
						</h3>

						<form onsubmit={handleRecordHistory} class="space-y-4">
							<!-- BUY / SELL Toggle -->
							<div class="flex rounded-xl bg-gray-50 p-1">
								<button
									type="button"
									onclick={() => (historyForm.transaction_type = 'BUY')}
									class="flex-1 rounded-lg py-2.5 text-[10px] font-black tracking-widest uppercase transition-all {historyForm.transaction_type ===
									'BUY'
										? 'bg-white text-green-600 shadow-sm'
										: 'text-gray-400'}"
								>
									Beli
								</button>
								<button
									type="button"
									onclick={() => (historyForm.transaction_type = 'SELL')}
									class="flex-1 rounded-lg py-2.5 text-[10px] font-black tracking-widest uppercase transition-all {historyForm.transaction_type ===
									'SELL'
										? 'bg-white text-red-500 shadow-sm'
										: 'text-gray-400'}"
								>
									Jual
								</button>
							</div>

							<!-- Jumlah Unit -->
							<div class="space-y-1.5">
								<label
									for="h-qty"
									class="px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
									>Jumlah Unit</label
								>
								<input
									id="h-qty"
									type="number"
									step="any"
									min="0"
									required
									bind:value={historyForm.quantity_change}
									placeholder="0.00004408"
									class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] outline-none transition-all focus:ring-2 focus:ring-teal-400"
								/>
							</div>

							<!-- Nilai Total Transaksi -->
							<div class="space-y-1.5">
								<label
									for="h-nilai"
									class="px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
									>Nilai {historyForm.transaction_type === 'BUY'
										? 'Pembelian'
										: 'Penjualan'} (Rp)</label
								>
								<input
									id="h-nilai"
									type="number"
									step="any"
									min="0"
									required
									bind:value={historyForm.nilai_total}
									placeholder="49998"
									class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] outline-none transition-all focus:ring-2 focus:ring-teal-400"
								/>
							</div>

							<!-- Preview harga per unit -->
							{#if previewPricePerUnit > 0}
								<div class="flex items-center justify-between rounded-xl bg-teal-50 px-4 py-2.5">
									<span class="text-[10px] font-black tracking-widest text-teal-600 uppercase"
										>Harga/Unit</span
									>
									<span class="text-sm font-black text-teal-800"
										>{formatCurrency(previewPricePerUnit)}</span
									>
								</div>
							{/if}

							<!-- Tanggal -->
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
									class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] outline-none transition-all focus:ring-2 focus:ring-teal-400"
								/>
							</div>

							<!-- Catatan -->
							<div class="space-y-1.5">
								<label
									for="h-notes"
									class="px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
									>Catatan (opsional)</label
								>
								<input
									id="h-notes"
									type="text"
									bind:value={historyForm.notes}
									placeholder="No. Order 78313339"
									class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] outline-none transition-all focus:ring-2 focus:ring-teal-400"
								/>
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