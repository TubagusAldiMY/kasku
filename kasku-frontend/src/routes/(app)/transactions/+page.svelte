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
		toAccount: string;
		amount: number;
		type: 'INCOME' | 'EXPENSE' | 'TRANSFER';
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
		type: 'EXPENSE' as 'INCOME' | 'EXPENSE' | 'TRANSFER',
		category_id: '',
		account_id: '',
		to_account_id: '',
		date: new Date().toISOString().split('T')[0]
	});

	const filteredCategories = $derived(
		allCategories.filter((c) => c.category_type === newTx.type || c.category_type === 'BOTH')
	);

	$effect(() => {
		if (newTx.type === 'TRANSFER' && newTx.to_account_id === newTx.account_id) {
			const other = myAccounts.find((a) => a.id !== newTx.account_id);
			newTx.to_account_id = other ? other.id : '';
		}
	});

	function projectTransaction(
		row: TransactionRow,
		accounts: AccountRow[],
		categories: CategoryRow[]
	): Transaction {
		const acc = accounts.find((a) => a.id === row.account_id);
		const toAcc = row.to_account_id ? accounts.find((a) => a.id === row.to_account_id) : undefined;
		const cat = categories.find((c) => c.id === row.category_id);
		const signed =
			row.transaction_type === 'EXPENSE' || row.transaction_type === 'TRANSFER'
				? -row.amount_idr
				: row.amount_idr;
		return {
			id: row.id,
			date: row.transaction_date,
			title: row.notes ?? row.transaction_type,
			category: cat?.name ?? (row.transaction_type === 'TRANSFER' ? 'Transfer' : 'Umum'),
			account: acc?.name ?? '—',
			toAccount: toAcc?.name ?? '',
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

	let transferError = $state('');

	async function handleAddTransaction(e: SubmitEvent) {
		e.preventDefault();
		transferError = '';

		if (newTx.type === 'TRANSFER') {
			if (!newTx.to_account_id || newTx.to_account_id === newTx.account_id) {
				transferError = 'Rekening sumber dan tujuan tidak boleh sama.';
				return;
			}
			const sourceAcc = await accountsRepo.getById(newTx.account_id);
			const amount = Math.abs(newTx.amount);
			if (sourceAcc && amount > sourceAcc.balance) {
				const fmt = new Intl.NumberFormat('id-ID', {
					style: 'currency',
					currency: 'IDR',
					minimumFractionDigits: 0
				});
				transferError = `Saldo ${sourceAcc.name} tidak mencukupi. Tersedia: ${fmt.format(sourceAcc.balance)}.`;
				return;
			}
		}

		try {
			const payload: Partial<TransactionRow> = {
				account_id: newTx.account_id,
				category_id: newTx.category_id,
				transaction_type: newTx.type,
				amount_idr: Math.abs(newTx.amount),
				transaction_date: newTx.date,
				notes: newTx.title
			};
			if (newTx.type === 'TRANSFER' && newTx.to_account_id) {
				payload.to_account_id = newTx.to_account_id;
			}
			await enqueueCreate<TransactionRow>('transactions', payload as TransactionRow);
			transferError = '';
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

	function formatDateShort(dateStr: string) {
		const d = new Date(dateStr);
		return d.toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' });
	}
</script>

<div class="animate-in fade-in space-y-6 duration-500">
	<!-- Header -->
	<div class="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
		<div>
			<h1 class="text-xl font-bold text-[#0a2e31] sm:text-2xl">Riwayat Transaksi</h1>
			<p class="text-sm text-gray-500">Pantau arus kas masuk dan keluar Anda.</p>
		</div>
		<button
			onclick={() => (showAddModal = true)}
			class="inline-flex items-center justify-center gap-2 rounded-2xl bg-[#217b84] px-5 py-3 font-bold text-white shadow-lg transition-all hover:bg-[#1a5f66] active:scale-95"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4" />
			</svg>
			Catat Transaksi
		</button>
	</div>

	<!-- List Transaksi -->
	{#if loading}
		<div class="space-y-3">
			{#each [0, 1, 2, 3] as i (i)}
				<div class="h-20 animate-pulse rounded-2xl bg-white"></div>
			{/each}
		</div>
	{:else if transactions.length === 0}
		<div class="flex flex-col items-center gap-4 rounded-[2rem] bg-white py-16 text-center shadow-sm">
			<div class="flex h-16 w-16 items-center justify-center rounded-full bg-gray-50 text-gray-300">
				<svg class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
				</svg>
			</div>
			<p class="font-bold text-[#0a2e31]">Belum ada transaksi</p>
			<p class="text-sm text-gray-400">Tekan "Catat Transaksi" untuk mulai mencatat.</p>
		</div>
	{:else}
		<!-- Mobile: Card List -->
		<div class="space-y-3 lg:hidden">
			{#each transactions as tx (tx.id)}
				<div
					class="flex items-center gap-3 rounded-2xl bg-white p-4 shadow-sm"
					in:fly={{ y: 8, duration: 200 }}
				>
					<!-- Indikator tipe -->
					<div
						class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full {tx.type === 'INCOME'
							? 'bg-green-50'
							: tx.type === 'TRANSFER'
								? 'bg-blue-50'
								: 'bg-red-50'}"
					>
						{#if tx.type === 'INCOME'}
							<svg class="h-5 w-5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
								<path stroke-linecap="round" stroke-linejoin="round" d="M5 10l7-7m0 0l7 7m-7-7v18" />
							</svg>
						{:else if tx.type === 'TRANSFER'}
							<svg class="h-5 w-5 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
								<path stroke-linecap="round" stroke-linejoin="round" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
							</svg>
						{:else}
							<svg class="h-5 w-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
								<path stroke-linecap="round" stroke-linejoin="round" d="M19 14l-7 7m0 0l-7-7m7 7V3" />
							</svg>
						{/if}
					</div>

					<!-- Info -->
					<div class="min-w-0 flex-1">
						<p class="truncate text-sm font-bold text-[#0a2e31]">{tx.title}</p>
						<div class="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-0.5">
							<span
								class="rounded-full px-2 py-0.5 text-[10px] font-bold tracking-tight uppercase {tx.type === 'TRANSFER'
									? 'bg-blue-50 text-blue-600'
									: 'bg-teal-50 text-teal-700'}"
							>{tx.category}</span>
							<span class="text-[11px] text-gray-400">
								{tx.account}{#if tx.toAccount && tx.type === 'TRANSFER'} → {tx.toAccount}{/if}
							</span>
						</div>
						<p class="mt-0.5 text-[11px] text-gray-400">{formatDateShort(tx.date)}</p>
					</div>

					<!-- Nominal + hapus -->
					<div class="flex shrink-0 flex-col items-end gap-2">
						<span
							class="text-sm font-black {tx.type === 'INCOME'
								? 'text-green-600'
								: tx.type === 'TRANSFER'
									? 'text-blue-500'
									: 'text-red-500'}"
						>{formatCurrency(tx.amount)}</span>
						<button
							onclick={() => handleDeleteTransaction(tx.id)}
							class="rounded-lg p-1.5 text-gray-300 transition-colors hover:bg-red-50 hover:text-red-500 active:bg-red-100"
							aria-label="Hapus transaksi"
						>
							<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
							</svg>
						</button>
					</div>
				</div>
			{/each}
		</div>

		<!-- Desktop: Table -->
		<div class="hidden overflow-hidden rounded-[2.5rem] border border-gray-100 bg-white text-sm shadow-sm lg:block">
			<div class="overflow-x-auto">
				<table class="w-full border-collapse text-left">
					<thead>
						<tr class="bg-gray-50/50">
							<th class="px-8 py-5 text-[10px] font-bold tracking-widest text-[#0a2e31] uppercase">Tanggal</th>
							<th class="px-8 py-5 text-[10px] font-bold tracking-widest text-[#0a2e31] uppercase">Keterangan</th>
							<th class="px-8 py-5 text-[10px] font-bold tracking-widest text-[#0a2e31] uppercase">Kategori</th>
							<th class="px-8 py-5 text-[10px] font-bold tracking-widest text-[#0a2e31] uppercase">Akun</th>
							<th class="px-8 py-5 text-right text-[10px] font-bold tracking-widest text-[#0a2e31] uppercase">Nominal</th>
							<th class="px-8 py-5"></th>
						</tr>
					</thead>
					<tbody class="divide-y divide-gray-50">
						{#each transactions as tx (tx.id)}
							<tr class="group transition-colors hover:bg-gray-50/80">
								<td class="px-8 py-5 whitespace-nowrap text-gray-500">{tx.date}</td>
								<td class="px-8 py-5 font-bold text-[#0a2e31]">{tx.title}</td>
								<td class="px-8 py-5">
									<span class="rounded-full px-3 py-1 text-[11px] font-bold tracking-tight uppercase {tx.type === 'TRANSFER' ? 'bg-blue-50 text-blue-600' : 'bg-teal-50 text-teal-700'}">
										{tx.category}
									</span>
								</td>
								<td class="px-8 py-5 font-medium text-gray-500">
									{tx.account}{#if tx.toAccount && tx.type === 'TRANSFER'}<span class="text-blue-500"> → {tx.toAccount}</span>{/if}
								</td>
								<td class="px-8 py-5 text-right font-black tracking-tight {tx.type === 'INCOME' ? 'text-green-600' : tx.type === 'TRANSFER' ? 'text-blue-500' : 'text-red-500'}">
									{formatCurrency(tx.amount)}
								</td>
								<td class="px-8 py-5 text-right">
									<button
										onclick={() => handleDeleteTransaction(tx.id)}
										class="p-2 text-gray-300 opacity-0 transition-colors group-hover:opacity-100 hover:text-red-500"
										aria-label="Hapus transaksi"
									>
										<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
										</svg>
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>

<!-- Modal Tambah Transaksi -->
{#if showAddModal}
	<div
		class="fixed inset-0 z-50 flex items-end justify-center bg-[#0a2e31]/40 backdrop-blur-sm sm:items-center sm:p-4"
		in:fade={{ duration: 200 }}
	>
		<div
			class="w-full max-h-[92dvh] overflow-y-auto rounded-t-[2rem] bg-white shadow-2xl sm:max-w-lg sm:rounded-[2.5rem]"
			in:fly={{ y: 40, duration: 350 }}
		>
			<!-- Handle bar mobile -->
			<div class="flex justify-center pt-3 pb-1 sm:hidden">
				<div class="h-1 w-10 rounded-full bg-gray-200"></div>
			</div>

			<div class="space-y-5 p-5 sm:p-8">
				<div class="flex items-center justify-between">
					<h2 class="text-xl font-bold text-[#0a2e31] sm:text-2xl">Catat Transaksi</h2>
					<button
						onclick={() => { showAddModal = false; transferError = ''; }}
						class="rounded-xl p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
						aria-label="Tutup modal"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<form onsubmit={handleAddTransaction} class="space-y-4">
					<!-- Toggle Tipe -->
					<div class="flex rounded-2xl border border-gray-100 bg-gray-50 p-1.5">
						<button
							type="button"
							onclick={() => {
								newTx.type = 'EXPENSE';
								const first = allCategories.find((c) => c.category_type === 'EXPENSE');
								newTx.category_id = first ? first.id : '';
							}}
							class="flex-1 rounded-xl py-2.5 text-sm font-bold transition-all {newTx.type === 'EXPENSE' ? 'bg-white text-red-500 shadow-sm' : 'text-gray-400'}"
						>Pengeluaran</button>
						<button
							type="button"
							onclick={() => {
								newTx.type = 'INCOME';
								const first = allCategories.find((c) => c.category_type === 'INCOME');
								newTx.category_id = first ? first.id : '';
							}}
							class="flex-1 rounded-xl py-2.5 text-sm font-bold transition-all {newTx.type === 'INCOME' ? 'bg-white text-green-600 shadow-sm' : 'text-gray-400'}"
						>Pemasukan</button>
						<button
							type="button"
							onclick={() => {
								newTx.type = 'TRANSFER';
								newTx.category_id = '';
								const second = myAccounts.find((a) => a.id !== newTx.account_id);
								newTx.to_account_id = second ? second.id : '';
							}}
							class="flex-1 rounded-xl py-2.5 text-sm font-bold transition-all {newTx.type === 'TRANSFER' ? 'bg-white text-blue-500 shadow-sm' : 'text-gray-400'}"
						>Transfer</button>
					</div>

					<!-- Keterangan -->
					<div>
						<label for="title" class="mb-1.5 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase">Keterangan</label>
						<input
							id="title"
							type="text"
							required
							bind:value={newTx.title}
							placeholder="Beli apa hari ini?"
							class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
						/>
					</div>

					<!-- Nominal -->
					<div>
						<label for="amount" class="mb-1.5 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase">Nominal</label>
						<div class="relative">
							<span class="absolute inset-y-0 left-4 flex items-center text-sm font-bold text-gray-400">Rp</span>
							<input
								id="amount"
								type="number"
								required
								bind:value={newTx.amount}
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 py-3 pr-4 pl-11 text-sm transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
							/>
						</div>
					</div>

					<!-- Rekening & Kategori (2 kolom di sm+, 1 kolom di mobile) -->
					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<div>
							<label for="account" class="mb-1.5 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase">
								{newTx.type === 'TRANSFER' ? 'Rekening Asal' : 'Rekening'}
							</label>
							<select
								id="account"
								bind:value={newTx.account_id}
								class="w-full cursor-pointer appearance-none rounded-2xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
							>
								{#each myAccounts as acc (acc.id)}
									<option value={acc.id}>{acc.name}</option>
								{/each}
							</select>
						</div>

						{#if newTx.type === 'TRANSFER'}
							<div>
								<label for="to_account" class="mb-1.5 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase">Rekening Tujuan</label>
								<select
									id="to_account"
									bind:value={newTx.to_account_id}
									required
									class="w-full cursor-pointer appearance-none rounded-2xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
								>
									{#each myAccounts.filter((a) => a.id !== newTx.account_id) as acc (acc.id)}
										<option value={acc.id}>{acc.name}</option>
									{/each}
								</select>
							</div>
						{:else}
							<div>
								<label for="category" class="mb-1.5 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase">Kategori</label>
								<select
									id="category"
									bind:value={newTx.category_id}
									class="w-full cursor-pointer appearance-none rounded-2xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
								>
									{#each filteredCategories as cat (cat.id)}
										<option value={cat.id}>{cat.name}</option>
									{/each}
								</select>
							</div>
						{/if}
					</div>

					<!-- Tanggal -->
					<div>
						<label for="date" class="mb-1.5 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase">Tanggal</label>
						<input
							id="date"
							type="date"
							required
							bind:value={newTx.date}
							class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
						/>
					</div>

					{#if transferError}
						<p class="rounded-2xl bg-red-50 px-4 py-3 text-sm font-medium text-red-600">{transferError}</p>
					{/if}

					<button
						type="submit"
						class="w-full rounded-2xl bg-[#0a2e31] py-4 font-bold text-white shadow-xl transition-all hover:bg-black active:scale-[0.98]"
					>
						Simpan Transaksi
					</button>
				</form>
			</div>
		</div>
	</div>
{/if}
