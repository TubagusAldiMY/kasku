<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { fade, fly } from 'svelte/transition';

	let transactions = $state<any[]>([]);
	let loading = $state(true);
	let showAddModal = $state(false);

	// Data pendukung untuk form
	let myAccounts = $state<any[]>([]);
	const categories = ['Makanan', 'Transportasi', 'Belanja', 'Gaji', 'Tagihan', 'Hiburan', 'Kesehatan', 'Pendidikan', 'Lainnya'];

	// State untuk form tambah transaksi
	let newTx = $state({
		title: '',
		amount: 0,
		type: 'EXPENSE',
		category: 'Makanan',
		account_id: '',
		date: new Date().toISOString().split('T')[0]
	});

	// Mock Data
	const mockTransactions = [
		{ id: '1', date: '2026-05-10', title: 'Kopi Kenangan', category: 'Makanan', account: 'BCA Utama', amount: -25000, type: 'EXPENSE' },
		{ id: '2', date: '2026-05-10', title: 'Gaji Mei', category: 'Gaji', account: 'BCA Utama', amount: 15000000, type: 'INCOME' },
		{ id: '3', date: '2026-05-09', title: 'Listrik Token', category: 'Tagihan', account: 'Mandiri', amount: -500000, type: 'EXPENSE' }
	];

	async function fetchData() {
		loading = true;
		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';
		
		if (isMock) {
			setTimeout(() => {
				transactions = mockTransactions;
				myAccounts = [
					{ id: 'acc-1', name: 'BCA Utama' },
					{ id: 'acc-2', name: 'Mandiri' },
					{ id: 'acc-3', name: 'Dompet' }
				];
				if (myAccounts.length > 0) newTx.account_id = myAccounts[0].id;
				loading = false;
			}, 800);
			return;
		}

		try {
			const [txRes, accRes] = await Promise.all([
				apiFetch('/transactions'),
				apiFetch('/accounts')
			]);
			const txData = await txRes.json();
			const accData = await accRes.json();
			
			if (txData.success) transactions = txData.data || [];
			if (accData.success) {
				myAccounts = accData.data || [];
				if (myAccounts.length > 0) newTx.account_id = myAccounts[0].id;
			}
		} catch (err) {
			console.error(err);
		} finally {
			loading = false;
		}
	}

	async function handleAddTransaction(e: SubmitEvent) {
		e.preventDefault();
		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';

		if (isMock) {
			const selectedAccount = myAccounts.find(a => a.id === newTx.account_id);
			const mockNew = {
				id: Math.random().toString(),
				...newTx,
				account: selectedAccount?.name || 'Unknown',
				amount: newTx.type === 'EXPENSE' ? -Math.abs(newTx.amount) : Math.abs(newTx.amount)
			};
			transactions = [mockNew, ...transactions];
			showAddModal = false;
			return;
		}

		try {
			const finalAmount = newTx.type === 'EXPENSE' ? -Math.abs(newTx.amount) : Math.abs(newTx.amount);
			const res = await apiFetch('/transactions', {
				method: 'POST',
				body: JSON.stringify({ ...newTx, amount: finalAmount })
			});
			const result = await res.json();
			if (result.success) {
				showAddModal = false;
				fetchData();
			}
		} catch (err) {
			console.error(err);
		}
	}

	onMount(fetchData);

	function formatCurrency(val: number) {
		const absVal = Math.abs(val);
		const formatted = new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(absVal);
		return val < 0 ? `- ${formatted}` : `+ ${formatted}`;
	}
</script>

<div class="space-y-8 animate-in fade-in duration-500">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
		<div>
			<h1 class="text-2xl font-bold text-[#0a2e31]">Riwayat Transaksi</h1>
			<p class="text-sm text-gray-500">Pantau arus kas masuk dan keluar Anda secara detail.</p>
		</div>
		<button 
			onclick={() => showAddModal = true}
			class="inline-flex items-center justify-center gap-2 px-6 py-3 bg-[#217b84] hover:bg-[#1a5f66] text-white font-bold rounded-2xl shadow-lg transition-all active:scale-95"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4" /></svg>
			Catat Transaksi
		</button>
	</div>

	<!-- List Transaksi -->
	<div class="bg-white rounded-[2.5rem] border border-gray-100 shadow-sm overflow-hidden text-sm">
		{#if loading}
			<div class="p-12 space-y-6">
				{#each Array(4) as _}
					<div class="h-12 bg-gray-50 rounded-xl animate-pulse"></div>
				{/each}
			</div>
		{:else if transactions.length === 0}
			<div class="p-20 text-center space-y-4">
				<div class="h-20 w-20 bg-gray-50 rounded-full mx-auto flex items-center justify-center text-gray-300">
					<svg class="h-10 w-10" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" /></svg>
				</div>
				<p class="font-bold text-[#0a2e31]">Belum ada transaksi</p>
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-left border-collapse">
					<thead>
						<tr class="bg-gray-50/50">
							<th class="px-8 py-5 font-bold text-[#0a2e31] uppercase text-[10px] tracking-widest">Tanggal</th>
							<th class="px-8 py-5 font-bold text-[#0a2e31] uppercase text-[10px] tracking-widest">Keterangan</th>
							<th class="px-8 py-5 font-bold text-[#0a2e31] uppercase text-[10px] tracking-widest">Kategori</th>
							<th class="px-8 py-5 font-bold text-[#0a2e31] uppercase text-[10px] tracking-widest">Akun</th>
							<th class="px-8 py-5 font-bold text-[#0a2e31] uppercase text-[10px] tracking-widest text-right">Nominal</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-gray-50">
						{#each transactions as tx}
							<tr class="hover:bg-gray-50/80 transition-colors group">
								<td class="px-8 py-5 text-gray-500 whitespace-nowrap">{tx.date}</td>
								<td class="px-8 py-5 font-bold text-[#0a2e31]">{tx.title}</td>
								<td class="px-8 py-5">
									<span class="px-3 py-1 bg-teal-50 text-teal-700 rounded-full text-[11px] font-bold uppercase tracking-tight">{tx.category}</span>
								</td>
								<td class="px-8 py-5 text-gray-500 font-medium">{tx.account}</td>
								<td class="px-8 py-5 text-right font-black tracking-tight {tx.type === 'INCOME' ? 'text-green-600' : 'text-red-500'}">
									{formatCurrency(tx.amount)}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</div>

<!-- Modal Tambah Transaksi -->
{#if showAddModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-[#0a2e31]/40 backdrop-blur-sm" in:fade={{ duration: 200 }}>
		<div 
			class="bg-white w-full max-w-lg rounded-[2.5rem] shadow-2xl overflow-hidden"
			in:fly={{ y: 20, duration: 400 }}
		>
			<div class="p-8 space-y-6">
				<div class="flex justify-between items-center">
					<h2 class="text-2xl font-bold text-[#0a2e31]">Catat Transaksi</h2>
					<button onclick={() => showAddModal = false} class="text-gray-400 hover:text-gray-600" aria-label="Tutup modal tambah transaksi">
						<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
					</button>
				</div>

				<form onsubmit={handleAddTransaction} class="space-y-6">
					<!-- Toggle Tipe Transaksi -->
					<div class="flex p-1.5 bg-gray-50 rounded-2xl border border-gray-100">
						<button 
							type="button"
							onclick={() => { newTx.type = 'EXPENSE'; newTx.category = 'Makanan'; }}
							class="flex-1 py-3 text-sm font-bold rounded-xl transition-all {newTx.type === 'EXPENSE' ? 'bg-white shadow-sm text-red-500' : 'text-gray-400'}"
						>
							Pengeluaran
						</button>
						<button 
							type="button"
							onclick={() => { newTx.type = 'INCOME'; newTx.category = 'Gaji'; }}
							class="flex-1 py-3 text-sm font-bold rounded-xl transition-all {newTx.type === 'INCOME' ? 'bg-white shadow-sm text-green-600' : 'text-gray-400'}"
						>
							Pemasukan
						</button>
					</div>

					<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
						<div class="col-span-2">
							<label for="title" class="block text-xs font-bold text-[#0a2e31] uppercase tracking-wider mb-2 px-1">Keterangan</label>
							<input id="title" type="text" required bind:value={newTx.title} placeholder="Beli apa hari ini?" class="w-full px-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 focus:border-[#217b84] outline-none transition-all" />
						</div>

						<div class="col-span-2">
							<label for="amount" class="block text-xs font-bold text-[#0a2e31] uppercase tracking-wider mb-2 px-1">Nominal</label>
							<div class="relative">
								<span class="absolute inset-y-0 left-5 flex items-center text-gray-400 font-bold text-sm">Rp</span>
								<input id="amount" type="number" required bind:value={newTx.amount} class="w-full pl-12 pr-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 focus:border-[#217b84] outline-none transition-all" />
							</div>
						</div>

						<div>
							<label for="account" class="block text-xs font-bold text-[#0a2e31] uppercase tracking-wider mb-2 px-1">Pilih Rekening</label>
							<select id="account" bind:value={newTx.account_id} class="w-full px-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 focus:border-[#217b84] outline-none transition-all appearance-none cursor-pointer">
								{#each myAccounts as acc}
									<option value={acc.id}>{acc.name}</option>
								{/each}
							</select>
						</div>

						<div>
							<label for="category" class="block text-xs font-bold text-[#0a2e31] uppercase tracking-wider mb-2 px-1">Kategori</label>
							<select id="category" bind:value={newTx.category} class="w-full px-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 focus:border-[#217b84] outline-none transition-all appearance-none cursor-pointer">
								{#each categories as cat}
									<option value={cat}>{cat}</option>
								{/each}
							</select>
						</div>

						<div class="col-span-2">
							<label for="date" class="block text-xs font-bold text-[#0a2e31] uppercase tracking-wider mb-2 px-1">Tanggal</label>
							<input id="date" type="date" required bind:value={newTx.date} class="w-full px-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 focus:border-[#217b84] outline-none transition-all" />
						</div>
					</div>

					<button 
						type="submit"
						class="w-full py-4 bg-[#0a2e31] hover:bg-black text-white font-bold rounded-2xl shadow-xl transition-all active:scale-[0.98] mt-4"
					>
						Simpan Transaksi
					</button>
				</form>
			</div>
		</div>
	</div>
{/if}
