<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { fade, fly } from 'svelte/transition';

	type AssetType = 'CRYPTO' | 'STOCK' | 'MUTUAL_FUND' | 'GOLD' | 'OTHER';

	// Model lokal (online). Field diselaraskan dari response backend:
	// backend `quantity` -> units, `avg_buy_price` -> avg_buy_price_idr.
	type Asset = {
		id: string;
		name: string;
		asset_type: AssetType;
		symbol: string | null;
		units: number;
		avg_buy_price_idr: number;
		sort_order: number;
	};

	// Server dapat mengirim snake_case (default) atau PascalCase — normalisasi defensif.
	type ServerAsset = {
		id?: string;
		ID?: string;
		name?: string;
		Name?: string;
		asset_type?: AssetType;
		AssetType?: AssetType;
		symbol?: string;
		Symbol?: string;
		quantity?: number;
		Quantity?: number;
		avg_buy_price?: number;
		AvgBuyPrice?: number;
		sort_order?: number;
		SortOrder?: number;
	};

	type HistoryEntry = {
		id: string;
		transaction_type: 'BUY' | 'SELL';
		quantity_change: number;
		price_per_unit: number;
		total_value: number;
		transaction_date: string;
		notes?: string;
	};

	type ServerHistory = {
		id?: string;
		ID?: string;
		transaction_type?: 'BUY' | 'SELL';
		TransactionType?: 'BUY' | 'SELL';
		quantity_change?: number;
		QuantityChange?: number;
		price_per_unit?: number;
		PricePerUnit?: number;
		total_value?: number;
		TotalValue?: number;
		transaction_date?: string;
		TransactionDate?: string;
		notes?: string;
		Notes?: string;
	};

	let assets = $state<Asset[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let errorMessage = $state('');
	let showAssetModal = $state(false);
	let showHistoryModal = $state(false);
	let selectedAsset = $state<Asset | null>(null);
	let history = $state<HistoryEntry[]>([]);
	let historyLoading = $state(false);

	// Price state
	let priceLoading = $state<Record<string, boolean>>({});
	let priceCache = $state<Record<string, number>>({});
	let priceFresh = $state<Record<string, boolean>>({});

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

	function normalizeAsset(item: ServerAsset): Asset | null {
		const id = item.id ?? item.ID;
		const name = item.name ?? item.Name;
		const assetType = item.asset_type ?? item.AssetType;
		if (!id || !name || !assetType) return null;

		const symbol = item.symbol ?? item.Symbol ?? '';
		return {
			id,
			name,
			asset_type: assetType,
			symbol: symbol || null,
			units: Number(item.quantity ?? item.Quantity ?? 0),
			avg_buy_price_idr: Number(item.avg_buy_price ?? item.AvgBuyPrice ?? 0),
			sort_order: Number(item.sort_order ?? item.SortOrder ?? 0)
		};
	}

	function normalizeAssets(data: unknown): Asset[] {
		if (!Array.isArray(data)) return [];
		return data
			.map((item) => normalizeAsset(item as ServerAsset))
			.filter((item): item is Asset => item !== null);
	}

	function normalizeHistory(data: unknown): HistoryEntry[] {
		if (!Array.isArray(data)) return [];
		const result: HistoryEntry[] = [];
		for (const item of data) {
			const h = item as ServerHistory;
			const id = h.id ?? h.ID;
			const type = h.transaction_type ?? h.TransactionType;
			const date = h.transaction_date ?? h.TransactionDate;
			if (!id || !type || !date) continue;
			result.push({
				id,
				transaction_type: type,
				quantity_change: Number(h.quantity_change ?? h.QuantityChange ?? 0),
				price_per_unit: Number(h.price_per_unit ?? h.PricePerUnit ?? 0),
				total_value: Number(h.total_value ?? h.TotalValue ?? 0),
				transaction_date: date,
				notes: h.notes ?? h.Notes
			});
		}
		return result;
	}

	async function fetchAssets() {
		loading = true;
		errorMessage = '';
		try {
			const res = await apiFetch('/investments');
			const result = await readApiResult(res);
			if (res.ok && result.success) {
				assets = normalizeAssets(result.data);
				void fetchAllPrices(assets);
			} else {
				errorMessage = result.error?.message || 'Gagal memuat instrumen investasi.';
			}
		} catch (err) {
			console.error(err);
			errorMessage = 'Gagal memuat instrumen. Periksa koneksi atau service backend.';
		} finally {
			loading = false;
		}
	}

	async function fetchPrice(asset: Asset) {
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

	async function fetchAllPrices(list: Asset[]) {
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
		sort_order: number; // dipertahankan saat edit agar tidak ter-reset ke 0
	};

	let assetForm = $state<AssetFormState>({
		id: '',
		name: '',
		asset_type: 'CRYPTO',
		symbol: '',
		units: 0,
		nilai_pembelian: 0,
		sort_order: 0
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
		{
			value: 'GOLD',
			label: 'Emas',
			hint: 'Simbol: tether-gold · Satuan units = gram · Harga live per gram (otomatis)'
		}
	];

	const selectedAssetTypeHint = $derived(
		assetTypes.find((t) => t.value === assetForm.asset_type)?.hint ?? ''
	);

	async function fetchHistory(assetId: string) {
		historyLoading = true;
		try {
			const res = await apiFetch(`/investments/${assetId}/history`);
			const result = await readApiResult(res);
			if (res.ok && result.success) history = normalizeHistory(result.data);
			else history = [];
		} catch {
			history = [];
		} finally {
			historyLoading = false;
		}
	}

	async function handleSaveAsset(e: SubmitEvent) {
		e.preventDefault();
		errorMessage = '';
		try {
			saving = true;
			const symbol = assetForm.symbol.trim();
			let res: Response;
			if (assetForm.id) {
				res = await apiFetch(`/investments/${assetForm.id}`, {
					method: 'PUT',
					body: JSON.stringify({
						name: assetForm.name.trim(),
						asset_type: assetForm.asset_type,
						symbol,
						currency: 'IDR',
						sort_order: assetForm.sort_order
					})
				});
			} else {
				const avg_buy_price = assetForm.units > 0 ? assetForm.nilai_pembelian / assetForm.units : 0;
				res = await apiFetch('/investments', {
					method: 'POST',
					body: JSON.stringify({
						name: assetForm.name.trim(),
						asset_type: assetForm.asset_type,
						symbol,
						quantity: assetForm.units,
						avg_buy_price,
						currency: 'IDR'
					})
				});
			}
			const result = await readApiResult(res);
			if (res.ok && result.success) {
				showAssetModal = false;
				await fetchAssets();
			} else {
				errorMessage = result.error?.message || 'Gagal menyimpan instrumen.';
			}
		} catch (err) {
			console.error(err);
			errorMessage = 'Gagal menyimpan instrumen. Periksa koneksi atau service backend.';
		} finally {
			saving = false;
		}
	}

	async function handleDeleteAsset(id: string) {
		if (!confirm('Hapus instrumen ini? Riwayat transaksi juga akan terhapus.')) return;
		errorMessage = '';
		try {
			saving = true;
			const res = await apiFetch(`/investments/${id}`, { method: 'DELETE' });
			const result = await readApiResult(res);
			if (res.ok && result.success) {
				showAssetModal = false;
				await fetchAssets();
			} else {
				errorMessage = result.error?.message || 'Gagal menghapus instrumen.';
			}
		} catch (err) {
			console.error(err);
			errorMessage = 'Gagal menghapus instrumen. Periksa koneksi atau service backend.';
		} finally {
			saving = false;
		}
	}

	async function handleRecordHistory(e: SubmitEvent) {
		e.preventDefault();
		if (!selectedAsset) return;
		errorMessage = '';
		const price_per_unit =
			historyForm.quantity_change > 0 ? historyForm.nilai_total / historyForm.quantity_change : 0;
		try {
			saving = true;
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
			const result = await readApiResult(res);
			if (res.ok && result.success) {
				historyForm = {
					transaction_type: 'BUY',
					quantity_change: 0,
					nilai_total: 0,
					transaction_date: new Date().toISOString().split('T')[0],
					notes: ''
				};
				await fetchHistory(selectedAsset.id);
				// Unit berubah → segarkan daftar aset (quantity & avg dihitung ulang server).
				await fetchAssets();
			} else {
				errorMessage = result.error?.message || 'Gagal menyimpan transaksi.';
			}
		} catch (err) {
			console.error(err);
			errorMessage = 'Gagal menyimpan transaksi. Periksa koneksi atau service backend.';
		} finally {
			saving = false;
		}
	}

	function openAddModal() {
		errorMessage = '';
		assetForm = {
			id: '',
			name: '',
			asset_type: 'CRYPTO',
			symbol: '',
			units: 0,
			nilai_pembelian: 0,
			sort_order: 0
		};
		showAssetModal = true;
	}

	function openEditModal(asset: Asset) {
		errorMessage = '';
		assetForm = {
			id: asset.id,
			name: asset.name,
			asset_type: asset.asset_type,
			symbol: asset.symbol ?? '',
			units: asset.units,
			nilai_pembelian: asset.units * asset.avg_buy_price_idr,
			sort_order: asset.sort_order
		};
		showAssetModal = true;
	}

	function openHistoryModal(asset: Asset) {
		errorMessage = '';
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

	onMount(fetchAssets);

	// Derived totals
	const totalCurrentValue = $derived(
		assets.reduce((acc, a) => {
			const price = priceCache[a.id] ?? a.avg_buy_price_idr;
			return acc + a.units * price;
		}, 0)
	);

	const totalCostBasis = $derived(
		assets.reduce((acc, a) => acc + a.units * a.avg_buy_price_idr, 0)
	);
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

<div class="pb-4">
	<!-- ═══════════ Header & Summary ═══════════ -->
	<div
		class="flex flex-col justify-between gap-6 border-b border-ink/10 pb-8 md:flex-row md:items-end"
	>
		<div>
			<p class="mb-2 text-[12px] font-semibold tracking-[0.14em] text-ink/45 uppercase">
				Nilai portofolio
				{#if assetsWithPrice > 0}
					<span class="text-teal">· ● live</span>
				{/if}
			</p>
			<p class="font-serif text-5xl leading-none tracking-tight text-ink tabular-nums sm:text-6xl">
				{formatCurrency(totalCurrentValue)}
			</p>
			{#if totalCostBasis > 0}
				<p class="mt-3 text-sm text-ink/60">
					Modal {formatCurrency(totalCostBasis)} ·
					<span class="{totalGainLoss >= 0 ? 'text-teal' : 'text-clay'} font-semibold">
						{totalGainLoss >= 0 ? 'untung' : 'rugi'}
						{totalGainLoss >= 0 ? '+' : ''}{formatCurrency(totalGainLoss)} ({formatPct(
							totalGainLossPct
						)})
					</span>
				</p>
			{/if}
		</div>
		<button
			onclick={openAddModal}
			class="shrink-0 rounded-full bg-teal px-5 py-2.5 text-[13px] font-semibold text-card transition-colors hover:bg-ink"
		>
			+ Tambah instrumen
		</button>
	</div>

	{#if errorMessage && !showAssetModal && !showHistoryModal}
		<div class="mt-6 rounded-[10px] border border-clay/25 bg-clay/5 px-4 py-3 text-sm text-clay">
			{errorMessage}
		</div>
	{/if}

	<!-- Assets -->
	{#if loading}
		<div class="space-y-3 pt-6">
			{#each [0, 1, 2] as i (i)}
				<div class="h-12 animate-pulse rounded bg-ink/5"></div>
			{/each}
		</div>
	{:else if assets.length === 0}
		<div class="flex flex-col items-center gap-3 py-20 text-center">
			<p class="font-serif text-2xl text-ink">Belum ada instrumen investasi</p>
			<p class="text-sm text-ink/45">Klik "+ Tambah instrumen" untuk mulai mencatat aset Anda.</p>
		</div>
	{:else}
		<!-- Desktop: editorial table -->
		<div class="hidden pt-4 lg:block">
			<div
				class="grid grid-cols-[1.4fr_0.8fr_1fr_1fr_1fr_1.1fr_auto] gap-4 border-b border-ink/25 py-3 text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
			>
				<span>Aset</span>
				<span>Kepemilikan</span>
				<span class="text-right">Modal</span>
				<span class="text-right">Harga live</span>
				<span class="text-right">Nilai sekarang</span>
				<span class="text-right">Untung / rugi</span>
				<span></span>
			</div>
			{#each assets as asset (asset.id)}
				{@const livePrice = priceCache[asset.id]}
				{@const currentValue = livePrice !== undefined ? livePrice * asset.units : null}
				{@const costBasis = asset.avg_buy_price_idr * asset.units}
				{@const gainLoss = currentValue !== null ? currentValue - costBasis : null}
				{@const gainLossPct =
					gainLoss !== null && costBasis > 0 ? (gainLoss / costBasis) * 100 : null}
				{@const unitLabel = asset.asset_type === 'GOLD' ? 'gram' : 'unit'}

				<!-- svelte-ignore a11y_click_events_have_key_events -->
				<div
					class="group grid grid-cols-[1.4fr_0.8fr_1fr_1fr_1fr_1.1fr_auto] items-baseline gap-4 border-b border-ink/8 py-5 text-[13.5px]"
					onclick={() => openHistoryModal(asset)}
					role="button"
					tabindex="0"
				>
					<span class="cursor-pointer">
						<span class="font-serif text-[19px] text-ink">{asset.name}</span>
						<span class="text-[11px] text-ink/45">· {asset.symbol ?? '—'} · {asset.asset_type}</span
						>
					</span>
					<span class="text-ink tabular-nums">{formatNumber(asset.units)} {unitLabel}</span>
					<span class="text-right text-ink/60 tabular-nums">{formatNumber(costBasis, 0)}</span>
					<span class="text-right tabular-nums">
						{#if priceLoading[asset.id]}
							<span class="text-ink/35">…</span>
						{:else if livePrice !== undefined}
							{formatNumber(livePrice, 0)}
							<span
								class="text-[10px] {priceFresh[asset.id] ? 'text-teal' : 'text-gold'}"
								title={priceFresh[asset.id] ? 'Data segar' : 'Data dari cache'}>●</span
							>
						{:else}
							<span class="text-ink/35">—</span>
						{/if}
					</span>
					<span class="text-right font-semibold text-ink tabular-nums">
						{currentValue !== null ? formatNumber(currentValue, 0) : '—'}
					</span>
					<span
						class="text-right tabular-nums {gainLoss !== null && gainLoss >= 0
							? 'font-semibold text-teal'
							: gainLoss !== null
								? 'font-semibold text-clay'
								: 'text-ink/35'}"
					>
						{#if gainLoss !== null && gainLossPct !== null && costBasis > 0}
							{formatPct(gainLossPct)} · {gainLoss >= 0 ? '+' : ''}{formatNumber(gainLoss, 0)}
						{:else}
							—
						{/if}
					</span>
					<span
						class="flex justify-end gap-0.5 opacity-0 transition-opacity group-hover:opacity-100"
					>
						<button
							onclick={(e) => {
								e.stopPropagation();
								fetchPrice(asset);
							}}
							disabled={priceLoading[asset.id] || !asset.symbol}
							class="p-1 text-ink/30 transition-colors hover:text-teal disabled:opacity-40"
							title="Perbarui harga"
							aria-label="Perbarui harga {asset.name}"
						>
							<svg
								class="h-4 w-4 {priceLoading[asset.id] ? 'animate-spin' : ''}"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="2"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
								/>
							</svg>
						</button>
						<button
							onclick={(e) => {
								e.stopPropagation();
								openEditModal(asset);
							}}
							class="p-1 text-ink/30 transition-colors hover:text-ink"
							aria-label="Edit {asset.name}"
						>
							<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
								/>
							</svg>
						</button>
					</span>
				</div>
			{/each}
			<p class="mt-3.5 text-[11.5px] text-ink/45">
				<span class="text-teal">●</span> harga segar · <span class="text-gold">●</span> dari cache — klik
				baris untuk riwayat beli/jual
			</p>
		</div>

		<!-- Mobile: editorial list -->
		<div class="pt-4 lg:hidden">
			{#each assets as asset (asset.id)}
				{@const livePrice = priceCache[asset.id]}
				{@const currentValue = livePrice !== undefined ? livePrice * asset.units : null}
				{@const costBasis = asset.avg_buy_price_idr * asset.units}
				{@const gainLoss = currentValue !== null ? currentValue - costBasis : null}
				{@const gainLossPct =
					gainLoss !== null && costBasis > 0 ? (gainLoss / costBasis) * 100 : null}
				{@const unitLabel = asset.asset_type === 'GOLD' ? 'gram' : 'unit'}

				<div class="border-b border-ink/8 py-4">
					<div class="flex items-baseline justify-between gap-3">
						<button class="min-w-0 text-left" onclick={() => openHistoryModal(asset)}>
							<span class="font-serif text-[19px] text-ink">{asset.name}</span>
							<p class="text-[11px] text-ink/45">
								{asset.symbol ?? '—'} · {asset.asset_type} · {formatNumber(asset.units)}
								{unitLabel}
							</p>
						</button>
						<div class="flex shrink-0 items-center gap-1">
							<button
								onclick={() => fetchPrice(asset)}
								disabled={priceLoading[asset.id] || !asset.symbol}
								class="p-1 text-ink/30 transition-colors hover:text-teal disabled:opacity-40"
								title="Perbarui harga"
								aria-label="Perbarui harga {asset.name}"
							>
								<svg
									class="h-4 w-4 {priceLoading[asset.id] ? 'animate-spin' : ''}"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="2"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
									/>
								</svg>
							</button>
							<button
								onclick={() => openEditModal(asset)}
								class="p-1 text-ink/30 transition-colors hover:text-ink"
								aria-label="Edit {asset.name}"
							>
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
									/>
								</svg>
							</button>
						</div>
					</div>
					<div class="mt-2 flex items-baseline justify-between text-[13px]">
						<span class="text-ink/60">
							{#if livePrice !== undefined}
								{formatCurrency(currentValue ?? 0)}
								<span
									class="text-[10px] {priceFresh[asset.id] ? 'text-teal' : 'text-gold'}"
									title={priceFresh[asset.id] ? 'Data segar' : 'Data dari cache'}>●</span
								>
							{:else}
								Modal {formatCurrency(costBasis)}
							{/if}
						</span>
						{#if gainLoss !== null && gainLossPct !== null && costBasis > 0}
							<span
								class="tabular-nums {gainLoss >= 0
									? 'font-semibold text-teal'
									: 'font-semibold text-clay'}"
							>
								{formatPct(gainLossPct)} · {gainLoss >= 0 ? '+' : ''}{formatNumber(gainLoss, 0)}
							</span>
						{/if}
					</div>
				</div>
			{/each}
			<p class="mt-3.5 text-[11.5px] text-ink/45">
				<span class="text-teal">●</span> harga segar · <span class="text-gold">●</span> dari cache — ketuk
				aset untuk riwayat.
			</p>
		</div>
	{/if}
</div>

<!-- Modal Tambah / Edit Instrumen -->
{#if showAssetModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
	>
		<div
			class="w-full max-w-lg overflow-hidden rounded-2xl border border-ink/10 bg-card shadow-xl"
			in:fly={{ y: 20, duration: 400 }}
		>
			<div class="space-y-6 p-8 sm:p-10">
				<div class="flex items-center justify-between">
					<h2 class="font-serif text-2xl text-ink">
						{assetForm.id ? 'Edit instrumen' : 'Tambah instrumen'}
					</h2>
					<button
						aria-label="Tutup modal"
						onclick={() => (showAssetModal = false)}
						class="rounded-full p-2 text-ink/40 transition-colors hover:bg-ink/5 hover:text-ink"
					>
						<svg
							class="h-5 w-5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"><path d="M6 18L18 6M6 6l12 12" /></svg
						>
					</button>
				</div>

				<form onsubmit={handleSaveAsset} class="space-y-5">
					<!-- Nama -->
					<div>
						<label
							for="name"
							class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
							>Nama aset</label
						>
						<input
							id="name"
							type="text"
							required
							bind:value={assetForm.name}
							placeholder="Contoh: Bitcoin, BBCA, Emas"
							class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink outline-none focus:border-teal"
						/>
					</div>

					<div class="grid grid-cols-2 gap-4">
						<!-- Tipe -->
						<div>
							<label
								for="type"
								class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>Tipe aset</label
							>
							<select
								id="type"
								bind:value={assetForm.asset_type}
								class="w-full cursor-pointer appearance-none rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink outline-none focus:border-teal"
							>
								{#each assetTypes as type (type.value)}
									<option value={type.value}>{type.label}</option>
								{/each}
							</select>
						</div>

						<!-- Simbol -->
						<div>
							<label
								for="symbol"
								class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
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
								class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink outline-none focus:border-teal"
							/>
						</div>
					</div>

					{#if selectedAssetTypeHint}
						<p class="text-[12px] text-teal">
							{selectedAssetTypeHint}
						</p>
					{/if}

					<!-- Pembelian awal (hanya saat tambah) -->
					{#if !assetForm.id}
						<div class="space-y-4 rounded-[10px] border border-ink/10 bg-field p-5">
							<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
								Pembelian awal
							</p>

							<div class="grid grid-cols-2 gap-4">
								<div>
									<label
										for="qty"
										class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
										>{assetForm.asset_type === 'GOLD' ? 'Jumlah (gram)' : 'Jumlah unit'}</label
									>
									<input
										id="qty"
										type="number"
										step="any"
										min="0"
										bind:value={assetForm.units}
										placeholder={assetForm.asset_type === 'GOLD' ? '10' : '0.00004408'}
										class="w-full rounded-[10px] border border-ink/25 bg-card px-4 py-3 text-sm text-ink tabular-nums outline-none focus:border-teal"
									/>
								</div>

								<div>
									<label
										for="nilai"
										class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
										>Nilai pembelian (Rp)</label
									>
									<input
										id="nilai"
										type="number"
										step="any"
										min="0"
										bind:value={assetForm.nilai_pembelian}
										placeholder="49998"
										class="w-full rounded-[10px] border border-ink/25 bg-card px-4 py-3 text-sm text-ink tabular-nums outline-none focus:border-teal"
									/>
								</div>
							</div>

							{#if previewAvgPrice > 0}
								<div class="flex items-center justify-between border-t border-ink/10 pt-3">
									<span class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
										>Harga rata-rata</span
									>
									<span class="text-sm font-semibold text-teal tabular-nums"
										>{formatCurrency(previewAvgPrice)}/unit</span
									>
								</div>
							{/if}
						</div>
					{/if}

					{#if errorMessage}
						<div class="rounded-[10px] border border-clay/25 bg-clay/5 px-4 py-3 text-xs text-clay">
							{errorMessage}
						</div>
					{/if}

					<div class="flex gap-3 pt-1">
						{#if assetForm.id}
							<button
								type="button"
								onclick={() => handleDeleteAsset(assetForm.id)}
								disabled={saving}
								class="rounded-full border border-ink/15 px-5 py-3 text-[13px] font-semibold text-clay transition-colors hover:border-clay/30 hover:bg-clay/5 disabled:opacity-60"
							>
								Hapus
							</button>
						{/if}
						<button
							type="submit"
							disabled={saving}
							class="flex-1 rounded-full bg-teal py-3.5 text-sm font-semibold text-card transition-colors hover:bg-ink disabled:cursor-not-allowed disabled:opacity-60"
						>
							{saving ? 'Menyimpan…' : assetForm.id ? 'Simpan perubahan' : 'Tambah aset'}
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
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
	>
		<div
			class="flex max-h-[90vh] w-full max-w-4xl flex-col overflow-hidden rounded-2xl border border-ink/10 bg-card shadow-xl"
			in:fly={{ y: 20, duration: 400 }}
		>
			<!-- Header -->
			<div class="flex items-center justify-between border-b border-ink/10 px-8 py-6">
				<div>
					<h2 class="font-serif text-2xl text-ink">{selectedAsset.name}</h2>
					<p class="mt-0.5 text-[12px] text-ink/50">
						{selectedAsset.symbol ?? '—'} &middot; {selectedAsset.asset_type} &middot;
						{formatNumber(selectedAsset.units)}
						{selectedAsset.asset_type === 'GOLD' ? 'gram' : 'unit'}
					</p>
				</div>
				<button
					aria-label="Tutup riwayat"
					onclick={() => (showHistoryModal = false)}
					class="rounded-full p-2 text-ink/40 transition-colors hover:bg-ink/5 hover:text-ink"
				>
					<svg
						class="h-5 w-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"><path d="M6 18L18 6M6 6l12 12" /></svg
					>
				</button>
			</div>

			<div class="grid flex-1 grid-cols-1 gap-8 overflow-y-auto p-8 lg:grid-cols-5">
				<!-- History List -->
				<div class="lg:col-span-3">
					<h3 class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
						Riwayat transaksi
					</h3>

					{#if historyLoading}
						<div class="space-y-3 pt-4">
							{#each [0, 1, 2] as i (i)}
								<div class="h-10 animate-pulse rounded bg-ink/5"></div>
							{/each}
						</div>
					{:else if history.length === 0}
						<p class="pt-6 text-sm text-ink/45">Belum ada riwayat transaksi.</p>
					{:else}
						<div class="pt-2">
							{#each history as entry (entry.id)}
								<div class="flex items-baseline justify-between gap-3 border-b border-ink/8 py-4">
									<div class="flex items-baseline gap-3">
										<span
											class="shrink-0 rounded-full border px-2 py-px text-[11px] font-semibold {entry.transaction_type ===
											'BUY'
												? 'border-teal/30 text-teal'
												: 'border-clay/30 text-clay'}"
										>
											{entry.transaction_type === 'BUY' ? 'Beli' : 'Jual'}
										</span>
										<div>
											<p class="text-sm font-medium text-ink tabular-nums">
												{formatNumber(entry.quantity_change)} unit
											</p>
											<p class="mt-0.5 text-[11px] text-ink/45">
												{new Date(entry.transaction_date).toLocaleDateString('id-ID', {
													day: 'numeric',
													month: 'short',
													year: 'numeric'
												})}
											</p>
										</div>
									</div>
									<div class="text-right">
										<p class="text-sm font-medium text-ink tabular-nums">
											{formatCurrency(entry.total_value)}
										</p>
										<p class="mt-0.5 text-[11px] text-ink/45 tabular-nums">
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
					<div class="sticky top-0 rounded-[10px] border border-ink/10 bg-field p-6">
						<h3 class="mb-4 text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
							Catat transaksi
						</h3>

						<form onsubmit={handleRecordHistory} class="space-y-4">
							<!-- BUY / SELL Toggle -->
							<div class="flex rounded-full border border-ink/20 p-1 text-[13px] font-semibold">
								<button
									type="button"
									onclick={() => (historyForm.transaction_type = 'BUY')}
									class="flex-1 rounded-full py-2 transition-colors {historyForm.transaction_type ===
									'BUY'
										? 'bg-ink text-card'
										: 'text-ink/60'}"
								>
									Beli
								</button>
								<button
									type="button"
									onclick={() => (historyForm.transaction_type = 'SELL')}
									class="flex-1 rounded-full py-2 transition-colors {historyForm.transaction_type ===
									'SELL'
										? 'bg-ink text-card'
										: 'text-ink/60'}"
								>
									Jual
								</button>
							</div>

							<!-- Jumlah Unit -->
							<div>
								<label
									for="h-qty"
									class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
									>Jumlah unit</label
								>
								<input
									id="h-qty"
									type="number"
									step="any"
									min="0"
									required
									bind:value={historyForm.quantity_change}
									placeholder="0.00004408"
									class="w-full rounded-[10px] border border-ink/25 bg-card px-4 py-3 text-sm text-ink tabular-nums outline-none focus:border-teal"
								/>
							</div>

							<!-- Nilai Total Transaksi -->
							<div>
								<label
									for="h-nilai"
									class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
									>Nilai {historyForm.transaction_type === 'BUY' ? 'pembelian' : 'penjualan'} (Rp)</label
								>
								<input
									id="h-nilai"
									type="number"
									step="any"
									min="0"
									required
									bind:value={historyForm.nilai_total}
									placeholder="49998"
									class="w-full rounded-[10px] border border-ink/25 bg-card px-4 py-3 text-sm text-ink tabular-nums outline-none focus:border-teal"
								/>
							</div>

							<!-- Preview harga per unit -->
							{#if previewPricePerUnit > 0}
								<div class="flex items-center justify-between border-t border-ink/10 pt-3">
									<span class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
										>Harga/unit</span
									>
									<span class="text-sm font-semibold text-teal tabular-nums"
										>{formatCurrency(previewPricePerUnit)}</span
									>
								</div>
							{/if}

							<!-- Tanggal -->
							<div>
								<label
									for="h-date"
									class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
									>Tanggal</label
								>
								<input
									id="h-date"
									type="date"
									required
									bind:value={historyForm.transaction_date}
									class="w-full rounded-[10px] border border-ink/25 bg-card px-4 py-3 text-sm text-ink outline-none focus:border-teal"
								/>
							</div>

							<!-- Catatan -->
							<div>
								<label
									for="h-notes"
									class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
									>Catatan (opsional)</label
								>
								<input
									id="h-notes"
									type="text"
									bind:value={historyForm.notes}
									placeholder="No. Order 78313339"
									class="w-full rounded-[10px] border border-ink/25 bg-card px-4 py-3 text-sm text-ink outline-none focus:border-teal"
								/>
							</div>

							{#if errorMessage}
								<div
									class="rounded-[10px] border border-clay/25 bg-clay/5 px-4 py-3 text-xs text-clay"
								>
									{errorMessage}
								</div>
							{/if}

							<button
								type="submit"
								disabled={saving}
								class="w-full rounded-full bg-teal py-3.5 text-sm font-semibold text-card transition-colors hover:bg-ink disabled:cursor-not-allowed disabled:opacity-60"
							>
								{saving ? 'Menyimpan…' : 'Simpan transaksi'}
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
		background: rgba(233, 237, 244, 0.15);
		border-radius: 10px;
	}
	::-webkit-scrollbar-thumb:hover {
		background: rgba(233, 237, 244, 0.3);
	}
</style>
