<script lang="ts">
	import { onMount } from 'svelte';
	import { fade, fly } from 'svelte/transition';
	import {
		accountsRepo,
		transactionsRepo,
		categoriesRepo,
		type AccountRow,
		type CategoryRow
	} from '$lib/db';
	import { enqueueCreate, enqueueDelete, syncStatus, triggerManualSync } from '$lib/sync';

	type AccountDetail = AccountRow & {
		transactions: {
			id: string;
			date: string;
			title: string;
			category: string;
			amount: number;
			type: 'INCOME' | 'EXPENSE' | 'TRANSFER';
			toAccount: string;
		}[];
	};

	let accounts = $state<AccountRow[]>([]);
	let loading = $state(true);
	let showAddModal = $state(false);

	// Detail drawer state
	let selectedAccount = $state<AccountDetail | null>(null);
	let detailLoading = $state(false);

	let newAccount = $state({
		name: '',
		account_type: 'BANK',
		balance: 0,
		currency: 'IDR',
		color: '#217b84'
	});

	async function reloadFromLocal() {
		try {
			accounts = await accountsRepo.getAll();
		} catch (err) {
			console.error('Gagal membaca akun dari penyimpanan lokal:', err);
		}
	}

	$effect(() => {
		void syncStatus.dataVersion;
		void reloadFromLocal();
	});

	async function openDetail(acc: AccountRow) {
		detailLoading = true;
		selectedAccount = { ...acc, transactions: [] };
		try {
			const [allTx, allCat, allAcc] = await Promise.all([
				transactionsRepo.getAll(),
				categoriesRepo.getAll(),
				accountsRepo.getAll()
			]);
			const catMap = new Map<string, CategoryRow>(allCat.map((c) => [c.id, c]));
			const accMap = new Map<string, AccountRow>(allAcc.map((a) => [a.id, a]));

			const txForAcc = allTx
				.filter((t) => t.account_id === acc.id || t.to_account_id === acc.id)
				.sort((a, b) => (a.transaction_date < b.transaction_date ? 1 : -1))
				.map((t) => {
					const isSource = t.account_id === acc.id;
					const signed =
						t.transaction_type === 'INCOME'
							? t.amount_idr
							: t.transaction_type === 'TRANSFER'
								? isSource
									? -t.amount_idr
									: t.amount_idr
								: -t.amount_idr;
					const toAcc = t.to_account_id ? (accMap.get(t.to_account_id)?.name ?? '') : '';
					return {
						id: t.id,
						date: t.transaction_date,
						title: t.notes ?? t.transaction_type,
						category:
							catMap.get(t.category_id ?? '')?.name ??
							(t.transaction_type === 'TRANSFER' ? 'Transfer' : 'Umum'),
						amount: signed,
						type: t.transaction_type,
						toAccount: toAcc
					};
				});

			selectedAccount = { ...acc, transactions: txForAcc };
		} catch (err) {
			console.error('Gagal memuat detail rekening:', err);
		} finally {
			detailLoading = false;
		}
	}

	function closeDetail() {
		selectedAccount = null;
	}

	async function handleAddAccount(e: SubmitEvent) {
		e.preventDefault();
		try {
			await enqueueCreate<AccountRow>('accounts', {
				name: newAccount.name,
				account_type: newAccount.account_type,
				balance: newAccount.balance,
				currency: newAccount.currency,
				color: newAccount.color
			});
			showAddModal = false;
			newAccount = {
				name: '',
				account_type: 'BANK',
				balance: 0,
				currency: 'IDR',
				color: '#217b84'
			};
		} catch (err) {
			console.error('Gagal menambah akun:', err);
		}
	}

	async function handleDeleteAccount(id: string) {
		if (
			!confirm(
				'Apakah Anda yakin ingin menghapus rekening ini? Seluruh riwayat transaksi terkait mungkin terpengaruh.'
			)
		)
			return;
		try {
			await enqueueDelete('accounts', id);
			if (selectedAccount?.id === id) closeDetail();
		} catch (err) {
			console.error('Gagal menghapus akun:', err);
		}
	}

	onMount(async () => {
		await reloadFromLocal();
		loading = false;
		void triggerManualSync();
	});

	const accountTypes = [
		{
			id: 'BANK',
			label: 'Bank',
			icon: 'M3 10h18M7 10V7a5 5 0 0110 0v3M4 10v10a1 1 0 001 1h14a1 1 0 001-1V10M10 14v4M14 14v4'
		},
		{
			id: 'EWALLET',
			label: 'E-Wallet',
			icon: 'M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z'
		},
		{
			id: 'CASH',
			label: 'Tunai',
			icon: 'M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z'
		}
	];

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(val);
	}

	function formatDateShort(dateStr: string) {
		return new Date(dateStr).toLocaleDateString('id-ID', {
			day: 'numeric',
			month: 'short',
			year: 'numeric'
		});
	}
</script>

<div class="animate-in fade-in space-y-6 duration-500">
	<!-- Header -->
	<div class="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
		<div>
			<h1 class="text-xl font-bold text-[#0a2e31] sm:text-2xl">Rekening Saya</h1>
			<p class="text-sm text-gray-500">Kelola semua sumber dana Anda di sini.</p>
		</div>
		<button
			onclick={() => (showAddModal = true)}
			class="inline-flex items-center justify-center gap-2 rounded-2xl bg-[#217b84] px-5 py-3 font-bold text-white shadow-lg transition-all hover:bg-[#1a5f66] active:scale-95"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2.5"
					d="M12 4v16m8-8H4"
				/>
			</svg>
			Tambah Rekening
		</button>
	</div>

	<!-- Stats -->
	<div class="grid grid-cols-3 gap-3 sm:gap-6">
		<div
			class="space-y-1 rounded-2xl border border-gray-100 bg-white p-4 shadow-sm sm:rounded-3xl sm:p-6"
		>
			<span class="text-[10px] font-bold tracking-wider text-gray-400 uppercase">Total Saldo</span>
			<div class="text-base font-black text-[#0a2e31] sm:text-2xl">
				{formatCurrency(accounts.reduce((s, a) => s + a.balance, 0))}
			</div>
		</div>
		<div
			class="space-y-1 rounded-2xl border border-gray-100 bg-white p-4 shadow-sm sm:rounded-3xl sm:p-6"
		>
			<span class="text-[10px] font-bold tracking-wider text-gray-400 uppercase">Jumlah Akun</span>
			<div class="text-base font-black text-[#0a2e31] sm:text-2xl">{accounts.length} Akun</div>
		</div>
		<div
			class="space-y-1 rounded-2xl border border-gray-100 bg-white p-4 shadow-sm sm:rounded-3xl sm:p-6"
		>
			<span class="text-[10px] font-bold tracking-wider text-gray-400 uppercase">Mata Uang</span>
			<div class="text-base font-black text-[#0a2e31] sm:text-2xl">IDR</div>
		</div>
	</div>

	<!-- Accounts Grid -->
	{#if loading}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each [0, 1, 2] as i (i)}
				<div class="h-44 animate-pulse rounded-3xl border border-gray-100 bg-gray-50"></div>
			{/each}
		</div>
	{:else if accounts.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-3xl border-2 border-dashed border-gray-200 bg-gray-50/50 py-16"
		>
			<div class="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-gray-100">
				<svg class="h-8 w-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h10a2 2 0 012 2v2M7 7h10"
					/>
				</svg>
			</div>
			<h3 class="text-lg font-bold text-[#0a2e31]">Belum ada rekening</h3>
			<p class="mb-6 text-sm text-gray-500">Mulai dengan menambahkan rekening pertama Anda.</p>
			<button onclick={() => (showAddModal = true)} class="font-bold text-[#217b84] hover:underline"
				>Tambah Sekarang</button
			>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each accounts as acc (acc.id)}
				<div
					class="group relative overflow-hidden rounded-[2rem] border border-gray-100 bg-white p-6 shadow-sm transition-all duration-300 hover:shadow-xl hover:shadow-teal-900/5"
				>
					<div class="absolute top-0 right-0 p-5">
						<div
							class="flex h-10 w-10 items-center justify-center rounded-2xl bg-gray-50 text-gray-400 transition-colors group-hover:bg-[#217b84] group-hover:text-white"
						>
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d={accountTypes.find((t) => t.id === acc.account_type)?.icon || ''}
								/>
							</svg>
						</div>
					</div>

					<div class="space-y-3">
						<div>
							<h3 class="pr-12 text-base font-bold text-[#0a2e31] sm:text-lg">{acc.name}</h3>
							<span class="text-[10px] font-bold tracking-widest text-gray-400 uppercase"
								>{acc.account_type}</span
							>
						</div>
						<div class="pt-2">
							<div class="mb-1 text-[11px] font-bold text-gray-400 uppercase">Saldo Saat Ini</div>
							<div class="text-xl font-black tracking-tight text-[#0a2e31] sm:text-2xl">
								{formatCurrency(acc.balance)}
							</div>
						</div>
					</div>

					<div class="mt-5 flex gap-2">
						<button
							onclick={() => openDetail(acc)}
							class="flex-1 rounded-xl bg-[#0a2e31] py-2.5 text-[11px] font-bold tracking-wider text-white uppercase transition-colors hover:bg-[#217b84] active:scale-95"
						>
							Detail
						</button>
						<button
							onclick={() => handleDeleteAccount(acc.id)}
							class="rounded-xl px-3 py-2.5 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-500 active:scale-95"
							aria-label="Hapus rekening"
						>
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
			{/each}
		</div>
	{/if}
</div>

<!-- ============================================
     Detail Drawer
     ============================================ -->
{#if selectedAccount !== null}
	<!-- Overlay -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 bg-[#0a2e31]/30 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
		onclick={closeDetail}
	></div>

	<!-- Drawer — bottom sheet di mobile, side panel di lg -->
	<div
		class="fixed inset-x-0 bottom-0 z-50 flex max-h-[88dvh] flex-col rounded-t-[2rem] bg-white shadow-2xl lg:inset-y-0 lg:right-0 lg:left-auto lg:max-h-none lg:w-[420px] lg:rounded-none lg:rounded-l-[2rem]"
		in:fly={{ y: 60, duration: 350 }}
	>
		<!-- Handle (mobile only) -->
		<div class="flex shrink-0 justify-center pt-3 pb-1 lg:hidden">
			<div class="h-1 w-10 rounded-full bg-gray-200"></div>
		</div>

		<!-- Header drawer -->
		<div class="shrink-0 border-b border-gray-100 px-6 py-5">
			<div class="flex items-start justify-between gap-3">
				<div class="min-w-0">
					<p class="text-[10px] font-bold tracking-widest text-gray-400 uppercase">
						{selectedAccount.account_type}
					</p>
					<h2 class="truncate text-lg font-bold text-[#0a2e31]">{selectedAccount.name}</h2>
				</div>
				<button
					onclick={closeDetail}
					class="shrink-0 rounded-xl p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
					aria-label="Tutup detail"
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

			<!-- Saldo + aksi -->
			<div class="mt-4 flex items-end justify-between">
				<div>
					<p class="text-[10px] font-bold tracking-widest text-gray-400 uppercase">
						Saldo Saat Ini
					</p>
					<p class="text-2xl font-black text-[#0a2e31]">
						{formatCurrency(selectedAccount.balance)}
					</p>
				</div>
				<button
					onclick={() => handleDeleteAccount(selectedAccount!.id)}
					class="flex items-center gap-1.5 rounded-xl border border-red-100 px-3 py-2 text-[11px] font-bold text-red-400 transition-colors hover:bg-red-50 hover:text-red-600"
				>
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
						/>
					</svg>
					Hapus
				</button>
			</div>
		</div>

		<!-- Riwayat transaksi -->
		<div class="min-h-0 flex-1 overflow-y-auto">
			<div class="px-6 pt-5 pb-3">
				<p class="text-[10px] font-bold tracking-widest text-gray-400 uppercase">
					Riwayat Transaksi
				</p>
			</div>

			{#if detailLoading}
				<div class="space-y-3 px-6 pb-6">
					{#each [0, 1, 2, 3] as i (i)}
						<div class="h-16 animate-pulse rounded-2xl bg-gray-50"></div>
					{/each}
				</div>
			{:else if selectedAccount.transactions.length === 0}
				<div class="flex flex-col items-center gap-3 py-12 text-center">
					<div
						class="flex h-12 w-12 items-center justify-center rounded-full bg-gray-50 text-gray-300"
					>
						<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
							/>
						</svg>
					</div>
					<p class="text-sm font-bold text-gray-400">Belum ada transaksi</p>
				</div>
			{:else}
				<div class="space-y-2 px-4 pb-8">
					{#each selectedAccount.transactions as tx (tx.id)}
						<div class="flex items-center gap-3 rounded-2xl p-3 hover:bg-gray-50">
							<!-- Ikon tipe -->
							<div
								class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full {tx.type ===
								'INCOME'
									? 'bg-green-50'
									: tx.type === 'TRANSFER'
										? 'bg-blue-50'
										: 'bg-red-50'}"
							>
								{#if tx.type === 'INCOME'}
									<svg
										class="h-4 w-4 text-green-500"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2.5"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M5 10l7-7m0 0l7 7m-7-7v18"
										/>
									</svg>
								{:else if tx.type === 'TRANSFER'}
									<svg
										class="h-4 w-4 text-blue-500"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2.5"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
										/>
									</svg>
								{:else}
									<svg
										class="h-4 w-4 text-red-400"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2.5"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M19 14l-7 7m0 0l-7-7m7 7V3"
										/>
									</svg>
								{/if}
							</div>

							<!-- Info -->
							<div class="min-w-0 flex-1">
								<p class="truncate text-sm font-bold text-[#0a2e31]">{tx.title}</p>
								<div class="flex items-center gap-1.5">
									<span class="text-[10px] text-gray-400">{tx.category}</span>
									{#if tx.toAccount && tx.type === 'TRANSFER'}
										<span class="text-[10px] text-gray-300">→ {tx.toAccount}</span>
									{/if}
								</div>
								<p class="text-[10px] text-gray-400">{formatDateShort(tx.date)}</p>
							</div>

							<!-- Nominal -->
							<span
								class="shrink-0 text-sm font-black {tx.amount > 0
									? 'text-green-600'
									: tx.type === 'TRANSFER'
										? 'text-blue-500'
										: 'text-red-500'}"
							>
								{tx.amount > 0 ? '+' : ''}{formatCurrency(tx.amount)}
							</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</div>
{/if}

<!-- ============================================
     Modal Tambah Rekening
     ============================================ -->
{#if showAddModal}
	<div
		class="fixed inset-0 z-50 flex items-end justify-center bg-[#0a2e31]/40 backdrop-blur-sm sm:items-center sm:p-4"
		in:fade={{ duration: 200 }}
	>
		<div
			class="max-h-[92dvh] w-full overflow-y-auto rounded-t-[2rem] bg-white shadow-2xl sm:max-w-md sm:rounded-[2.5rem]"
			in:fly={{ y: 40, duration: 350 }}
		>
			<!-- Handle (mobile) -->
			<div class="flex justify-center pt-3 pb-1 sm:hidden">
				<div class="h-1 w-10 rounded-full bg-gray-200"></div>
			</div>

			<div class="space-y-5 p-5 sm:p-8">
				<div class="flex items-center justify-between">
					<h2 class="text-xl font-bold text-[#0a2e31] sm:text-2xl">Tambah Rekening</h2>
					<button
						onclick={() => (showAddModal = false)}
						class="rounded-xl p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
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

				<form onsubmit={handleAddAccount} class="space-y-4">
					<div>
						<label
							for="name"
							class="mb-1.5 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase"
							>Nama Rekening</label
						>
						<input
							id="name"
							type="text"
							required
							bind:value={newAccount.name}
							placeholder="Contoh: BCA Utama, Dompet Jajan"
							class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
						/>
					</div>

					<div>
						<span
							class="mb-1.5 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase"
							>Tipe Akun</span
						>
						<div class="grid grid-cols-3 gap-2">
							{#each accountTypes as type (type.id)}
								<button
									type="button"
									onclick={() => (newAccount.account_type = type.id)}
									class="flex flex-col items-center gap-2 rounded-2xl border-2 p-3 transition-all {newAccount.account_type ===
									type.id
										? 'border-[#217b84] bg-teal-50 text-[#217b84]'
										: 'border-gray-50 text-gray-400'}"
								>
									<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d={type.icon}
										/>
									</svg>
									<span class="text-[10px] font-bold uppercase">{type.label}</span>
								</button>
							{/each}
						</div>
					</div>

					<div>
						<label
							for="balance"
							class="mb-1.5 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase"
							>Saldo Awal</label
						>
						<div class="relative">
							<span
								class="absolute inset-y-0 left-4 flex items-center text-sm font-bold text-gray-400"
								>Rp</span
							>
							<input
								id="balance"
								type="number"
								required
								bind:value={newAccount.balance}
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 py-3 pr-4 pl-11 text-sm transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
							/>
						</div>
					</div>

					<button
						type="submit"
						class="w-full rounded-2xl bg-[#217b84] py-4 font-bold text-white shadow-xl transition-all hover:bg-[#1a5f66] active:scale-[0.98]"
					>
						Simpan Rekening
					</button>
				</form>
			</div>
		</div>
	</div>
{/if}
