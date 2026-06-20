<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { fade, fly } from 'svelte/transition';

	type Direction = 'RECEIVABLE' | 'PAYABLE';
	type Status = 'ACTIVE' | 'SETTLED';

	type Debt = {
		ID: string;
		Direction: Direction;
		PersonName: string;
		TotalAmount: number;
		RemainingAmount: number;
		DueDate?: string | null;
		Notes?: string;
		Status: Status;
		CreatedAt: string;
	};

	type Payment = {
		ID: string;
		DebtID: string;
		Amount: number;
		PaymentDate: string;
		Notes?: string;
		CreatedAt: string;
	};

	let debts = $state<Debt[]>([]);
	let loading = $state(true);
	let activeTab = $state<Direction>('RECEIVABLE');
	let showDebtModal = $state(false);
	let showPaymentModal = $state(false);
	let selectedDebt = $state<Debt | null>(null);
	let payments = $state<Payment[]>([]);
	let paymentsLoading = $state(false);
	let error = $state<string | null>(null);

	// Form tambah/edit hutang
	let debtForm = $state({
		id: '',
		direction: 'RECEIVABLE' as Direction,
		person_name: '',
		amount: 0,
		due_date: '',
		notes: ''
	});

	// Form catat pembayaran
	let payForm = $state({
		amount: 0,
		payment_date: new Date().toISOString().split('T')[0],
		notes: ''
	});

	async function fetchDebts() {
		loading = true;
		error = null;
		try {
			const res = await apiFetch('/debts');
			const result = await res.json();
			if (result.success) {
				debts = (result.data ?? []) as Debt[];
			} else {
				error = result.error?.message ?? 'Gagal memuat data.';
			}
		} catch {
			error = 'Gagal menghubungi server.';
		} finally {
			loading = false;
		}
	}

	async function fetchPayments(debtId: string) {
		paymentsLoading = true;
		try {
			const res = await apiFetch(`/debts/${debtId}/payments`);
			const result = await res.json();
			if (result.success) payments = (result.data ?? []) as Payment[];
		} catch {
			payments = [];
		} finally {
			paymentsLoading = false;
		}
	}

	async function handleSaveDebt(e: SubmitEvent) {
		e.preventDefault();
		try {
			const body: Record<string, unknown> = {
				direction: debtForm.direction,
				person_name: debtForm.person_name,
				amount: debtForm.amount,
				notes: debtForm.notes
			};
			if (debtForm.due_date) body.due_date = debtForm.due_date;

			const res = debtForm.id
				? await apiFetch(`/debts/${debtForm.id}`, { method: 'PUT', body: JSON.stringify({ person_name: debtForm.person_name, due_date: debtForm.due_date || null, notes: debtForm.notes }) })
				: await apiFetch('/debts', { method: 'POST', body: JSON.stringify(body) });

			const result = await res.json();
			if (result.success) {
				showDebtModal = false;
				await fetchDebts();
			} else {
				error = result.error?.message ?? 'Gagal menyimpan.';
			}
		} catch {
			error = 'Gagal menghubungi server.';
		}
	}

	async function handleDeleteDebt() {
		if (!selectedDebt || !confirm('Hapus catatan hutang ini beserta riwayat pembayarannya?')) return;
		try {
			const res = await apiFetch(`/debts/${selectedDebt.ID}`, { method: 'DELETE' });
			const result = await res.json();
			if (result.success) {
				showDebtModal = false;
				showPaymentModal = false;
				selectedDebt = null;
				await fetchDebts();
			}
		} catch {
			error = 'Gagal menghapus.';
		}
	}

	async function handleRecordPayment(e: SubmitEvent) {
		e.preventDefault();
		if (!selectedDebt) return;
		try {
			const res = await apiFetch(`/debts/${selectedDebt.ID}/payments`, {
				method: 'POST',
				body: JSON.stringify({
					amount: payForm.amount,
					payment_date: payForm.payment_date,
					notes: payForm.notes
				})
			});
			const result = await res.json();
			if (result.success) {
				payForm = { amount: 0, payment_date: new Date().toISOString().split('T')[0], notes: '' };
				await Promise.all([fetchDebts(), fetchPayments(selectedDebt.ID)]);
				// Update selectedDebt dari list yang baru
				const updated = debts.find((d) => d.ID === selectedDebt!.ID);
				if (updated) selectedDebt = updated;
			} else {
				error = result.error?.message ?? 'Gagal mencatat pembayaran.';
			}
		} catch {
			error = 'Gagal menghubungi server.';
		}
	}

	function openAddDebt(direction: Direction) {
		debtForm = { id: '', direction, person_name: '', amount: 0, due_date: '', notes: '' };
		showDebtModal = true;
	}

	function openEditDebt(debt: Debt) {
		debtForm = {
			id: debt.ID,
			direction: debt.Direction,
			person_name: debt.PersonName,
			amount: debt.TotalAmount,
			due_date: debt.DueDate ? debt.DueDate.split('T')[0] : '',
			notes: debt.Notes ?? ''
		};
		showDebtModal = true;
	}

	function openPaymentModal(debt: Debt) {
		selectedDebt = debt;
		payments = [];
		payForm = { amount: debt.RemainingAmount, payment_date: new Date().toISOString().split('T')[0], notes: '' };
		void fetchPayments(debt.ID);
		showPaymentModal = true;
	}

	onMount(() => { void fetchDebts(); });

	const receivables = $derived(debts.filter((d) => d.Direction === 'RECEIVABLE'));
	const payables = $derived(debts.filter((d) => d.Direction === 'PAYABLE'));
	const activeList = $derived(activeTab === 'RECEIVABLE' ? receivables : payables);

	const totalReceivable = $derived(receivables.filter((d) => d.Status === 'ACTIVE').reduce((a, d) => a + d.RemainingAmount, 0));
	const totalPayable = $derived(payables.filter((d) => d.Status === 'ACTIVE').reduce((a, d) => a + d.RemainingAmount, 0));

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(val);
	}

	function formatDate(iso?: string | null) {
		if (!iso) return '—';
		try { return new Date(iso).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' }); }
		catch { return '—'; }
	}

	function paidPct(debt: Debt) {
		if (debt.TotalAmount <= 0) return 0;
		return Math.min(100, Math.round(((debt.TotalAmount - debt.RemainingAmount) / debt.TotalAmount) * 100));
	}

	function isOverdue(debt: Debt) {
		if (!debt.DueDate || debt.Status === 'SETTLED') return false;
		return new Date(debt.DueDate) < new Date();
	}
</script>

<div class="animate-in fade-in space-y-8 pb-24 duration-500">
	<!-- Header -->
	<div class="flex flex-col justify-between gap-6 md:flex-row md:items-start">
		<div class="space-y-1">
			<h1 class="text-3xl font-black text-[#0a2e31]">Hutang & Piutang</h1>
			<p class="text-sm font-medium text-gray-500">Catat siapa yang berutang ke Anda dan berapa yang harus Anda bayar.</p>
		</div>

		<!-- Summary Cards -->
		<div class="flex flex-col gap-3 sm:flex-row">
			<button
				onclick={() => (activeTab = 'RECEIVABLE')}
				class="flex flex-col items-end rounded-[2rem] px-7 py-5 text-left shadow-xl transition-all {activeTab === 'RECEIVABLE' ? 'bg-[#0a2e31] text-white shadow-teal-950/20' : 'border border-gray-100 bg-white text-[#0a2e31]'}"
			>
				<span class="mb-1 text-[10px] font-black tracking-[0.2em] {activeTab === 'RECEIVABLE' ? 'text-teal-400/80' : 'text-gray-400'} uppercase">Piutang Saya</span>
				<span class="text-xl font-black tabular-nums">{formatCurrency(totalReceivable)}</span>
				<span class="mt-0.5 text-[10px] font-bold {activeTab === 'RECEIVABLE' ? 'text-teal-300/70' : 'text-gray-400'}">{receivables.filter(d => d.Status === 'ACTIVE').length} aktif</span>
			</button>
			<button
				onclick={() => (activeTab = 'PAYABLE')}
				class="flex flex-col items-end rounded-[2rem] px-7 py-5 text-left shadow-xl transition-all {activeTab === 'PAYABLE' ? 'bg-red-700 text-white shadow-red-900/20' : 'border border-gray-100 bg-white text-[#0a2e31]'}"
			>
				<span class="mb-1 text-[10px] font-black tracking-[0.2em] {activeTab === 'PAYABLE' ? 'text-red-300/80' : 'text-gray-400'} uppercase">Hutang Saya</span>
				<span class="text-xl font-black tabular-nums">{formatCurrency(totalPayable)}</span>
				<span class="mt-0.5 text-[10px] font-bold {activeTab === 'PAYABLE' ? 'text-red-300/70' : 'text-gray-400'}">{payables.filter(d => d.Status === 'ACTIVE').length} aktif</span>
			</button>
		</div>
	</div>

	<!-- Error Banner -->
	{#if error}
		<div class="flex items-center gap-3 rounded-2xl border border-red-100 bg-red-50 p-4 text-red-800" in:fly={{ y: -8, duration: 300 }}>
			<svg class="h-5 w-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
			<span class="text-sm font-bold">{error}</span>
			<button onclick={() => (error = null)} aria-label="Tutup pesan error" class="ml-auto text-red-400 hover:text-red-600">✕</button>
		</div>
	{/if}

	<!-- Tab Action Row -->
	<div class="flex items-center justify-between">
		<div class="flex rounded-2xl bg-gray-100 p-1">
			<button
				onclick={() => (activeTab = 'RECEIVABLE')}
				class="rounded-xl px-5 py-2.5 text-xs font-black tracking-widest uppercase transition-all {activeTab === 'RECEIVABLE' ? 'bg-[#0a2e31] text-white shadow-sm' : 'text-gray-500'}"
			>Piutang</button>
			<button
				onclick={() => (activeTab = 'PAYABLE')}
				class="rounded-xl px-5 py-2.5 text-xs font-black tracking-widest uppercase transition-all {activeTab === 'PAYABLE' ? 'bg-red-700 text-white shadow-sm' : 'text-gray-500'}"
			>Hutang</button>
		</div>

		<button
			onclick={() => openAddDebt(activeTab)}
			class="inline-flex items-center gap-2 rounded-2xl px-6 py-3.5 text-xs font-black tracking-widest uppercase shadow-lg transition-all active:scale-95 {activeTab === 'RECEIVABLE' ? 'bg-[#217b84] text-white hover:bg-[#1a5f66]' : 'bg-red-600 text-white hover:bg-red-700'}"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3"><path d="M12 4v16m8-8H4"/></svg>
			{activeTab === 'RECEIVABLE' ? 'Tambah Piutang' : 'Tambah Hutang'}
		</button>
	</div>

	<!-- Debt Cards -->
	<div class="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
		{#if loading}
			{#each [0, 1, 2] as i (i)}
				<div class="h-52 animate-pulse rounded-[2.5rem] border border-gray-100 bg-white"></div>
			{/each}
		{:else if activeList.length === 0}
			<div class="col-span-full space-y-4 rounded-[2.5rem] border border-dashed border-gray-200 bg-white py-20 text-center">
				<div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-gray-50 text-gray-300">
					<svg class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z"/></svg>
				</div>
				<div class="space-y-1">
					<p class="font-bold text-[#0a2e31]">Belum ada catatan {activeTab === 'RECEIVABLE' ? 'piutang' : 'hutang'}</p>
					<p class="text-xs text-gray-400">Klik tombol di atas untuk mulai mencatat.</p>
				</div>
			</div>
		{:else}
			{#each activeList as debt (debt.ID)}
				{@const settled = debt.Status === 'SETTLED'}
				{@const overdue = isOverdue(debt)}
				{@const paid = paidPct(debt)}
				{@const isPayable = debt.Direction === 'PAYABLE'}

				<div class="group relative overflow-hidden rounded-[2.5rem] border bg-white p-7 shadow-sm transition-all hover:-translate-y-1 hover:shadow-xl {settled ? 'border-gray-100 opacity-70' : overdue ? 'border-red-100' : 'border-gray-100'}">
					<div class="absolute -top-4 -right-4 h-20 w-20 rounded-full {isPayable ? 'bg-red-50/50' : 'bg-teal-50/50'} transition-transform duration-700 group-hover:scale-125"></div>

					<div class="relative z-10 flex h-full flex-col gap-4">
						<!-- Header -->
						<div class="flex items-start justify-between">
							<div>
								{#if overdue}
									<span class="mb-1 inline-block rounded-full bg-red-50 px-2.5 py-0.5 text-[9px] font-black tracking-widest text-red-600 uppercase">Jatuh Tempo!</span>
								{:else if settled}
									<span class="mb-1 inline-block rounded-full bg-green-50 px-2.5 py-0.5 text-[9px] font-black tracking-widest text-green-600 uppercase">Lunas</span>
								{/if}
								<h3 class="text-lg font-black text-[#0a2e31]">{debt.PersonName}</h3>
								{#if debt.DueDate}
									<p class="text-[11px] font-bold text-gray-400">Jatuh tempo: {formatDate(debt.DueDate)}</p>
								{/if}
							</div>
							<button
								onclick={() => openEditDebt(debt)}
								class="p-2 text-gray-300 hover:text-[#0a2e31]"
								aria-label="Edit catatan"
							>
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
							</button>
						</div>

						<!-- Amount Info -->
						<div class="grid grid-cols-2 gap-3">
							<div class="space-y-0.5 rounded-2xl bg-gray-50 p-4">
								<p class="text-[10px] font-black tracking-widest text-gray-400 uppercase">Total</p>
								<p class="text-base font-black text-[#0a2e31]">{formatCurrency(debt.TotalAmount)}</p>
							</div>
							<div class="space-y-0.5 rounded-2xl {settled ? 'bg-green-50' : isPayable ? 'bg-red-50' : 'bg-teal-50'} p-4">
								<p class="text-[10px] font-black tracking-widest {settled ? 'text-green-600' : isPayable ? 'text-red-500' : 'text-teal-600'} uppercase">Sisa</p>
								<p class="text-base font-black {settled ? 'text-green-700' : 'text-[#0a2e31]'}">{formatCurrency(debt.RemainingAmount)}</p>
							</div>
						</div>

						<!-- Progress Bar -->
						<div class="space-y-1.5">
							<div class="flex justify-between text-[10px] font-bold text-gray-400">
								<span>Sudah dibayar</span>
								<span>{paid}%</span>
							</div>
							<div class="h-2 overflow-hidden rounded-full bg-gray-100">
								<div
									class="h-full rounded-full transition-all duration-700 {settled ? 'bg-green-400' : isPayable ? 'bg-red-400' : 'bg-teal-400'}"
									style="width: {paid}%"
								></div>
							</div>
						</div>

						{#if debt.Notes}
							<p class="truncate text-xs font-medium text-gray-400">"{debt.Notes}"</p>
						{/if}

						<!-- Action Buttons -->
						{#if !settled}
							<button
								onclick={() => openPaymentModal(debt)}
								class="mt-auto w-full rounded-xl py-3 text-[11px] font-black tracking-[0.1em] uppercase transition-all {isPayable ? 'bg-red-50 text-red-700 hover:bg-red-100' : 'bg-teal-50 text-teal-700 hover:bg-teal-100'}"
							>
								{isPayable ? 'Bayar / Cicil' : 'Terima Pembayaran'}
							</button>
						{:else}
							<button
								onclick={() => openPaymentModal(debt)}
								class="mt-auto w-full rounded-xl bg-gray-50 py-3 text-[11px] font-black tracking-[0.1em] text-gray-500 uppercase transition-all hover:bg-gray-100"
							>
								Lihat Riwayat
							</button>
						{/if}
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>

<!-- Modal Tambah / Edit Catatan -->
{#if showDebtModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-[#0a2e31]/40 p-4 backdrop-blur-sm" in:fade={{ duration: 200 }}>
		<div class="w-full max-w-lg overflow-hidden rounded-[2.5rem] bg-white shadow-2xl" in:fly={{ y: 20, duration: 400 }}>
			<div class="space-y-7 p-10">
				<div class="flex items-center justify-between">
					<div>
						<h2 class="text-2xl font-black text-[#0a2e31]">
							{debtForm.id ? 'Edit Catatan' : debtForm.direction === 'RECEIVABLE' ? 'Tambah Piutang' : 'Tambah Hutang'}
						</h2>
						<p class="text-xs font-medium text-gray-400">
							{debtForm.direction === 'RECEIVABLE' ? 'Orang yang berhutang kepada Anda' : 'Hutang Anda kepada orang lain'}
						</p>
					</div>
					<button onclick={() => (showDebtModal = false)} aria-label="Tutup modal" class="text-gray-300 hover:text-gray-500">
						<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path d="M6 18L18 6M6 6l12 12"/></svg>
					</button>
				</div>

				<form onsubmit={handleSaveDebt} class="space-y-5">
					<!-- Nama -->
					<div class="space-y-2">
						<label for="person" class="block px-1 text-[11px] font-black tracking-widest text-gray-400 uppercase">
							{debtForm.direction === 'RECEIVABLE' ? 'Nama Peminjam' : 'Nama Pemberi Hutang'}
						</label>
						<input
							id="person"
							type="text"
							required
							bind:value={debtForm.person_name}
							placeholder="Nama lengkap"
							class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-bold text-[#0a2e31] outline-none transition-all focus:ring-4 focus:ring-teal-50"
						/>
					</div>

					<!-- Nominal (hanya tambah baru) -->
					{#if !debtForm.id}
						<div class="space-y-2">
							<label for="amount" class="block px-1 text-[11px] font-black tracking-widest text-gray-400 uppercase">Nominal (Rp)</label>
							<input
								id="amount"
								type="number"
								required
								min="1"
								bind:value={debtForm.amount}
								placeholder="0"
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-bold text-[#0a2e31] outline-none transition-all focus:ring-4 focus:ring-teal-50"
							/>
						</div>
					{/if}

					<div class="grid grid-cols-2 gap-4">
						<!-- Jatuh Tempo -->
						<div class="space-y-2">
							<label for="due" class="block px-1 text-[11px] font-black tracking-widest text-gray-400 uppercase">Jatuh Tempo</label>
							<input
								id="due"
								type="date"
								bind:value={debtForm.due_date}
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-bold text-[#0a2e31] outline-none transition-all focus:ring-4 focus:ring-teal-50"
							/>
						</div>

						<!-- Catatan -->
						<div class="space-y-2">
							<label for="notes-d" class="block px-1 text-[11px] font-black tracking-widest text-gray-400 uppercase">Keterangan</label>
							<input
								id="notes-d"
								type="text"
								bind:value={debtForm.notes}
								placeholder="Opsional"
								class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-5 py-3.5 font-bold text-[#0a2e31] outline-none transition-all focus:ring-4 focus:ring-teal-50"
							/>
						</div>
					</div>

					<div class="flex gap-4 pt-2">
						{#if debtForm.id}
							<button
								type="button"
								onclick={handleDeleteDebt}
								class="rounded-2xl border-2 border-red-50 px-6 py-4 text-xs font-black tracking-widest text-red-500 uppercase transition-all hover:bg-red-50"
							>Hapus</button>
						{/if}
						<button
							type="submit"
							class="flex-1 rounded-2xl py-4 text-xs font-black tracking-widest text-white uppercase shadow-xl transition-all active:scale-[0.98] {debtForm.direction === 'RECEIVABLE' ? 'bg-[#0a2e31] hover:bg-black' : 'bg-red-700 hover:bg-red-800'}"
						>
							{debtForm.id ? 'Simpan Perubahan' : 'Simpan'}
						</button>
					</div>
				</form>
			</div>
		</div>
	</div>
{/if}

<!-- Modal Riwayat & Pembayaran -->
{#if showPaymentModal && selectedDebt}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-[#0a2e31]/40 p-4 backdrop-blur-sm" in:fade={{ duration: 200 }}>
		<div class="flex max-h-[90vh] w-full max-w-4xl flex-col overflow-hidden rounded-[3rem] bg-gray-50 shadow-2xl" in:fly={{ y: 20, duration: 400 }}>
			<!-- Header -->
			<div class="flex items-center justify-between border-b border-gray-100 bg-white p-8">
				<div>
					<h2 class="text-2xl font-black text-[#0a2e31]">{selectedDebt.PersonName}</h2>
					<p class="text-xs font-bold text-gray-400">
						{selectedDebt.Direction === 'RECEIVABLE' ? 'Piutang Anda' : 'Hutang Anda'} &middot;
						Sisa {formatCurrency(selectedDebt.RemainingAmount)} dari {formatCurrency(selectedDebt.TotalAmount)}
					</p>
				</div>
				<button onclick={() => (showPaymentModal = false)} aria-label="Tutup modal" class="p-2 text-gray-300 hover:text-gray-500">
					<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path d="M6 18L18 6M6 6l12 12"/></svg>
				</button>
			</div>

			<div class="grid flex-1 grid-cols-1 gap-8 overflow-y-auto p-8 lg:grid-cols-5">
				<!-- Payment History -->
				<div class="space-y-4 lg:col-span-3">
					<h3 class="text-sm font-black tracking-widest text-[#0a2e31] uppercase">Riwayat Pembayaran</h3>
					{#if paymentsLoading}
						<div class="space-y-3">
							{#each [0, 1, 2] as i (i)}<div class="h-16 animate-pulse rounded-2xl bg-white"></div>{/each}
						</div>
					{:else if payments.length === 0}
						<div class="rounded-3xl border border-gray-100 bg-white p-10 text-center">
							<p class="text-sm font-bold text-gray-400">Belum ada riwayat pembayaran.</p>
						</div>
					{:else}
						<div class="space-y-3">
							{#each payments as p (p.ID)}
								<div class="flex items-center justify-between rounded-2xl border border-gray-100 bg-white p-5">
									<div class="flex items-center gap-4">
										<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-green-50 text-green-600">
											<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"/></svg>
										</div>
										<div>
											<p class="text-sm font-black text-[#0a2e31]">{formatCurrency(p.Amount)}</p>
											<p class="text-[10px] font-bold text-gray-400">{formatDate(p.PaymentDate)}</p>
										</div>
									</div>
									{#if p.Notes}
										<p class="max-w-[40%] truncate text-right text-xs text-gray-400">"{p.Notes}"</p>
									{/if}
								</div>
							{/each}
						</div>
					{/if}
				</div>

				<!-- Form Catat Pembayaran -->
				<div class="lg:col-span-2">
					{#if selectedDebt.Status !== 'SETTLED'}
						<div class="sticky top-0 space-y-5 rounded-3xl border border-gray-100 bg-white p-7 shadow-sm">
							<h3 class="text-sm font-black tracking-widest text-[#0a2e31] uppercase">
								{selectedDebt.Direction === 'RECEIVABLE' ? 'Terima Pembayaran' : 'Catat Cicilan'}
							</h3>

							<form onsubmit={handleRecordPayment} class="space-y-4">
								<div class="space-y-1.5">
									<label for="pay-amount" class="px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase">Jumlah (Rp)</label>
									<input
										id="pay-amount"
										type="number"
										required
										min="1"
										max={selectedDebt.RemainingAmount}
										bind:value={payForm.amount}
										class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] outline-none focus:ring-2 focus:ring-teal-400"
									/>
									{#if payForm.amount >= selectedDebt.RemainingAmount}
										<p class="px-1 text-[10px] font-bold text-green-600">Ini akan melunasi hutang</p>
									{:else if payForm.amount > 0}
										<p class="px-1 text-[10px] font-medium text-gray-400">Sisa setelah ini: {formatCurrency(selectedDebt.RemainingAmount - payForm.amount)}</p>
									{/if}
								</div>

								<div class="space-y-1.5">
									<label for="pay-date" class="px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase">Tanggal</label>
									<input
										id="pay-date"
										type="date"
										required
										bind:value={payForm.payment_date}
										class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] outline-none focus:ring-2 focus:ring-teal-400"
									/>
								</div>

								<div class="space-y-1.5">
									<label for="pay-notes" class="px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase">Keterangan</label>
									<input
										id="pay-notes"
										type="text"
										bind:value={payForm.notes}
										placeholder="Opsional"
										class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] outline-none focus:ring-2 focus:ring-teal-400"
									/>
								</div>

								<button
									type="submit"
									class="w-full rounded-xl py-3.5 text-[10px] font-black tracking-widest uppercase shadow-lg transition-all active:scale-[0.98] {selectedDebt.Direction === 'RECEIVABLE' ? 'bg-[#0a2e31] hover:bg-black' : 'bg-red-700 hover:bg-red-800'} text-white"
								>
									Simpan
								</button>
							</form>
						</div>
					{:else}
						<div class="rounded-3xl border border-green-100 bg-green-50 p-7 text-center">
							<div class="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-full bg-green-100 text-green-600">
								<svg class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"/></svg>
							</div>
							<p class="font-black text-green-800">Lunas!</p>
							<p class="text-xs font-medium text-green-600">Semua pembayaran sudah diterima.</p>
						</div>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	::-webkit-scrollbar { width: 6px; }
	::-webkit-scrollbar-track { background: transparent; }
	::-webkit-scrollbar-thumb { background: #e5e7eb; border-radius: 10px; }
	::-webkit-scrollbar-thumb:hover { background: #d1d5db; }
</style>
