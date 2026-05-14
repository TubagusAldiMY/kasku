<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { fade, fly } from 'svelte/transition';

	// State untuk data akun
	let accounts = $state<any[]>([]);
	let loading = $state(true);
	let showAddModal = $state(false);

	// State untuk form tambah akun
	let newAccount = $state({
		name: '',
		account_type: 'BANK',
		balance: 0,
		currency: 'IDR',
		color: '#217b84'
	});

	async function fetchAccounts() {
		loading = true;
		try {
			const res = await apiFetch('/accounts'); // Endpoint finance-service via gateway
			const result = await res.json();
			if (result.success) {
				accounts = result.data || [];
			}
		} catch (err) {
			console.error('Gagal mengambil data akun:', err);
		} finally {
			loading = false;
		}
	}

	async function handleAddAccount(e: SubmitEvent) {
		e.preventDefault();
		try {
			const res = await apiFetch('/accounts', {
				method: 'POST',
				body: JSON.stringify(newAccount)
			});
			const result = await res.json();
			if (result.success) {
				showAddModal = false;
				fetchAccounts(); // Refresh daftar
				// Reset form
				newAccount = { name: '', account_type: 'BANK', balance: 0, currency: 'IDR', color: '#217b84' };
			}
		} catch (err) {
			console.error('Gagal menambah akun:', err);
		}
	}

	onMount(fetchAccounts);

	const accountTypes = [
		{ id: 'BANK', label: 'Bank', icon: 'M3 10h18M7 10V7a5 5 0 0110 0v3M4 10v10a1 1 0 001 1h14a1 1 0 001-1V10M10 14v4M14 14v4' },
		{ id: 'EWALLET', label: 'E-Wallet', icon: 'M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z' },
		{ id: 'CASH', label: 'Tunai', icon: 'M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z' }
	];

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(val);
	}
</script>

<div class="space-y-8 animate-in fade-in duration-500">
	<!-- Header Section -->
	<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
		<div>
			<h1 class="text-2xl font-bold text-[#0a2e31]">Rekening Saya</h1>
			<p class="text-sm text-gray-500">Kelola semua sumber dana Anda di sini.</p>
		</div>
		<button 
			onclick={() => showAddModal = true}
			class="inline-flex items-center justify-center gap-2 px-6 py-3 bg-[#217b84] hover:bg-[#1a5f66] text-white font-bold rounded-2xl shadow-lg shadow-teal-900/10 transition-all active:scale-95"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4" />
			</svg>
			Tambah Rekening
		</button>
	</div>

	<!-- Stats Overview -->
	<div class="grid grid-cols-1 md:grid-cols-3 gap-6">
		<div class="bg-white p-6 rounded-3xl border border-gray-100 shadow-sm space-y-2">
			<span class="text-[11px] font-bold uppercase tracking-wider text-gray-400">Total Saldo</span>
			<div class="text-2xl font-black text-[#0a2e31]">
				{formatCurrency(accounts.reduce((acc, curr) => acc + curr.balance, 0))}
			</div>
		</div>
		<div class="bg-white p-6 rounded-3xl border border-gray-100 shadow-sm space-y-2">
			<span class="text-[11px] font-bold uppercase tracking-wider text-gray-400">Jumlah Akun</span>
			<div class="text-2xl font-black text-[#0a2e31]">{accounts.length} Akun</div>
		</div>
		<div class="bg-white p-6 rounded-3xl border border-gray-100 shadow-sm space-y-2">
			<span class="text-[11px] font-bold uppercase tracking-wider text-gray-400">Mata Uang</span>
			<div class="text-2xl font-black text-[#0a2e31]">IDR (Rp)</div>
		</div>
	</div>

	<!-- Accounts Grid -->
	{#if loading}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
			{#each Array(3) as _}
				<div class="h-48 bg-gray-50 rounded-3xl animate-pulse border border-gray-100"></div>
			{/each}
		</div>
	{:else if accounts.length === 0}
		<div class="flex flex-col items-center justify-center py-20 bg-gray-50/50 rounded-3xl border-2 border-dashed border-gray-200">
			<div class="h-16 w-16 bg-gray-100 rounded-full flex items-center justify-center mb-4">
				<svg class="h-8 w-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h10a2 2 0 012 2v2M7 7h10" />
				</svg>
			</div>
			<h3 class="text-lg font-bold text-[#0a2e31]">Belum ada rekening</h3>
			<p class="text-sm text-gray-500 mb-6">Mulai dengan menambahkan rekening pertama Anda.</p>
			<button onclick={() => showAddModal = true} class="text-[#217b84] font-bold hover:underline">Tambah Sekarang</button>
		</div>
	{:else}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
			{#each accounts as acc}
				<div 
					class="group relative overflow-hidden bg-white p-7 rounded-[2rem] border border-gray-100 shadow-sm hover:shadow-xl hover:shadow-teal-900/5 transition-all duration-300 active:scale-[0.98]"
				>
					<div class="absolute top-0 right-0 p-6">
						<div class="h-10 w-10 rounded-2xl bg-gray-50 flex items-center justify-center text-gray-400 group-hover:bg-[#217b84] group-hover:text-white transition-colors">
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={accountTypes.find(t => t.id === acc.account_type)?.icon || ''} />
							</svg>
						</div>
					</div>
					
					<div class="space-y-4">
						<div>
							<h3 class="text-lg font-bold text-[#0a2e31]">{acc.name}</h3>
							<span class="text-[10px] font-bold uppercase tracking-widest text-gray-400">{acc.account_type}</span>
						</div>
						<div class="pt-4">
							<div class="text-[11px] font-bold text-gray-400 uppercase mb-1">Saldo Saat Ini</div>
							<div class="text-2xl font-black text-[#0a2e31] tracking-tight">
								{formatCurrency(acc.balance)}
							</div>
						</div>
					</div>

					<div class="mt-6 flex gap-2">
						<button class="flex-1 py-2 text-[11px] font-bold uppercase tracking-wider text-gray-500 bg-gray-50 rounded-xl hover:bg-gray-100 transition-colors">Detail</button>
						<button class="px-3 py-2 text-gray-400 hover:text-red-500 transition-colors" aria-label="Hapus rekening">
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Modal Tambah Akun -->
{#if showAddModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-[#0a2e31]/40 backdrop-blur-sm" in:fade={{ duration: 200 }}>
		<div 
			class="bg-white w-full max-w-md rounded-[2.5rem] shadow-2xl overflow-hidden"
			in:fly={{ y: 20, duration: 400 }}
		>
			<div class="p-8 space-y-6">
				<div class="flex justify-between items-center">
					<h2 class="text-2xl font-bold text-[#0a2e31]">Tambah Rekening</h2>
					<button onclick={() => showAddModal = false} class="text-gray-400 hover:text-gray-600" aria-label="Tutup modal tambah rekening">
						<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
					</button>
				</div>

				<form onsubmit={handleAddAccount} class="space-y-5">
					<div class="space-y-4">
						<div>
							<label for="name" class="block text-xs font-bold text-[#0a2e31] uppercase tracking-wider mb-2 px-1">Nama Rekening</label>
							<input 
								id="name"
								type="text" 
								required 
								bind:value={newAccount.name}
								placeholder="Contoh: BCA Utama, Dompet Jajan"
								class="w-full px-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 focus:border-[#217b84] outline-none transition-all"
							/>
						</div>

						<div>
							<span class="block text-xs font-bold text-[#0a2e31] uppercase tracking-wider mb-2 px-1">Tipe Akun</span>
							<div class="grid grid-cols-3 gap-2">
								{#each accountTypes as type}
									<button 
										type="button"
										onclick={() => newAccount.account_type = type.id}
										class="flex flex-col items-center gap-2 p-3 rounded-2xl border-2 transition-all {newAccount.account_type === type.id ? 'border-[#217b84] bg-teal-50 text-[#217b84]' : 'border-gray-50 text-gray-400'}"
									>
										<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={type.icon} /></svg>
										<span class="text-[10px] font-bold uppercase">{type.label}</span>
									</button>
								{/each}
							</div>
						</div>

						<div>
							<label for="balance" class="block text-xs font-bold text-[#0a2e31] uppercase tracking-wider mb-2 px-1">Saldo Awal</label>
							<div class="relative">
								<span class="absolute inset-y-0 left-5 flex items-center text-gray-400 font-bold text-sm">Rp</span>
								<input 
									id="balance"
									type="number" 
									required 
									bind:value={newAccount.balance}
									class="w-full pl-12 pr-5 py-3.5 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 focus:border-[#217b84] outline-none transition-all"
								/>
							</div>
						</div>
					</div>

					<button 
						type="submit"
						class="w-full py-4 bg-[#217b84] hover:bg-[#1a5f66] text-white font-bold rounded-2xl shadow-xl transition-all active:scale-[0.98]"
					>
						Simpan Rekening
					</button>
				</form>
			</div>
		</div>
	</div>
{/if}
