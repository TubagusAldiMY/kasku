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

	// ── Editorial presentation helpers (additive, no logic change) ──
	// Cycling top-accent border for account cards (mockup: ink / teal / gold).
	const ACCENT_BORDERS = ['border-ink', 'border-teal', 'border-gold'];
</script>

<div class="pb-4">
	<!-- ═══════════ Header ═══════════ -->
	<div
		class="flex flex-col justify-between gap-5 border-b border-ink/10 pb-8 sm:flex-row sm:items-end"
	>
		<div>
			<h1 class="font-serif text-4xl tracking-tight text-ink">Rekening</h1>
			<p class="mt-2.5 text-sm text-ink/60">
				Total saldo
				<span class="font-semibold text-ink"
					>{formatCurrency(accounts.reduce((s, a) => s + a.balance, 0))}</span
				>
				di {accounts.length} rekening.
			</p>
		</div>
		<button
			onclick={() => (showAddModal = true)}
			class="shrink-0 rounded-full bg-teal px-5 py-2.5 text-[13px] font-semibold text-card transition-colors hover:bg-ink"
		>
			+ Tambah rekening
		</button>
	</div>

	<!-- Accounts Grid -->
	{#if loading}
		<div class="grid grid-cols-1 gap-7 pt-10 sm:grid-cols-2 lg:grid-cols-3">
			{#each [0, 1, 2] as i (i)}
				<div class="h-40 animate-pulse border-t-2 border-ink/10 bg-ink/5"></div>
			{/each}
		</div>
	{:else if accounts.length === 0}
		<div class="flex flex-col items-center gap-3 py-20 text-center">
			<p class="font-serif text-2xl text-ink">Belum ada rekening</p>
			<p class="text-sm text-ink/45">Mulai dengan menambahkan rekening pertama Anda.</p>
			<button
				onclick={() => (showAddModal = true)}
				class="mt-1 text-sm font-semibold text-teal transition-colors hover:text-ink"
				>Tambah sekarang →</button
			>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-x-7 gap-y-9 pt-10 sm:grid-cols-2 lg:grid-cols-3">
			{#each accounts as acc, i (acc.id)}
				<div class="group border-t-2 pt-5 {ACCENT_BORDERS[i % ACCENT_BORDERS.length]}">
					<div class="flex items-baseline justify-between">
						<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
							{acc.account_type}
						</p>
						<button
							onclick={() => handleDeleteAccount(acc.id)}
							class="text-[11.5px] font-semibold text-ink/40 transition-colors hover:text-clay"
							aria-label="Hapus rekening"
						>
							Hapus
						</button>
					</div>
					<p class="mt-3 font-serif text-[22px] leading-tight text-ink">{acc.name}</p>
					<p class="mt-1 font-serif text-[34px] leading-none tracking-tight text-ink tabular-nums">
						{formatCurrency(acc.balance)}
					</p>
					<button
						onclick={() => openDetail(acc)}
						class="mt-4 text-[13px] font-semibold text-teal transition-colors hover:text-ink"
					>
						Lihat detail →
					</button>
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
		class="fixed inset-0 z-50 bg-ink/25 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
		onclick={closeDetail}
	></div>

	<!-- Drawer — bottom sheet di mobile, side panel di lg -->
	<div
		class="fixed inset-x-0 bottom-0 z-50 flex max-h-[88dvh] flex-col rounded-t-2xl border-t border-ink/10 bg-card shadow-xl lg:inset-y-0 lg:right-0 lg:left-auto lg:max-h-none lg:w-[420px] lg:rounded-none lg:border-t-0 lg:border-l"
		in:fly={{ y: 60, duration: 350 }}
	>
		<!-- Handle (mobile only) -->
		<div class="flex shrink-0 justify-center pt-3 pb-1 lg:hidden">
			<div class="h-1 w-10 rounded-full bg-ink/15"></div>
		</div>

		<!-- Header drawer -->
		<div class="shrink-0 border-b border-ink/10 px-6 py-6">
			<div class="flex items-start justify-between gap-3">
				<div class="min-w-0">
					<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
						{selectedAccount.account_type}
					</p>
					<h2 class="mt-1 truncate font-serif text-2xl text-ink">{selectedAccount.name}</h2>
				</div>
				<button
					onclick={closeDetail}
					class="shrink-0 rounded-full p-2 text-ink/40 transition-colors hover:bg-ink/5 hover:text-ink"
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
					<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
						Saldo saat ini
					</p>
					<p class="mt-1 font-serif text-3xl text-ink tabular-nums">
						{formatCurrency(selectedAccount.balance)}
					</p>
				</div>
				<button
					onclick={() => handleDeleteAccount(selectedAccount!.id)}
					class="rounded-full border border-ink/15 px-3.5 py-2 text-[12px] font-semibold text-clay transition-colors hover:border-clay/30 hover:bg-clay/5"
				>
					Hapus
				</button>
			</div>
		</div>

		<!-- Riwayat transaksi -->
		<div class="min-h-0 flex-1 overflow-y-auto">
			<div class="px-6 pt-5 pb-2">
				<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
					Riwayat transaksi
				</p>
			</div>

			{#if detailLoading}
				<div class="space-y-3 px-6 pb-6">
					{#each [0, 1, 2, 3] as i (i)}
						<div class="h-10 animate-pulse rounded bg-ink/5"></div>
					{/each}
				</div>
			{:else if selectedAccount.transactions.length === 0}
				<p class="px-6 py-10 text-center text-sm text-ink/45">Belum ada transaksi</p>
			{:else}
				<div class="px-6 pb-8">
					{#each selectedAccount.transactions as tx (tx.id)}
						<div class="flex items-baseline justify-between gap-3 border-b border-ink/8 py-3.5">
							<div class="min-w-0 flex-1">
								<p class="truncate text-sm font-medium text-ink">{tx.title}</p>
								<div class="mt-0.5 flex items-center gap-1.5">
									<span class="text-[11px] text-ink/45">{tx.category}</span>
									{#if tx.toAccount && tx.type === 'TRANSFER'}
										<span class="text-[11px] text-steel">→ {tx.toAccount}</span>
									{/if}
									<span class="text-[11px] text-ink/35">· {formatDateShort(tx.date)}</span>
								</div>
							</div>
							<span
								class="shrink-0 text-sm tabular-nums {tx.amount > 0
									? 'font-semibold text-teal'
									: tx.type === 'TRANSFER'
										? 'text-steel'
										: 'text-ink'}"
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
		class="fixed inset-0 z-50 flex items-end justify-center bg-ink/25 backdrop-blur-sm sm:items-center sm:p-4"
		in:fade={{ duration: 200 }}
	>
		<div
			class="max-h-[92dvh] w-full overflow-y-auto rounded-t-2xl border border-ink/10 bg-card shadow-xl sm:max-w-md sm:rounded-2xl"
			in:fly={{ y: 40, duration: 350 }}
		>
			<!-- Handle (mobile) -->
			<div class="flex justify-center pt-3 pb-1 sm:hidden">
				<div class="h-1 w-10 rounded-full bg-ink/15"></div>
			</div>

			<div class="space-y-5 p-5 sm:p-8">
				<div class="flex items-center justify-between">
					<h2 class="font-serif text-2xl text-ink">Tambah rekening</h2>
					<button
						onclick={() => (showAddModal = false)}
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

				<form onsubmit={handleAddAccount} class="space-y-4">
					<div>
						<label
							for="name"
							class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
							>Nama rekening</label
						>
						<input
							id="name"
							type="text"
							required
							bind:value={newAccount.name}
							placeholder="Contoh: BCA Utama, Dompet Jajan"
							class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink outline-none focus:border-teal"
						/>
					</div>

					<div>
						<span
							class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
							>Tipe akun</span
						>
						<div class="grid grid-cols-3 gap-2">
							{#each accountTypes as type (type.id)}
								<button
									type="button"
									onclick={() => (newAccount.account_type = type.id)}
									class="flex flex-col items-center gap-2 rounded-[10px] border p-3 transition-colors {newAccount.account_type ===
									type.id
										? 'border-teal bg-teal/5 text-teal'
										: 'border-ink/15 text-ink/50 hover:border-ink/30'}"
								>
									<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="1.8"
											d={type.icon}
										/>
									</svg>
									<span class="text-[10px] font-semibold tracking-wide uppercase">{type.label}</span
									>
								</button>
							{/each}
						</div>
					</div>

					<div>
						<label
							for="balance"
							class="mb-1.5 block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
							>Saldo awal</label
						>
						<div class="relative">
							<span class="absolute inset-y-0 left-4 flex items-center text-sm text-ink/45">Rp</span
							>
							<input
								id="balance"
								type="number"
								required
								bind:value={newAccount.balance}
								class="w-full rounded-[10px] border border-ink/25 bg-field py-3 pr-4 pl-11 text-sm text-ink tabular-nums outline-none focus:border-teal"
							/>
						</div>
					</div>

					<button
						type="submit"
						class="w-full rounded-full bg-teal py-3.5 text-sm font-semibold text-card transition-colors hover:bg-ink"
					>
						Simpan rekening
					</button>
				</form>
			</div>
		</div>
	</div>
{/if}
