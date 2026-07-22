<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { fade, fly } from 'svelte/transition';

	type TxType = 'INCOME' | 'EXPENSE' | 'TRANSFER';
	type CategoryType = 'INCOME' | 'EXPENSE' | 'BOTH';

	type Transaction = {
		id: string;
		date: string;
		title: string;
		category: string;
		account: string;
		toAccount: string;
		budget: string;
		amount: number;
		type: TxType;
	};
	type AccountRef = { id: string; name: string; balance: number };
	type CategoryRef = { id: string; name: string; category_type: CategoryType };
	type BudgetRef = { id: string; name: string };

	// Raw transaksi (server → normalized snake_case) untuk proyeksi tampilan & edit.
	type TxRow = {
		id: string;
		account_id: string;
		category_id: string;
		budget_id: string;
		transaction_type: TxType;
		amount_idr: number;
		transaction_date: string;
		notes: string;
		to_account_id: string;
	};

	// Envelope server bisa snake_case (punya json tag) atau PascalCase (entity tanpa tag).
	// eslint-disable-next-line @typescript-eslint/no-explicit-any -- bentuk respons bervariasi (snake_case vs PascalCase)
	type ServerObj = Record<string, any>;

	let txRows = $state<TxRow[]>([]);
	let loading = $state(true);
	let errorMessage = $state('');
	let showAddModal = $state(false);
	let editingId = $state('');

	let myAccounts = $state<AccountRef[]>([]);
	let allCategories = $state<CategoryRef[]>([]);
	let budgets = $state<BudgetRef[]>([]);

	let newTx = $state({
		title: '',
		amount: 0,
		type: 'EXPENSE' as TxType,
		category_id: '',
		budget_id: '',
		account_id: '',
		to_account_id: '',
		date: new Date().toISOString().split('T')[0]
	});

	const filteredCategories = $derived(
		allCategories.filter((c) => c.category_type === newTx.type || c.category_type === 'BOTH')
	);

	function firstCategoryId(type: TxType) {
		if (type === 'TRANSFER') return '';
		return (
			allCategories.find((c) => c.category_type === type || c.category_type === 'BOTH')?.id ?? ''
		);
	}

	$effect(() => {
		if (newTx.type === 'TRANSFER' && newTx.to_account_id === newTx.account_id) {
			const other = myAccounts.find((a) => a.id !== newTx.account_id);
			newTx.to_account_id = other ? other.id : '';
		}
		if (newTx.type !== 'TRANSFER' && filteredCategories.length > 0) {
			const stillValid = filteredCategories.some((c) => c.id === newTx.category_id);
			if (!stillValid) newTx.category_id = filteredCategories[0].id;
		}
		if (newTx.type !== 'EXPENSE') {
			newTx.budget_id = '';
		}
	});

	function projectTransaction(row: TxRow): Transaction {
		const acc = myAccounts.find((a) => a.id === row.account_id);
		const toAcc = row.to_account_id
			? myAccounts.find((a) => a.id === row.to_account_id)
			: undefined;
		const cat = allCategories.find((c) => c.id === row.category_id);
		const budget = row.budget_id ? budgets.find((b) => b.id === row.budget_id) : undefined;
		const signed =
			row.transaction_type === 'EXPENSE' || row.transaction_type === 'TRANSFER'
				? -row.amount_idr
				: row.amount_idr;
		return {
			id: row.id,
			date: row.transaction_date,
			title: row.notes || row.transaction_type,
			category: cat?.name ?? (row.transaction_type === 'TRANSFER' ? 'Transfer' : 'Umum'),
			account: acc?.name ?? '—',
			toAccount: toAcc?.name ?? '',
			budget: budget?.name ?? '',
			amount: signed,
			type: row.transaction_type
		};
	}

	const transactions = $derived(
		txRows.map(projectTransaction).sort((a, b) => (a.date < b.date ? 1 : -1))
	);

	// ── Normalisasi respons server (snake_case ATAU PascalCase) ──
	function normalizeAccount(a: ServerObj): AccountRef | null {
		const id = a.id ?? a.ID;
		const name = a.name ?? a.Name;
		if (!id || !name) return null;
		return { id, name, balance: Number(a.balance ?? a.Balance ?? 0) };
	}

	function normalizeCategory(c: ServerObj): CategoryRef | null {
		const id = c.id ?? c.ID;
		const name = c.name ?? c.Name;
		const category_type = c.category_type ?? c.CategoryType;
		if (!id || !name || !category_type) return null;
		return { id, name, category_type };
	}

	function normalizeBudget(b: ServerObj): BudgetRef | null {
		const id = b.id ?? b.ID;
		const name = b.name ?? b.Name;
		if (!id || !name) return null;
		return { id, name };
	}

	function normalizeTx(t: ServerObj): TxRow | null {
		const id = t.id ?? t.ID;
		const account_id = t.account_id ?? t.AccountID;
		const transaction_type = t.transaction_type ?? t.TransactionType;
		if (!id || !account_id || !transaction_type) return null;
		return {
			id,
			account_id,
			category_id: t.category_id ?? t.CategoryID ?? '',
			budget_id: t.budget_id ?? t.BudgetID ?? '',
			transaction_type,
			amount_idr: Number(t.amount_idr ?? t.AmountIDR ?? 0),
			transaction_date: t.transaction_date ?? t.TransactionDate ?? '',
			notes: t.notes ?? t.Notes ?? '',
			to_account_id: t.to_account_id ?? t.ToAccountID ?? ''
		};
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

	function normalizeList<T>(data: unknown, fn: (o: ServerObj) => T | null): T[] {
		if (!Array.isArray(data)) return [];
		return data.map((item) => fn(item as ServerObj)).filter((v): v is T => v !== null);
	}

	async function loadData(showLoading = false) {
		if (showLoading) loading = true;
		errorMessage = '';
		try {
			const [accRes, catRes, budRes, txRes] = await Promise.all([
				apiFetch('/accounts'),
				apiFetch('/categories'),
				apiFetch('/budgets'),
				apiFetch('/transactions')
			]);
			const [accR, catR, budR, txR] = await Promise.all([
				readApiResult(accRes),
				readApiResult(catRes),
				readApiResult(budRes),
				readApiResult(txRes)
			]);

			if (accRes.ok && accR.success) {
				myAccounts = normalizeList(accR.data, normalizeAccount);
			} else {
				errorMessage = accR.error?.message || 'Gagal memuat rekening.';
			}
			if (catRes.ok && catR.success) {
				allCategories = normalizeList(catR.data, normalizeCategory);
			} else if (!errorMessage) {
				errorMessage = catR.error?.message || 'Gagal memuat kategori.';
			}
			// Anggaran opsional (tier tertentu) — jangan jadikan error fatal.
			budgets = budRes.ok && budR.success ? normalizeList(budR.data, normalizeBudget) : [];
			if (txRes.ok && txR.success) {
				txRows = normalizeList(txR.data, normalizeTx);
			} else if (!errorMessage) {
				errorMessage = txR.error?.message || 'Gagal memuat transaksi.';
			}

			if (!newTx.account_id && myAccounts.length > 0) newTx.account_id = myAccounts[0].id;
			if (!newTx.category_id && filteredCategories.length > 0)
				newTx.category_id = filteredCategories[0].id;
		} catch (err) {
			console.error(err);
			errorMessage = 'Gagal memuat data. Periksa koneksi atau service backend.';
		} finally {
			if (showLoading) loading = false;
		}
	}

	let transferError = $state('');

	function resetForm() {
		newTx = {
			title: '',
			amount: 0,
			type: 'EXPENSE',
			category_id: firstCategoryId('EXPENSE'),
			budget_id: '',
			account_id: myAccounts[0]?.id ?? '',
			to_account_id: '',
			date: new Date().toISOString().split('T')[0]
		};
		editingId = '';
		transferError = '';
	}

	function openEditTransaction(id: string) {
		const row = txRows.find((t) => t.id === id);
		if (!row) return;
		newTx = {
			title: row.notes ?? '',
			amount: row.amount_idr,
			type: row.transaction_type,
			category_id: row.category_id ?? '',
			budget_id: row.transaction_type === 'EXPENSE' ? (row.budget_id ?? '') : '',
			account_id: row.account_id,
			to_account_id: row.to_account_id ?? '',
			date: (row.transaction_date ?? '').split('T')[0]
		};
		editingId = id;
		transferError = '';
		showAddModal = true;
	}

	async function handleSaveTransaction(e: SubmitEvent) {
		e.preventDefault();
		transferError = '';

		if (newTx.type === 'TRANSFER') {
			if (!newTx.to_account_id || newTx.to_account_id === newTx.account_id) {
				transferError = 'Rekening sumber dan tujuan tidak boleh sama.';
				return;
			}
			const sourceAcc = myAccounts.find((a) => a.id === newTx.account_id);
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
			const payload = {
				account_id: newTx.account_id,
				category_id: newTx.category_id,
				budget_id: newTx.type === 'EXPENSE' ? newTx.budget_id : '',
				transaction_type: newTx.type,
				amount_idr: Math.abs(newTx.amount),
				transaction_date: newTx.date,
				notes: newTx.title,
				to_account_id: newTx.type === 'TRANSFER' ? newTx.to_account_id : ''
			};
			const method = editingId ? 'PUT' : 'POST';
			const url = editingId ? `/transactions/${editingId}` : '/transactions';
			const res = await apiFetch(url, { method, body: JSON.stringify(payload) });
			const result = await readApiResult(res);
			if (res.ok && result.success) {
				transferError = '';
				showAddModal = false;
				editingId = '';
				await loadData();
			} else {
				transferError = result.error?.message || 'Gagal menyimpan transaksi.';
			}
		} catch (err) {
			console.error('Gagal menyimpan transaksi:', err);
			transferError = 'Gagal menyimpan transaksi. Periksa koneksi atau service backend.';
		}
	}

	async function handleDeleteTransaction(id: string) {
		if (!confirm('Hapus transaksi ini? Saldo rekening Anda akan disesuaikan kembali.')) return;
		try {
			const res = await apiFetch(`/transactions/${id}`, { method: 'DELETE' });
			const result = await readApiResult(res);
			if (res.ok && result.success) {
				await loadData();
			} else {
				errorMessage = result.error?.message || 'Gagal menghapus transaksi.';
			}
		} catch (err) {
			console.error('Gagal menghapus transaksi:', err);
			errorMessage = 'Gagal menghapus transaksi. Periksa koneksi atau service backend.';
		}
	}

	onMount(() => loadData(true));

	// ── Editorial presentation helpers (additive, no logic change) ──
	// Dense signed money for the table: "−45.000" / "+12.500.000" (no "Rp").
	function fmtSigned(amount: number) {
		const sign = amount < 0 ? '−' : '+';
		return sign + new Intl.NumberFormat('id-ID').format(Math.abs(amount));
	}

	function fmtDayShort(dateStr: string) {
		return new Date(dateStr).toLocaleDateString('id-ID', { day: 'numeric', month: 'short' });
	}
</script>

<div class="pb-4">
	<!-- ═══════════ Header ═══════════ -->
	<div
		class="flex flex-col justify-between gap-5 border-b border-ink/10 pb-8 sm:flex-row sm:items-end"
	>
		<div>
			<h1 class="font-serif text-4xl tracking-tight text-ink">Riwayat transaksi</h1>
			<p class="mt-2.5 text-sm text-ink/60">
				{transactions.length} transaksi tercatat.
			</p>
		</div>
		<button
			onclick={() => {
				resetForm();
				showAddModal = true;
			}}
			class="shrink-0 rounded-full bg-teal px-5 py-2.5 text-[13px] font-semibold text-card transition-colors hover:bg-ink"
		>
			+ Catat
		</button>
	</div>

	{#if errorMessage && !showAddModal}
		<div class="mt-6 rounded-[10px] border border-clay/25 bg-clay/5 px-4 py-3 text-sm text-clay">
			{errorMessage}
		</div>
	{/if}

	<!-- List Transaksi -->
	{#if loading}
		<div class="space-y-3 pt-6">
			{#each [0, 1, 2, 3] as i (i)}
				<div class="h-10 animate-pulse rounded bg-ink/5"></div>
			{/each}
		</div>
	{:else if transactions.length === 0}
		<div class="flex flex-col items-center gap-3 py-20 text-center">
			<p class="font-serif text-2xl text-ink">Belum ada transaksi</p>
			<p class="text-sm text-ink/45">Tekan "+ Catat" untuk mulai mencatat.</p>
		</div>
	{:else}
		<!-- Mobile: editorial list -->
		<div class="pt-4 lg:hidden">
			{#each transactions as tx, i (tx.id)}
				<div
					class="group flex items-baseline justify-between gap-3 border-b border-ink/8 py-4"
					in:fly={{ y: 8, duration: 200, delay: Math.min(i * 30, 300) }}
				>
					<div class="min-w-0 flex-1">
						<div class="flex items-baseline gap-2.5">
							<span class="truncate text-sm font-medium text-ink">{tx.title}</span>
							<span class="shrink-0 text-xs text-ink/40">{fmtDayShort(tx.date)}</span>
						</div>
						<div class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1">
							<span
								class="rounded-full border px-2 py-px text-[11px] {tx.type === 'TRANSFER'
									? 'border-steel/30 text-steel'
									: 'border-ink/15 text-ink/55'}"
							>
								{tx.category}
							</span>
							<span class="text-[11px] text-ink/45">
								{tx.account}{#if tx.toAccount && tx.type === 'TRANSFER'}
									<span class="text-steel"> → {tx.toAccount}</span>{/if}
							</span>
							{#if tx.budget}
								<span class="rounded-full border border-gold/30 px-2 py-px text-[11px] text-gold">
									{tx.budget}
								</span>
							{/if}
						</div>
					</div>

					<div class="flex shrink-0 flex-col items-end gap-1.5">
						<span
							class="text-sm tabular-nums {tx.type === 'INCOME'
								? 'font-semibold text-teal'
								: tx.type === 'TRANSFER'
									? 'text-steel'
									: 'text-ink'}"
						>
							{fmtSigned(tx.amount)}
						</span>
						<div class="flex items-center gap-1">
							<button
								onclick={() => openEditTransaction(tx.id)}
								class="rounded-md p-1 text-ink/30 transition-colors hover:text-teal"
								aria-label="Edit transaksi"
							>
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"
									/>
								</svg>
							</button>
							<button
								onclick={() => handleDeleteTransaction(tx.id)}
								class="rounded-md p-1 text-ink/30 transition-colors hover:text-clay"
								aria-label="Hapus transaksi"
							>
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
									/>
								</svg>
							</button>
						</div>
					</div>
				</div>
			{/each}
		</div>

		<!-- Desktop: editorial table (CSS grid) -->
		<div class="hidden pt-4 lg:block">
			<div
				class="grid grid-cols-[90px_1.6fr_1fr_1fr_1fr_150px_40px] gap-4 border-b border-ink/25 py-3 text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
			>
				<span>Tanggal</span>
				<span>Keterangan</span>
				<span>Kategori</span>
				<span>Anggaran</span>
				<span>Akun</span>
				<span class="text-right">Nominal</span>
				<span></span>
			</div>
			{#each transactions as tx (tx.id)}
				<div
					class="group grid grid-cols-[90px_1.6fr_1fr_1fr_1fr_150px_40px] items-baseline gap-4 border-b border-ink/8 py-4 text-[13.5px]"
				>
					<span class="text-[12.5px] text-ink/45">{fmtDayShort(tx.date)}</span>
					<span class="truncate font-medium text-ink">{tx.title}</span>
					<span>
						<span
							class="rounded-full border px-2 py-px text-[11px] {tx.type === 'TRANSFER'
								? 'border-steel/30 text-steel'
								: 'border-ink/15 text-ink/70'}"
						>
							{tx.category}
						</span>
					</span>
					<span>
						{#if tx.budget}
							<span class="rounded-full border border-gold/30 px-2 py-px text-[11px] text-gold">
								{tx.budget}
							</span>
						{:else}
							<span class="text-xs text-ink/35">Tanpa anggaran</span>
						{/if}
					</span>
					<span class="truncate text-ink/60">
						{tx.account}{#if tx.toAccount && tx.type === 'TRANSFER'}<span class="text-steel">
								→ {tx.toAccount}</span
							>{/if}
					</span>
					<span
						class="text-right tabular-nums {tx.type === 'INCOME'
							? 'font-semibold text-teal'
							: tx.type === 'TRANSFER'
								? 'text-steel'
								: 'text-ink'}"
					>
						{fmtSigned(tx.amount)}
					</span>
					<span
						class="flex justify-end gap-0.5 opacity-0 transition-opacity group-hover:opacity-100"
					>
						<button
							onclick={() => openEditTransaction(tx.id)}
							class="p-1 text-ink/30 transition-colors hover:text-teal"
							aria-label="Edit transaksi"
						>
							<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"
								/>
							</svg>
						</button>
						<button
							onclick={() => handleDeleteTransaction(tx.id)}
							class="p-1 text-ink/30 transition-colors hover:text-clay"
							aria-label="Hapus transaksi"
						>
							<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
								/>
							</svg>
						</button>
					</span>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Modal Tambah Transaksi -->
{#if showAddModal}
	<div
		class="fixed inset-0 z-50 flex items-end justify-center bg-black/60 backdrop-blur-sm sm:items-center sm:p-4"
		in:fade={{ duration: 200 }}
		out:fade={{ duration: 150 }}
	>
		<div
			class="max-h-[92dvh] w-full overflow-y-auto rounded-t-2xl border border-ink/10 bg-card shadow-xl sm:max-w-lg sm:rounded-2xl"
			style="padding-bottom: env(safe-area-inset-bottom);"
			in:fly={{ y: 40, duration: 350 }}
			out:fly={{ y: 40, duration: 250 }}
		>
			<!-- Handle bar mobile -->
			<div class="flex justify-center pt-3 pb-1 sm:hidden">
				<div class="h-1 w-10 rounded-full bg-ink/15"></div>
			</div>

			<div class="space-y-5 p-5 sm:p-8">
				<div class="flex items-center justify-between">
					<h2 class="font-serif text-2xl text-ink">
						{editingId ? 'Edit transaksi' : 'Catat transaksi'}
					</h2>
					<button
						onclick={() => {
							showAddModal = false;
							resetForm();
						}}
						class="rounded-full p-2 text-ink/40 transition-colors hover:bg-ink/5 hover:text-ink"
						aria-label="Tutup modal"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M6 18L18 6M6 6l12 12"
							/>
						</svg>
					</button>
				</div>

				<form onsubmit={handleSaveTransaction} class="space-y-4">
					<!-- Toggle Tipe -->
					<div class="flex rounded-full border border-ink/20 p-1 text-[13px] font-semibold">
						<button
							type="button"
							onclick={() => {
								newTx.type = 'EXPENSE';
								newTx.category_id = firstCategoryId('EXPENSE');
							}}
							class="flex-1 rounded-full py-2 transition-colors {newTx.type === 'EXPENSE'
								? 'bg-ink text-card'
								: 'text-ink/60'}">Pengeluaran</button
						>
						<button
							type="button"
							onclick={() => {
								newTx.type = 'INCOME';
								newTx.category_id = firstCategoryId('INCOME');
							}}
							class="flex-1 rounded-full py-2 transition-colors {newTx.type === 'INCOME'
								? 'bg-ink text-card'
								: 'text-ink/60'}">Pemasukan</button
						>
						<button
							type="button"
							onclick={() => {
								newTx.type = 'TRANSFER';
								newTx.category_id = '';
								const second = myAccounts.find((a) => a.id !== newTx.account_id);
								newTx.to_account_id = second ? second.id : '';
							}}
							class="flex-1 rounded-full py-2 transition-colors {newTx.type === 'TRANSFER'
								? 'bg-ink text-card'
								: 'text-ink/60'}">Transfer</button
						>
					</div>

					<!-- Keterangan -->
					<div>
						<label
							for="title"
							class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
							>Keterangan</label
						>
						<input
							id="title"
							type="text"
							required
							bind:value={newTx.title}
							placeholder="Beli apa hari ini?"
							class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink outline-none focus:border-teal"
						/>
					</div>

					<!-- Nominal -->
					<div>
						<label
							for="amount"
							class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
							>Nominal</label
						>
						<div class="relative">
							<span class="absolute inset-y-0 left-4 flex items-center text-sm text-ink/45">Rp</span
							>
							<input
								id="amount"
								type="number"
								min="1"
								required
								bind:value={newTx.amount}
								class="w-full rounded-[10px] border border-ink/25 bg-field py-3 pr-4 pl-11 text-sm text-ink tabular-nums outline-none focus:border-teal"
							/>
						</div>
					</div>

					<!-- Rekening & Kategori (2 kolom di sm+, 1 kolom di mobile) -->
					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<div>
							<label
								for="account"
								class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
							>
								{newTx.type === 'TRANSFER' ? 'Rekening asal' : 'Rekening'}
							</label>
							<select
								id="account"
								bind:value={newTx.account_id}
								class="w-full cursor-pointer appearance-none rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink outline-none focus:border-teal"
							>
								{#each myAccounts as acc (acc.id)}
									<option value={acc.id}>{acc.name}</option>
								{/each}
							</select>
						</div>

						{#if newTx.type === 'TRANSFER'}
							<div>
								<label
									for="to_account"
									class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
									>Rekening tujuan</label
								>
								<select
									id="to_account"
									bind:value={newTx.to_account_id}
									required
									class="w-full cursor-pointer appearance-none rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink outline-none focus:border-teal"
								>
									{#each myAccounts.filter((a) => a.id !== newTx.account_id) as acc (acc.id)}
										<option value={acc.id}>{acc.name}</option>
									{/each}
								</select>
							</div>
						{:else}
							<div>
								<label
									for="category"
									class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
									>Kategori</label
								>
								<select
									id="category"
									bind:value={newTx.category_id}
									class="w-full cursor-pointer appearance-none rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink outline-none focus:border-teal"
								>
									{#each filteredCategories as cat (cat.id)}
										<option value={cat.id}>{cat.name}</option>
									{/each}
								</select>
							</div>
						{/if}
					</div>

					{#if newTx.type === 'EXPENSE'}
						<div>
							<label
								for="budget"
								class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>Anggaran</label
							>
							<select
								id="budget"
								bind:value={newTx.budget_id}
								class="w-full cursor-pointer appearance-none rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink outline-none focus:border-teal"
							>
								<option value="">Tanpa anggaran</option>
								{#each budgets as budget (budget.id)}
									<option value={budget.id}>{budget.name}</option>
								{/each}
							</select>
						</div>
					{/if}

					<!-- Tanggal -->
					<div>
						<label
							for="date"
							class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
							>Tanggal</label
						>
						<input
							id="date"
							type="date"
							required
							bind:value={newTx.date}
							class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink outline-none focus:border-teal"
						/>
					</div>

					{#if transferError}
						<p class="rounded-[10px] border border-clay/25 bg-clay/5 px-4 py-3 text-sm text-clay">
							{transferError}
						</p>
					{/if}

					<button
						type="submit"
						class="w-full rounded-full bg-teal py-3.5 text-sm font-semibold text-card transition-[background-color,transform] hover:bg-ink active:scale-[0.98]"
					>
						{editingId ? 'Perbarui transaksi' : 'Simpan transaksi'}
					</button>
				</form>
			</div>
		</div>
	</div>
{/if}
