<script lang="ts">
	import { onMount } from 'svelte';
	import { fade, fly } from 'svelte/transition';
	import { accountsRepo, type AccountRow } from '$lib/db';
	import { enqueueCreate, enqueueDelete, syncStatus, triggerManualSync } from '$lib/sync';

	let accounts = $state<AccountRow[]>([]);
	let loading = $state(true);
	let showAddModal = $state(false);

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
		// Re-fetch dari IDB setiap kali engine apply server data atau user mutate.
		void syncStatus.dataVersion;
		void reloadFromLocal();
	});

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
		} catch (err) {
			console.error('Gagal menghapus akun:', err);
		}
	}

	onMount(async () => {
		await reloadFromLocal();
		loading = false;
		// Stale-while-revalidate: kick async sync untuk refresh dari server.
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
</script>

<div class="animate-in fade-in space-y-8 duration-500">
	<!-- Header Section -->
	<div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
		<div>
			<h1 class="text-2xl font-bold text-[#0a2e31]">Rekening Saya</h1>
			<p class="text-sm text-gray-500">Kelola semua sumber dana Anda di sini.</p>
		</div>
		<button
			onclick={() => (showAddModal = true)}
			class="inline-flex items-center justify-center gap-2 rounded-2xl bg-[#217b84] px-6 py-3 font-bold text-white shadow-lg shadow-teal-900/10 transition-all hover:bg-[#1a5f66] active:scale-95"
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

	<!-- Stats Overview -->
	<div class="grid grid-cols-1 gap-6 md:grid-cols-3">
		<div class="space-y-2 rounded-3xl border border-gray-100 bg-white p-6 shadow-sm">
			<span class="text-[11px] font-bold tracking-wider text-gray-400 uppercase">Total Saldo</span>
			<div class="text-2xl font-black text-[#0a2e31]">
				{formatCurrency(accounts.reduce((acc, curr) => acc + curr.balance, 0))}
			</div>
		</div>
		<div class="space-y-2 rounded-3xl border border-gray-100 bg-white p-6 shadow-sm">
			<span class="text-[11px] font-bold tracking-wider text-gray-400 uppercase">Jumlah Akun</span>
			<div class="text-2xl font-black text-[#0a2e31]">{accounts.length} Akun</div>
		</div>
		<div class="space-y-2 rounded-3xl border border-gray-100 bg-white p-6 shadow-sm">
			<span class="text-[11px] font-bold tracking-wider text-gray-400 uppercase">Mata Uang</span>
			<div class="text-2xl font-black text-[#0a2e31]">IDR (Rp)</div>
		</div>
	</div>

	<!-- Accounts Grid -->
	{#if loading}
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
			{#each [0, 1, 2] as i (i)}
				<div class="h-48 animate-pulse rounded-3xl border border-gray-100 bg-gray-50"></div>
			{/each}
		</div>
	{:else if accounts.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-3xl border-2 border-dashed border-gray-200 bg-gray-50/50 py-20"
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
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
			{#each accounts as acc (acc.id)}
				<div
					class="group relative overflow-hidden rounded-[2rem] border border-gray-100 bg-white p-7 shadow-sm transition-all duration-300 hover:shadow-xl hover:shadow-teal-900/5 active:scale-[0.98]"
				>
					<div class="absolute top-0 right-0 p-6">
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

					<div class="space-y-4">
						<div>
							<h3 class="text-lg font-bold text-[#0a2e31]">{acc.name}</h3>
							<span class="text-[10px] font-bold tracking-widest text-gray-400 uppercase"
								>{acc.account_type}</span
							>
						</div>
						<div class="pt-4">
							<div class="mb-1 text-[11px] font-bold text-gray-400 uppercase">Saldo Saat Ini</div>
							<div class="text-2xl font-black tracking-tight text-[#0a2e31]">
								{formatCurrency(acc.balance)}
							</div>
						</div>
					</div>

					<div class="mt-6 flex gap-2">
						<button
							class="flex-1 rounded-xl bg-gray-50 py-2 text-[11px] font-bold tracking-wider text-gray-500 uppercase transition-colors hover:bg-gray-100"
							>Detail</button
						>
						<button
							onclick={() => handleDeleteAccount(acc.id)}
							class="px-3 py-2 text-gray-400 transition-colors hover:text-red-500"
							aria-label="Hapus rekening"
						>
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"
								><path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
								/></svg
							>
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Modal Tambah Akun -->
{#if showAddModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-[#0a2e31]/40 p-4 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
	>
		<div
			class="w-full max-w-md overflow-hidden rounded-[2.5rem] bg-white shadow-2xl"
			in:fly={{ y: 20, duration: 400 }}
		>
			<div class="space-y-6 p-8">
				<div class="flex items-center justify-between">
					<h2 class="text-2xl font-bold text-[#0a2e31]">Tambah Rekening</h2>
					<button
						onclick={() => (showAddModal = false)}
						class="text-gray-400 hover:text-gray-600"
						aria-label="Tutup modal tambah rekening"
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

				<form onsubmit={handleAddAccount} class="space-y-5">
					<div class="space-y-4">
						<div>
							<label
								for="name"
								class="mb-2 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase"
								>Nama Rekening</label
							>
							<input
								id="name"
								type="text"
								required
								bind:value={newAccount.name}
								placeholder="Contoh: BCA Utama, Dompet Jajan"
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
							/>
						</div>

						<div>
							<span
								class="mb-2 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase"
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
										<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"
											><path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d={type.icon}
											/></svg
										>
										<span class="text-[10px] font-bold uppercase">{type.label}</span>
									</button>
								{/each}
							</div>
						</div>

						<div>
							<label
								for="balance"
								class="mb-2 block px-1 text-xs font-bold tracking-wider text-[#0a2e31] uppercase"
								>Saldo Awal</label
							>
							<div class="relative">
								<span
									class="absolute inset-y-0 left-5 flex items-center text-sm font-bold text-gray-400"
									>Rp</span
								>
								<input
									id="balance"
									type="number"
									required
									bind:value={newAccount.balance}
									class="w-full rounded-2xl border border-gray-100 bg-gray-50 py-3.5 pr-5 pl-12 transition-all outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
								/>
							</div>
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
