<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { fade, fly } from 'svelte/transition';

	let assets = $state<any[]>([]);
	let loading = $state(true);
	let showAssetModal = $state(false);
	let showHistoryModal = $state(false);
	let selectedAsset = $state<any>(null);
	let history = $state<any[]>([]);
	let historyLoading = $state(false);

	// Price Service State
	let priceLoading = $state<Record<string, boolean>>({});

	async function handleFetchPrice(asset: any) {
		priceLoading[asset.id] = true;
		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';

		if (isMock) {
			await new Promise(resolve => setTimeout(resolve, 800));
			const mockPrice = asset.avg_buy_price * (1 + (Math.random() * 0.2 - 0.1)); // random +- 10%
			assets = assets.map(a => a.id === asset.id ? { ...a, current_price: mockPrice } : a);
			priceLoading[asset.id] = false;
			return;
		}

		try {
			const res = await apiFetch(`/prices/${asset.symbol}`);
			const result = await res.json();
			if (result.success && result.data) {
				assets = assets.map(a => a.id === asset.id ? { ...a, current_price: result.data.price_idr } : a);
			}
		} catch (err) {
			console.error('Gagal mengambil harga:', err);
		} finally {
			priceLoading[asset.id] = false;
		}
	}

	// Form State for Asset
	let assetForm = $state({
		id: '',
		name: '',
		asset_type: 'STOCK',
		symbol: '',
		quantity: 0,
		avg_buy_price: 0,
		currency: 'IDR'
	});

	// Form State for History/Transaction
	let historyForm = $state({
		transaction_type: 'BUY',
		quantity_change: 0,
		price_per_unit: 0,
		transaction_date: new Date().toISOString().split('T')[0],
		notes: ''
	});

	const assetTypes = [
		{ value: 'STOCK', label: 'Saham' },
		{ value: 'CRYPTO', label: 'Kripto' },
		{ value: 'MUTUAL_FUND', label: 'Reksa Dana' },
		{ value: 'GOLD', label: 'Emas' },
		{ value: 'OBLIGATION', label: 'Obligasi' },
		{ value: 'OTHER', label: 'Lainnya' }
	];

	// Mock Data
	const mockAssets = [
		{ id: '1', name: 'BBCA', asset_type: 'STOCK', symbol: 'BBCA.JK', quantity: 1000, avg_buy_price: 9000, currency: 'IDR' },
		{ id: '2', name: 'Bitcoin', asset_type: 'CRYPTO', symbol: 'BTC', quantity: 0.05, avg_buy_price: 900000000, currency: 'IDR' },
		{ id: '3', name: 'Suasane Gold', asset_type: 'GOLD', symbol: 'GOLD', quantity: 10, avg_buy_price: 1100000, currency: 'IDR' }
	];

	async function fetchAssets() {
		loading = true;
		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';
		
		if (isMock) {
			setTimeout(() => {
				assets = mockAssets;
				loading = false;
			}, 800);
			return;
		}

		try {
			const res = await apiFetch('/investments');
			const result = await res.json();
			if (result.success) assets = result.data || [];
		} catch (err) {
			console.error(err);
		} finally {
			loading = false;
		}
	}

	async function fetchHistory(assetId: string) {
		historyLoading = true;
		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';

		if (isMock) {
			setTimeout(() => {
				history = [
					{ id: 'h1', transaction_date: '2026-05-01', transaction_type: 'BUY', quantity_change: 50, price_per_unit: 8500, notes: 'Initial buy' },
					{ id: 'h2', transaction_date: '2026-05-15', transaction_type: 'BUY', quantity_change: 50, price_per_unit: 9500, notes: 'Accumulate' }
				];
				historyLoading = false;
			}, 600);
			return;
		}

		try {
			const res = await apiFetch(`/investments/${assetId}/history`);
			const result = await res.json();
			if (result.success) history = result.data || [];
		} catch (err) {
			console.error(err);
		} finally {
			historyLoading = false;
		}
	}

	async function handleSaveAsset(e: SubmitEvent) {
		e.preventDefault();
		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';

		if (isMock) {
			if (assetForm.id) {
				assets = assets.map(a => a.id === assetForm.id ? { ...a, ...assetForm } : a);
			} else {
				assets = [...assets, { ...assetForm, id: Math.random().toString() }];
			}
			showAssetModal = false;
			return;
		}

		try {
			const method = assetForm.id ? 'PUT' : 'POST';
			const url = assetForm.id ? `/investments/${assetForm.id}` : '/investments';
			const res = await apiFetch(url, {
				method,
				body: JSON.stringify(assetForm)
			});
			const result = await res.json();
			if (result.success) {
				showAssetModal = false;
				fetchAssets();
			}
		} catch (err) {
			console.error(err);
		}
	}

	async function handleDeleteAsset(id: string) {
		if (!confirm('Hapus instrumen ini? Riwayat transaksi juga akan terhapus.')) return;
		
		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';
		if (isMock) {
			assets = assets.filter(a => a.id !== id);
			showAssetModal = false;
			return;
		}

		try {
			const res = await apiFetch(`/investments/${id}`, { method: 'DELETE' });
			const result = await res.json();
			if (result.success) {
				showAssetModal = false;
				fetchAssets();
			}
		} catch (err) {
			console.error(err);
		}
	}

	async function handleRecordHistory(e: SubmitEvent) {
		e.preventDefault();
		if (!selectedAsset) return;

		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';
		if (isMock) {
			const newEntry = { 
				id: Math.random().toString(), 
				...historyForm 
			};
			history = [newEntry, ...history];
			// Update asset quantity in list
			const change = historyForm.transaction_type === 'SELL' ? -historyForm.quantity_change : historyForm.quantity_change;
			assets = assets.map(a => a.id === selectedAsset.id ? { ...a, quantity: a.quantity + change } : a);
			selectedAsset.quantity += change;
			return;
		}

		try {
			const res = await apiFetch(`/investments/${selectedAsset.id}/history`, {
				method: 'POST',
				body: JSON.stringify(historyForm)
			});
			const result = await res.json();
			if (result.success) {
				fetchHistory(selectedAsset.id);
				fetchAssets(); // update quantity in main list
			}
		} catch (err) {
			console.error(err);
		}
	}

	function openAddModal() {
		assetForm = { id: '', name: '', asset_type: 'STOCK', symbol: '', quantity: 0, avg_buy_price: 0, currency: 'IDR' };
		showAssetModal = true;
	}

	function openEditModal(asset: any) {
		assetForm = { ...asset };
		showAssetModal = true;
	}

	function openHistoryModal(asset: any) {
		selectedAsset = asset;
		history = [];
		fetchHistory(asset.id);
		showHistoryModal = true;
	}

	onMount(fetchAssets);

	function formatCurrency(val: number, currency = 'IDR') {
		return new Intl.NumberFormat('id-ID', { 
			style: 'currency', 
			currency, 
			minimumFractionDigits: currency === 'IDR' ? 0 : 2 
		}).format(val);
	}

	function formatNumber(val: number) {
		return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 8 }).format(val);
	}

	const totalValue = $derived(assets.reduce((acc, a) => acc + (a.quantity * a.avg_buy_price), 0));
</script>

<div class="space-y-8 animate-in fade-in duration-500 pb-20">
	<!-- Header & Summary -->
	<div class="flex flex-col md:flex-row md:items-end justify-between gap-6">
		<div class="space-y-1">
			<h1 class="text-3xl font-black text-[#0a2e31]">Portofolio Investasi</h1>
			<p class="text-gray-500 font-medium text-sm">Kelola instrumen aset dan pantau pertumbuhan unit Anda.</p>
		</div>
		
		<div class="bg-[#0a2e31] px-8 py-5 rounded-[2rem] text-white shadow-xl shadow-teal-950/20 flex flex-col items-end">
			<span class="text-[10px] font-black uppercase tracking-[0.2em] text-teal-400/80 mb-1">Total Estimasi Aset</span>
			<span class="text-2xl font-black tabular-nums">{formatCurrency(totalValue)}</span>
		</div>
	</div>

	<!-- Action Bar -->
	<div class="flex justify-end">
		<button 
			onclick={openAddModal}
			class="inline-flex items-center gap-2 px-6 py-3.5 bg-[#217b84] hover:bg-[#1a5f66] text-white font-black text-xs uppercase tracking-widest rounded-2xl shadow-lg transition-all active:scale-95"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3"><path d="M12 4v16m8-8H4" /></svg>
			Tambah Instrumen
		</button>
	</div>

	<!-- Assets List -->
	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
		{#if loading}
			{#each Array(3) as _}
				<div class="h-48 bg-white rounded-[2.5rem] border border-gray-100 animate-pulse"></div>
			{/each}
		{:else if assets.length === 0}
			<div class="col-span-full py-20 bg-white rounded-[2.5rem] border border-dashed border-gray-200 text-center space-y-4">
				<div class="h-16 w-16 bg-gray-50 rounded-full mx-auto flex items-center justify-center text-gray-300">
					<svg class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" /></svg>
				</div>
				<div class="space-y-1">
					<p class="font-bold text-[#0a2e31]">Belum ada instrumen investasi</p>
					<p class="text-xs text-gray-400">Klik tombol di atas untuk mulai mencatat aset Anda.</p>
				</div>
			</div>
		{:else}
			{#each assets as asset (asset.id)}
				<div class="bg-white p-8 rounded-[2.5rem] border border-gray-100 shadow-sm hover:shadow-xl hover:-translate-y-1 transition-all group relative overflow-hidden">
					<div class="absolute -right-4 -top-4 h-24 w-24 bg-teal-50/50 rounded-full group-hover:scale-125 transition-transform duration-700"></div>
					
					<div class="relative z-10 flex flex-col h-full">
						<div class="flex justify-between items-start mb-6">
							<div>
								<span class="text-[10px] font-black uppercase tracking-widest text-teal-600 bg-teal-50 px-3 py-1 rounded-full mb-2 inline-block">
									{asset.asset_type}
								</span>
								<h3 class="text-xl font-black text-[#0a2e31]">{asset.name}</h3>
								<p class="text-xs font-bold text-gray-400">{asset.symbol}</p>
							</div>
							<button
								aria-label="Edit {asset.name}"
								onclick={() => openEditModal(asset)}
								class="p-2 text-gray-300 hover:text-[#0a2e31] transition-colors"
							>
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg>
							</button>
						</div>

						<div class="mt-auto space-y-4">
							<div class="flex justify-between items-end">
								<div class="space-y-0.5">
									<p class="text-[10px] font-bold text-gray-400 uppercase tracking-widest">Kepemilikan</p>
									<p class="text-lg font-black text-[#0a2e31]">{formatNumber(asset.quantity)} <span class="text-xs text-gray-400 font-bold ml-1">Unit</span></p>
								</div>
								<div class="text-right space-y-0.5">
									<p class="text-[10px] font-bold text-gray-400 uppercase tracking-widest">Avg Price</p>
									<p class="font-bold text-[#0a2e31] text-sm">{formatCurrency(asset.avg_buy_price, asset.currency)}</p>
								</div>
							</div>

							{#if asset.current_price}
								<div class="flex justify-between items-center p-3 bg-teal-50/50 rounded-xl border border-teal-100/50">
									<span class="text-[10px] font-black text-teal-700 uppercase tracking-widest">Harga Terkini</span>
									<span class="text-sm font-black text-teal-800">{formatCurrency(asset.current_price, asset.currency)}</span>
								</div>
							{/if}

							<div class="flex gap-2">
								<button 
									onclick={() => handleFetchPrice(asset)}
									disabled={priceLoading[asset.id]}
									class="p-3 bg-gray-50 hover:bg-gray-100 text-[#0a2e31] rounded-xl transition-all disabled:opacity-50"
									title="Perbarui Harga"
								>
									<svg class="h-4 w-4 {priceLoading[asset.id] ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
										<path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
									</svg>
								</button>
								<button 
									onclick={() => openHistoryModal(asset)}
									class="flex-1 py-3 bg-gray-50 hover:bg-teal-50 text-[#0a2e31] hover:text-teal-700 text-[11px] font-black uppercase tracking-[0.1em] rounded-xl transition-all"
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
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-[#0a2e31]/40 backdrop-blur-sm" in:fade={{ duration: 200 }}>
		<div class="bg-white w-full max-w-lg rounded-[2.5rem] shadow-2xl overflow-hidden" in:fly={{ y: 20, duration: 400 }}>
			<div class="p-10 space-y-8">
				<div class="flex justify-between items-center">
					<h2 class="text-2xl font-black text-[#0a2e31]">{assetForm.id ? 'Edit Instrumen' : 'Tambah Instrumen'}</h2>
					<button aria-label="Tutup modal" onclick={() => showAssetModal = false} class="text-gray-300 hover:text-gray-500 transition-colors">
						<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path d="M6 18L18 6M6 6l12 12" /></svg>
					</button>
				</div>

				<form onsubmit={handleSaveAsset} class="space-y-6">
					<div class="grid grid-cols-1 md:grid-cols-2 gap-5">
						<div class="md:col-span-2 space-y-2">
							<label for="name" class="block text-[11px] font-black text-gray-400 uppercase tracking-widest px-1">Nama Aset</label>
							<input id="name" type="text" required bind:value={assetForm.name} placeholder="Contoh: Saham BBCA" class="w-full px-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 outline-none transition-all font-bold text-[#0a2e31]" />
						</div>

						<div class="space-y-2">
							<label for="type" class="block text-[11px] font-black text-gray-400 uppercase tracking-widest px-1">Tipe Aset</label>
							<select id="type" bind:value={assetForm.asset_type} class="w-full px-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 outline-none transition-all font-bold text-[#0a2e31]">
								{#each assetTypes as type}
									<option value={type.value}>{type.label}</option>
								{/each}
							</select>
						</div>

						<div class="space-y-2">
							<label for="symbol" class="block text-[11px] font-black text-gray-400 uppercase tracking-widest px-1">Ticker / Simbol</label>
							<input id="symbol" type="text" required bind:value={assetForm.symbol} placeholder="Contoh: BBCA.JK" class="w-full px-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 outline-none transition-all font-bold text-[#0a2e31]" />
						</div>

						{#if !assetForm.id}
							<div class="space-y-2">
								<label for="qty" class="block text-[11px] font-black text-gray-400 uppercase tracking-widest px-1">Kuantitas Awal</label>
								<input id="qty" type="number" step="any" bind:value={assetForm.quantity} class="w-full px-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 outline-none transition-all font-bold text-[#0a2e31]" />
							</div>

							<div class="space-y-2">
								<label for="price" class="block text-[11px] font-black text-gray-400 uppercase tracking-widest px-1">Harga Beli Rata-Rata</label>
								<input id="price" type="number" step="any" bind:value={assetForm.avg_buy_price} class="w-full px-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 outline-none transition-all font-bold text-[#0a2e31]" />
							</div>
						{/if}

						<div class="space-y-2">
							<label for="currency" class="block text-[11px] font-black text-gray-400 uppercase tracking-widest px-1">Mata Uang</label>
							<input id="currency" type="text" bind:value={assetForm.currency} class="w-full px-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 outline-none transition-all font-bold text-[#0a2e31]" />
						</div>
					</div>

					<div class="flex gap-4 pt-4">
						{#if assetForm.id}
							<button 
								type="button"
								onclick={() => handleDeleteAsset(assetForm.id)}
								class="px-6 py-4 border-2 border-red-50 text-red-500 hover:bg-red-50 font-black text-xs uppercase tracking-widest rounded-2xl transition-all"
							>
								Hapus
							</button>
						{/if}
						<button 
							type="submit"
							class="flex-1 py-4 bg-[#0a2e31] hover:bg-black text-white font-black text-xs uppercase tracking-widest rounded-2xl shadow-xl transition-all active:scale-[0.98]"
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
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-[#0a2e31]/40 backdrop-blur-sm" in:fade={{ duration: 200 }}>
		<div class="bg-gray-50 w-full max-w-4xl max-h-[90vh] rounded-[3rem] shadow-2xl overflow-hidden flex flex-col" in:fly={{ y: 20, duration: 400 }}>
			<!-- Modal Header -->
			<div class="bg-white p-8 border-b border-gray-100 flex justify-between items-center">
				<div>
					<h2 class="text-2xl font-black text-[#0a2e31]">{selectedAsset.name}</h2>
					<p class="text-xs font-bold text-gray-400">{selectedAsset.symbol} — {selectedAsset.asset_type}</p>
				</div>
				<button aria-label="Tutup riwayat" onclick={() => showHistoryModal = false} class="p-2 text-gray-300 hover:text-gray-500">
					<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path d="M6 18L18 6M6 6l12 12" /></svg>
				</button>
			</div>

			<div class="flex-1 overflow-y-auto p-8 grid grid-cols-1 lg:grid-cols-5 gap-8">
				<!-- History List -->
				<div class="lg:col-span-3 space-y-6">
					<h3 class="text-sm font-black text-[#0a2e31] uppercase tracking-widest">Riwayat Perubahan Unit</h3>
					
					{#if historyLoading}
						<div class="space-y-4">
							{#each Array(3) as _}
								<div class="h-20 bg-white rounded-2xl animate-pulse"></div>
							{/each}
						</div>
					{:else if history.length === 0}
						<div class="bg-white p-12 rounded-3xl text-center border border-gray-100">
							<p class="text-gray-400 font-bold text-sm">Belum ada riwayat transaksi.</p>
						</div>
					{:else}
						<div class="space-y-3">
							{#each history as entry}
								<div class="bg-white p-5 rounded-2xl border border-gray-100 flex justify-between items-center group hover:border-teal-200 transition-colors">
									<div class="flex items-center gap-4">
										<div class="h-10 w-10 rounded-xl flex items-center justify-center font-black text-[10px] {entry.transaction_type === 'BUY' ? 'bg-green-50 text-green-600' : 'bg-red-50 text-red-500'}">
											{entry.transaction_type}
										</div>
										<div>
											<p class="text-sm font-black text-[#0a2e31]">{formatNumber(entry.quantity_change)} Unit</p>
											<p class="text-[10px] font-bold text-gray-400">{entry.transaction_date}</p>
										</div>
									</div>
									<div class="text-right">
										<p class="text-sm font-bold text-[#0a2e31]">{formatCurrency(entry.price_per_unit, selectedAsset.currency)}</p>
										<p class="text-[10px] font-medium text-gray-400 italic truncate max-w-[100px]">{entry.notes || '-'}</p>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>

				<!-- Record Form -->
				<div class="lg:col-span-2">
					<div class="bg-white p-8 rounded-3xl border border-gray-100 shadow-sm sticky top-0 space-y-6">
						<h3 class="text-sm font-black text-[#0a2e31] uppercase tracking-widest">Catat Transaksi Baru</h3>
						
						<form onsubmit={handleRecordHistory} class="space-y-4">
							<div class="flex p-1 bg-gray-50 rounded-xl">
								<button 
									type="button"
									onclick={() => historyForm.transaction_type = 'BUY'}
									class="flex-1 py-2 text-[10px] font-black uppercase tracking-widest rounded-lg transition-all {historyForm.transaction_type === 'BUY' ? 'bg-white shadow-sm text-green-600' : 'text-gray-400'}"
								>
									Beli
								</button>
								<button 
									type="button"
									onclick={() => historyForm.transaction_type = 'SELL'}
									class="flex-1 py-2 text-[10px] font-black uppercase tracking-widest rounded-lg transition-all {historyForm.transaction_type === 'SELL' ? 'bg-white shadow-sm text-red-500' : 'text-gray-400'}"
								>
									Jual
								</button>
							</div>

							<div class="space-y-1.5">
								<label for="h-qty" class="text-[10px] font-black text-gray-400 uppercase tracking-widest px-1">Perubahan Kuantitas</label>
								<input id="h-qty" type="number" step="any" required bind:value={historyForm.quantity_change} class="w-full px-4 py-2.5 bg-gray-50 border border-gray-100 rounded-xl focus:ring-2 focus:ring-teal-500 outline-none transition-all font-bold text-sm" />
							</div>

							<div class="space-y-1.5">
								<label for="h-price" class="text-[10px] font-black text-gray-400 uppercase tracking-widest px-1">Harga per Unit</label>
								<input id="h-price" type="number" step="any" required bind:value={historyForm.price_per_unit} class="w-full px-4 py-2.5 bg-gray-50 border border-gray-100 rounded-xl focus:ring-2 focus:ring-teal-500 outline-none transition-all font-bold text-sm" />
							</div>

							<div class="space-y-1.5">
								<label for="h-date" class="text-[10px] font-black text-gray-400 uppercase tracking-widest px-1">Tanggal</label>
								<input id="h-date" type="date" required bind:value={historyForm.transaction_date} class="w-full px-4 py-2.5 bg-gray-50 border border-gray-100 rounded-xl focus:ring-2 focus:ring-teal-500 outline-none transition-all font-bold text-sm" />
							</div>

							<div class="space-y-1.5">
								<label for="h-notes" class="text-[10px] font-black text-gray-400 uppercase tracking-widest px-1">Catatan</label>
								<textarea id="h-notes" bind:value={historyForm.notes} rows="2" class="w-full px-4 py-2.5 bg-gray-50 border border-gray-100 rounded-xl focus:ring-2 focus:ring-teal-500 outline-none transition-all font-bold text-sm"></textarea>
							</div>

							<button 
								type="submit"
								class="w-full py-3.5 bg-[#0a2e31] hover:bg-black text-white font-black text-[10px] uppercase tracking-widest rounded-xl transition-all active:scale-[0.98] shadow-lg"
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
	/* Custom scrollbar for better look */
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
