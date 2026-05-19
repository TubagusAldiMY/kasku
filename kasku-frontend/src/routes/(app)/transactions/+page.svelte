<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { fade, fly } from 'svelte/transition';
	import {
		transactionsRepo,
		accountsRepo,
		categoriesRepo,
		type TransactionRow,
		type AccountRow,
		type CategoryRow
	} from '$lib/db';
	import { enqueueCreate, enqueueDelete, syncStatus, triggerManualSync } from '$lib/sync';

	type Transaction = {
		id: string;
		date: string;
		title: string;
		category: string;
		account: string;
		amount: number;
		type: 'INCOME' | 'EXPENSE';
	};
	type AccountRef = { id: string; name: string };
	type CategoryRef = { id: string; name: string; category_type: 'INCOME' | 'EXPENSE' | 'BOTH' };

	let transactions = $state<Transaction[]>([]);
	let loading = $state(true);
	let showAddModal = $state(false);

	let myAccounts = $state<AccountRef[]>([]);
	let allCategories = $state<CategoryRef[]>([]);

	let newTx = $state({
		title: '',
		amount: 0,
		type: 'EXPENSE' as 'INCOME' | 'EXPENSE',
		category_id: '',
		account_id: '',
		date: new Date().toISOString().split('T')[0]
	});

	const filteredCategories = $derived(
		allCategories.filter((c) => c.category_type === newTx.type || c.category_type === 'BOTH')
	);

	function projectTransaction(
		row: TransactionRow,
		accounts: AccountRow[],
		categories: CategoryRow[]
	): Transaction {
		const acc = accounts.find((a) => a.id === row.account_id);
		const cat = categories.find((c) => c.id === row.category_id);
		const signed = row.transaction_type === 'EXPENSE' ? -row.amount_idr : row.amount_idr;
		return {
			id: row.id,
			date: row.transaction_date,
			title: row.notes ?? row.transaction_type,
			category: cat?.name ?? 'Umum',
			account: acc?.name ?? '—',
			amount: signed,
			type: row.transaction_type
		};
	}

	async function reloadFromLocal() {
		try {
			const [txRows, accRows, catRows] = await Promise.all([
				transactionsRepo.getAll(),
				accountsRepo.getAll(),
				categoriesRepo.getAll()
			]);
			myAccounts = accRows.map((a) => ({ id: a.id, name: a.name }));
			allCategories = catRows.map((c) => ({
				id: c.id,
				name: c.name,
				category_type: c.category_type
			}));
			transactions = txRows
				.map((t) => projectTransaction(t, accRows, catRows))
				.sort((a, b) => (a.date < b.date ? 1 : -1));
			if (!newTx.account_id && myAccounts.length > 0) newTx.account_id = myAccounts[0].id;
			if (!newTx.category_id && filteredCategories.length > 0)
				newTx.category_id = filteredCategories[0].id;
		} catch (err) {
			console.error('Gagal membaca transaksi dari penyimpanan lokal:', err);
		}
	}

	async function hydrateCategoriesFromServer() {
		// Categories belum ikut sync engine — fetch on-demand & cache ke IDB.
		// TODO(sync): integrasikan categories ke SyncableResource saat siap.
		try {
			const res = await apiFetch('/categories');
			const result = await res.json();
			if (result.success && Array.isArray(result.data)) {
				const rows = result.data as CategoryRow[];
				await categoriesRepo.clear();
				await categoriesRepo.putMany(rows);
				await reloadFromLocal();
			}
		} catch {
			// Offline → tetap pakai cache IDB.
		}
	}

	$effect(() => {
		void syncStatus.dataVersion;
		void reloadFromLocal();
	});

	async function handleAddTransaction(e: SubmitEvent) {
		e.preventDefault();
		try {
			await enqueueCreate<TransactionRow>('transactions', {
				account_id: newTx.account_id,
				category_id: newTx.category_id,
				transaction_type: newTx.type,
				amount_idr: Math.abs(newTx.amount),
				transaction_date: newTx.date,
				notes: newTx.title
			});
			showAddModal = false;
		} catch (err) {
			console.error('Gagal menambah transaksi:', err);
		}
	}

	async function handleDeleteTransaction(id: string) {
		if (!confirm('Hapus transaksi ini? Saldo rekening Anda akan disesuaikan kembali.')) return;
		try {
			await enqueueDelete('transactions', id);
		} catch (err) {
			console.error('Gagal menghapus transaksi:', err);
		}
	}

	onMount(async () => {
		await reloadFromLocal();
		loading = false;
		void hydrateCategoriesFromServer();
		void triggerManualSync();
	});

	function formatCurrency(val: number) {
		const absVal = Math.abs(val);
		const formatted = new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(absVal);
		return val < 0 ? `- ${formatted}` : `+ ${formatted}`;
	}
</script>

<div class="animate-in fade-in space-y-8 duration-500">
	<!-- Header -->
	<div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
		<div>
			<h1 class="text-2xl font-bold text-[#0a2e31]">Riwayat Transaksi</h1>
			<p class="text-sm text-gray-500">Pantau arus kas masuk dan keluar Anda secara detail.</p>
		</div>
		<button
			onclick={() => (showAddModal = true)}
			class="inline-flex items-center justify-center gap-2 rounded-2xl bg-[#217b84] px-6 py-3 font-bold text-white shadow-lg transition-all hover:bg-[#1a5f66] active:scale-95"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"
				><path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2.5"
					d="M12 4v16m8-8H4"
				/></svg
			>
			Catat Transaksi
		</button>
	</div>

	<!-- List Transaksi -->
	<div class="overflow-hidden rounded-[2.5rem] border border-gray-100 bg-white text-sm shadow-sm">
		{#if loading}
			<div class="space-y-6 p-12">
				{#each [0, 1, 2, 3] as i (i)}
					<div class="h-12 animate-pulse rounded-xl bg-gray-50"></div>
				{/each}
			</div>
		{:else if transactions.length === 0}
			<div class="space-y-4 p-20 text-center">
				<div
					class="mx-auto flex h-20 w-20 items-center justify-center rounded-full bg-gray-50 text-gray-300"
				>
					<svg class="h-10 w-10" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
						/></svg
					>
				</div>
				<p class="font-bold text-[#0a2e31]">Belum ada transaksi</p>
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full border-collapse text-left">
					<thead>
						<tr class="bg-gray-50/50">
							<th class="px-8 py-5 text-[10px] font-bold tracking-widest text-[#0a2e31] uppercase"
								>Tanggal</th
							>
							<th class="px-8 py-5 text-[10px] font-bold tracking-widest text-[#0a2e31] uppercase"
								>Keterangan</th
							>
							<th class="px-8 py-5 text-[10px] font-bold tracking-widest text-[#0a2e31] uppercase"
								>Kategori</th
							>
							<th class="px-8 py-5 text-[10px] font-bold tracking-widest text-[#0a2e31] uppercase"
								>Akun</th
							>
							<th
								class="px-8 py-5 text-right text-[10px] font-bold tracking-widest text-[#0a2e31] uppercase"
								>Nominal</th
							>
							<th class="px-8 py-5"></th>
						</tr>
					</thead>
					<tbody class="divide-y divide-gray-50">
						{#each transactions as tx (tx.id)}
							<tr class="group transition-colors hover:bg-gray-50/80">
								<td class="px-8 py-5 whitespace-nowrap text-gray-500">{tx.date}</td>
								<td class="px-8 py-5 font-bold text-[#0a2e31]">{tx.title}</td>
								<td class="px-8 py-5">
									<span
										class="rounded-full bg-teal-50 px-3 py-1 text-[11px] font-bold tracking-tight text-teal-700 uppercase"
										>{tx.category}</span
									>
								</td>
								<td class="px-8 py-5 font-medium text-gray-500">{tx.account}</td>
								<td
									class="px-8 py-5 text-right font-black tracking-tight {tx.type === 'INCOME'
										? 'text-green-600'
										: 'text-red-500'}"
								>
									{formatCurrency(tx.amount)}
								</td>
								<td class="px-8 py-5 text-right">
									<button
										onclick={() => handleDeleteTransaction(tx.id)}
										class="p-2 text-gray-300 opacity-0 transition-colors group-hover:opacity-100 hover:text-red-500"
										aria-label="Hapus transaksi"
									>
										<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"
											><path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
											/></svg
										>
									</button>
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
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-[#0a2e31]/40 p-4 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
	>
		<div
			class="w-full max-w-lg overflow-hidden rounded-[2.5rem] bg-white shadow-2xl"
			in:fly={{ y: 20, duration: 400 }}
		>
			<div class="space-y-6 p-8">
				<div class="flex items-center justify-between">
					<h2 class="text-2xl font-bold text-[#0a2e31]">Catat Transaksi</h2>
					<button
						onclick={() => (showAddModal = false)}
						class="text-gray-400 hover:text-gray-600"
						aria-label="Tutup modal tambah transaksi"
					>
						<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"
							><path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M6 18L18 6M6 6l12 12"
							/></svg
						>
					</button>
				</div>

				<form onsubmit={handleAddTransaction} class="space-y-6">
					<!-- Toggle Tipe Transaksi -->
					<div class="flex rounded-2xl border border-gray-100 bg-gray-50 p-1.5">
						<button
							type="button"
							onclick={() => {
								newTx.type = 'EXPENSE';
								const first = allCategories.find((c) => c.category_type === 'EXPENSE');
								newTx.category_id = first ? first.id : '';
							}}
							class="flex-1 rounded-xl py-3 text-sm font-bold transition-all {newTx.type ===
							'EXPENSE'
								? 'bg-white text-red-500 shadow-sm'
								: 'text-gray-400'}"
						>
							Pengeluaran
						</button>
						<button
							type="button"
							onclick={() => {
								newTx.type = 'INCOME';
								const first = allCategories.find((c) => c.category_type === 'INCOME');
								newTx.category_id = first ? first.id : '';
							}}
							class="flex-1 rounded-xl py-3 text-sm font-bold transition-all {newTx.type ===
							'INCOME'
								? 'bg-white text-green-600 shadow-sm'
								: 'text-gray-400'}"
						>
							Pemasukan
						</button>
					</div>

					<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
						<div class="col-span-2">
							<label
								for="title"
								class="mb-2 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase"
								>Keterangan</label
							>
							<input
								id="title"
								type="text"
								required
								bind:value={newTx.title}
								placeholder="Beli apa hari ini?"
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
							/>
						</div>

						<div class="col-span-2">
							<label
								for="amount"
								class="mb-2 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase"
								>Nominal</label
							>
							<div class="relative">
								<span
									class="absolute inset-y-0 left-5 flex items-center text-sm font-bold text-gray-400"
									>Rp</span
								>
								<input
									id="amount"
									type="number"
									required
									bind:value={newTx.amount}
									class="w-full rounded-2xl border border-gray-100 bg-gray-50 py-3.5 pr-5 pl-12 transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
								/>
							</div>
						</div>

						<div>
							<label
								for="account"
								class="mb-2 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase"
								>Pilih Rekening</label
							>
							<select
								id="account"
								bind:value={newTx.account_id}
								class="w-full cursor-pointer appearance-none rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
							>
								{#each myAccounts as acc (acc.id)}
									<option value={acc.id}>{acc.name}</option>
								{/each}
							</select>
						</div>

						<div>
							<label
								for="category"
								class="mb-2 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase"
								>Kategori</label
							>
							<select
								id="category"
								bind:value={newTx.category_id}
								class="w-full cursor-pointer appearance-none rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
							>
								{#each filteredCategories as cat (cat.id)}
									<option value={cat.id}>{cat.name}</option>
								{/each}
							</select>
						</div>

						<div class="col-span-2">
							<label
								for="date"
								class="mb-2 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase"
								>Tanggal</label
							>
							<input
								id="date"
								type="date"
								required
								bind:value={newTx.date}
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
							/>
						</div>
					</div>

					<button
						type="submit"
						class="mt-4 w-full rounded-2xl bg-[#0a2e31] py-4 font-bold text-white shadow-xl transition-all hover:bg-black active:scale-[0.98]"
					>
						Simpan Transaksi
					</button>
				</form>
			</div>
		</div>
	</div>
{/if}
