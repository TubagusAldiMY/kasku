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
				? await apiFetch(`/debts/${debtForm.id}`, {
						method: 'PUT',
						body: JSON.stringify({
							person_name: debtForm.person_name,
							due_date: debtForm.due_date || null,
							notes: debtForm.notes
						})
					})
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
		if (!selectedDebt || !confirm('Hapus catatan hutang ini beserta riwayat pembayarannya?'))
			return;
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
		payForm = {
			amount: debt.RemainingAmount,
			payment_date: new Date().toISOString().split('T')[0],
			notes: ''
		};
		void fetchPayments(debt.ID);
		showPaymentModal = true;
	}

	onMount(() => {
		void fetchDebts();
	});

	const receivables = $derived(debts.filter((d) => d.Direction === 'RECEIVABLE'));
	const payables = $derived(debts.filter((d) => d.Direction === 'PAYABLE'));
	const activeList = $derived(activeTab === 'RECEIVABLE' ? receivables : payables);

	const totalReceivable = $derived(
		receivables.filter((d) => d.Status === 'ACTIVE').reduce((a, d) => a + d.RemainingAmount, 0)
	);
	const totalPayable = $derived(
		payables.filter((d) => d.Status === 'ACTIVE').reduce((a, d) => a + d.RemainingAmount, 0)
	);

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(val);
	}

	function formatDate(iso?: string | null) {
		if (!iso) return '—';
		try {
			return new Date(iso).toLocaleDateString('id-ID', {
				day: 'numeric',
				month: 'short',
				year: 'numeric'
			});
		} catch {
			return '—';
		}
	}

	function paidPct(debt: Debt) {
		if (debt.TotalAmount <= 0) return 0;
		return Math.min(
			100,
			Math.round(((debt.TotalAmount - debt.RemainingAmount) / debt.TotalAmount) * 100)
		);
	}

	function isOverdue(debt: Debt) {
		if (!debt.DueDate || debt.Status === 'SETTLED') return false;
		return new Date(debt.DueDate) < new Date();
	}
</script>

<div class="animate-in fade-in pb-4 duration-500">
	<!-- Header -->
	<section
		class="flex flex-col justify-between gap-8 border-b border-ink/10 pb-8 md:flex-row md:items-end"
	>
		<div>
			<h1 class="font-serif text-4xl tracking-tight text-ink">Hutang &amp; Piutang</h1>
			<p class="mt-2 text-sm text-ink/60">
				Catat siapa yang berutang ke Anda dan berapa yang harus Anda bayar.
			</p>
		</div>

		<!-- Summary figures -->
		<div class="grid grid-cols-2 gap-8 sm:gap-12">
			<button onclick={() => (activeTab = 'RECEIVABLE')} class="text-left">
				<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
					Piutang saya
				</p>
				<p
					class="mt-1.5 font-serif text-3xl tabular-nums {activeTab === 'RECEIVABLE'
						? 'text-teal'
						: 'text-ink'}"
				>
					{formatCurrency(totalReceivable)}
				</p>
				<p class="mt-1 text-xs text-ink/45">
					{receivables.filter((d) => d.Status === 'ACTIVE').length} aktif
				</p>
			</button>
			<button
				onclick={() => (activeTab = 'PAYABLE')}
				class="border-l border-ink/12 pl-8 text-left sm:pl-12"
			>
				<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">Hutang saya</p>
				<p
					class="mt-1.5 font-serif text-3xl tabular-nums {activeTab === 'PAYABLE'
						? 'text-clay'
						: 'text-ink'}"
				>
					{formatCurrency(totalPayable)}
				</p>
				<p class="mt-1 text-xs text-ink/45">
					{payables.filter((d) => d.Status === 'ACTIVE').length} aktif
				</p>
			</button>
		</div>
	</section>

	<!-- Error Banner -->
	{#if error}
		<div
			class="mt-6 flex items-center gap-3 rounded-xl border border-clay/25 bg-clay/5 px-4 py-3 text-clay"
			in:fly={{ y: -8, duration: 300 }}
		>
			<svg class="h-5 w-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"
				><path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
				/></svg
			>
			<span class="text-sm font-medium">{error}</span>
			<button
				onclick={() => (error = null)}
				aria-label="Tutup pesan error"
				class="ml-auto text-clay/60 hover:text-clay">✕</button
			>
		</div>
	{/if}

	<!-- Tab + Action Row -->
	<div class="mt-8 flex items-center justify-between">
		<div class="flex overflow-hidden rounded-full border border-ink/20 text-[12.5px] font-semibold">
			<button
				onclick={() => (activeTab = 'RECEIVABLE')}
				class="px-5 py-2 transition-colors {activeTab === 'RECEIVABLE'
					? 'bg-teal text-card'
					: 'text-ink/60 hover:text-ink'}">Piutang</button
			>
			<button
				onclick={() => (activeTab = 'PAYABLE')}
				class="px-5 py-2 transition-colors {activeTab === 'PAYABLE'
					? 'bg-clay text-card'
					: 'text-ink/60 hover:text-ink'}">Hutang</button
			>
		</div>

		<button
			onclick={() => openAddDebt(activeTab)}
			class="inline-flex items-center gap-2 rounded-full px-6 py-2.5 text-[13px] font-semibold transition-colors {activeTab ===
			'RECEIVABLE'
				? 'bg-teal text-card hover:bg-ink'
				: 'bg-clay text-card hover:bg-ink'}"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"
				><path d="M12 4v16m8-8H4" /></svg
			>
			{activeTab === 'RECEIVABLE' ? 'Tambah Piutang' : 'Tambah Hutang'}
		</button>
	</div>

	<!-- Debt Cards -->
	<div class="mt-6 grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
		{#if loading}
			{#each [0, 1, 2] as i (i)}
				<div class="h-52 animate-pulse rounded-2xl border border-ink/10 bg-card"></div>
			{/each}
		{:else if activeList.length === 0}
			<div
				class="col-span-full rounded-2xl border border-dashed border-ink/20 bg-card py-20 text-center"
			>
				<div
					class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-ink/5 text-ink/30"
				>
					<svg class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z"
						/></svg
					>
				</div>
				<p class="mt-4 font-serif text-xl text-ink">
					Belum ada catatan {activeTab === 'RECEIVABLE' ? 'piutang' : 'hutang'}
				</p>
				<p class="mt-1 text-sm text-ink/45">Klik tombol di atas untuk mulai mencatat.</p>
			</div>
		{:else}
			{#each activeList as debt (debt.ID)}
				{@const settled = debt.Status === 'SETTLED'}
				{@const overdue = isOverdue(debt)}
				{@const paid = paidPct(debt)}
				{@const isPayable = debt.Direction === 'PAYABLE'}

				<div
					class="flex flex-col rounded-2xl border bg-card p-6 transition-colors {settled
						? 'border-ink/10 opacity-70'
						: overdue
							? 'border-clay/25 bg-clay/5'
							: 'border-ink/10'}"
				>
					<div class="flex flex-col gap-4">
						<!-- Header -->
						<div class="flex items-start justify-between">
							<div>
								{#if overdue}
									<span
										class="mb-1.5 inline-block rounded-full border border-clay/30 px-2 py-px text-[10px] font-semibold tracking-[0.12em] text-clay uppercase"
										>Jatuh Tempo</span
									>
								{:else if settled}
									<span
										class="mb-1.5 inline-block rounded-full border border-teal/30 px-2 py-px text-[10px] font-semibold tracking-[0.12em] text-teal uppercase"
										>Lunas</span
									>
								{/if}
								<h3 class="font-serif text-2xl text-ink">{debt.PersonName}</h3>
								{#if debt.DueDate}
									<p class="mt-0.5 text-[12px] text-ink/45">
										Jatuh tempo: {formatDate(debt.DueDate)}
									</p>
								{/if}
							</div>
							<button
								onclick={() => openEditDebt(debt)}
								class="p-2 text-ink/30 transition-colors hover:text-ink"
								aria-label="Edit catatan"
							>
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"
									><path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
									/></svg
								>
							</button>
						</div>

						<!-- Amount Info -->
						<div class="grid grid-cols-2 gap-6 border-t border-ink/8 pt-4">
							<div>
								<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
									Total
								</p>
								<p class="mt-1 font-serif text-xl text-ink tabular-nums">
									{formatCurrency(debt.TotalAmount)}
								</p>
							</div>
							<div>
								<p
									class="text-[11px] font-semibold tracking-[0.12em] uppercase {settled
										? 'text-teal'
										: isPayable
											? 'text-clay'
											: 'text-teal'}"
								>
									Sisa
								</p>
								<p
									class="mt-1 font-serif text-xl tabular-nums {settled ? 'text-teal' : 'text-ink'}"
								>
									{formatCurrency(debt.RemainingAmount)}
								</p>
							</div>
						</div>

						<!-- Progress Bar -->
						<div class="space-y-1.5">
							<div class="flex justify-between text-[12px]">
								<span class="text-ink/55">Sudah dibayar</span>
								<span class="font-semibold text-ink/60">{paid}%</span>
							</div>
							<div class="h-[3px] bg-ink/10">
								<div
									class="h-full transition-all duration-700 {settled
										? 'bg-teal'
										: isPayable
											? 'bg-clay'
											: 'bg-teal'}"
									style="width: {paid}%"
								></div>
							</div>
						</div>

						{#if debt.Notes}
							<p class="truncate text-xs text-ink/45 italic">"{debt.Notes}"</p>
						{/if}

						<!-- Action Buttons -->
						{#if !settled}
							<button
								onclick={() => openPaymentModal(debt)}
								class="mt-2 w-full rounded-full border py-2.5 text-[12px] font-semibold transition-colors {isPayable
									? 'border-clay/30 text-clay hover:bg-clay/5'
									: 'border-teal/30 text-teal hover:bg-teal/5'}"
							>
								{isPayable ? 'Bayar / Cicil' : 'Terima Pembayaran'}
							</button>
						{:else}
							<button
								onclick={() => openPaymentModal(debt)}
								class="mt-2 w-full rounded-full border border-ink/15 py-2.5 text-[12px] font-semibold text-ink/55 transition-colors hover:bg-ink/5"
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
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
	>
		<div
			class="w-full max-w-lg overflow-hidden rounded-2xl border border-ink/10 bg-card shadow-2xl"
			in:fly={{ y: 20, duration: 400 }}
		>
			<div class="space-y-7 p-8 sm:p-10">
				<div class="flex items-start justify-between">
					<div>
						<h2 class="font-serif text-2xl text-ink">
							{debtForm.id
								? 'Edit Catatan'
								: debtForm.direction === 'RECEIVABLE'
									? 'Tambah Piutang'
									: 'Tambah Hutang'}
						</h2>
						<p class="mt-1 text-sm text-ink/55">
							{debtForm.direction === 'RECEIVABLE'
								? 'Orang yang berhutang kepada Anda'
								: 'Hutang Anda kepada orang lain'}
						</p>
					</div>
					<button
						onclick={() => (showDebtModal = false)}
						aria-label="Tutup modal"
						class="p-1 text-ink/30 transition-colors hover:text-ink"
					>
						<svg
							class="h-6 w-6"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"><path d="M6 18L18 6M6 6l12 12" /></svg
						>
					</button>
				</div>

				<form onsubmit={handleSaveDebt} class="space-y-5">
					<!-- Nama -->
					<div class="space-y-2">
						<label
							for="person"
							class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
						>
							{debtForm.direction === 'RECEIVABLE' ? 'Nama Peminjam' : 'Nama Pemberi Hutang'}
						</label>
						<input
							id="person"
							type="text"
							required
							bind:value={debtForm.person_name}
							placeholder="Nama lengkap"
							class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-ink transition-colors outline-none focus:border-teal"
						/>
					</div>

					<!-- Nominal (hanya tambah baru) -->
					{#if !debtForm.id}
						<div class="space-y-2">
							<label
								for="amount"
								class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>Nominal (Rp)</label
							>
							<input
								id="amount"
								type="number"
								required
								min="1"
								bind:value={debtForm.amount}
								placeholder="0"
								class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-ink transition-colors outline-none focus:border-teal"
							/>
						</div>
					{/if}

					<div class="grid grid-cols-2 gap-4">
						<!-- Jatuh Tempo -->
						<div class="space-y-2">
							<label
								for="due"
								class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>Jatuh Tempo</label
							>
							<input
								id="due"
								type="date"
								bind:value={debtForm.due_date}
								class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-ink transition-colors outline-none focus:border-teal"
							/>
						</div>

						<!-- Catatan -->
						<div class="space-y-2">
							<label
								for="notes-d"
								class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>Keterangan</label
							>
							<input
								id="notes-d"
								type="text"
								bind:value={debtForm.notes}
								placeholder="Opsional"
								class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-ink transition-colors outline-none focus:border-teal"
							/>
						</div>
					</div>

					<div class="flex gap-4 pt-2">
						{#if debtForm.id}
							<button
								type="button"
								onclick={handleDeleteDebt}
								class="rounded-full border border-ink/15 px-6 py-3 text-[12px] font-semibold text-clay transition-colors hover:bg-clay/5"
								>Hapus</button
							>
						{/if}
						<button
							type="submit"
							class="flex-1 rounded-full py-3 text-[13px] font-semibold text-card transition-colors {debtForm.direction ===
							'RECEIVABLE'
								? 'bg-teal hover:bg-ink'
								: 'bg-clay hover:bg-ink'}"
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
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
	>
		<div
			class="flex max-h-[90vh] w-full max-w-4xl flex-col overflow-hidden rounded-2xl border border-ink/10 bg-paper shadow-2xl"
			in:fly={{ y: 20, duration: 400 }}
		>
			<!-- Header -->
			<div class="flex items-start justify-between border-b border-ink/10 bg-card p-8">
				<div>
					<h2 class="font-serif text-2xl text-ink">{selectedDebt.PersonName}</h2>
					<p class="mt-1 text-sm text-ink/55">
						{selectedDebt.Direction === 'RECEIVABLE' ? 'Piutang Anda' : 'Hutang Anda'} &middot; Sisa {formatCurrency(
							selectedDebt.RemainingAmount
						)} dari {formatCurrency(selectedDebt.TotalAmount)}
					</p>
				</div>
				<button
					onclick={() => (showPaymentModal = false)}
					aria-label="Tutup modal"
					class="p-1 text-ink/30 transition-colors hover:text-ink"
				>
					<svg
						class="h-6 w-6"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"><path d="M6 18L18 6M6 6l12 12" /></svg
					>
				</button>
			</div>

			<div class="grid flex-1 grid-cols-1 gap-8 overflow-y-auto p-8 lg:grid-cols-5">
				<!-- Payment History -->
				<div class="space-y-4 lg:col-span-3">
					<h3 class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
						Riwayat Pembayaran
					</h3>
					{#if paymentsLoading}
						<div class="space-y-3">
							{#each [0, 1, 2] as i (i)}<div
									class="h-16 animate-pulse rounded-xl bg-card"
								></div>{/each}
						</div>
					{:else if payments.length === 0}
						<div class="rounded-2xl border border-ink/10 bg-card p-10 text-center">
							<p class="text-sm text-ink/45">Belum ada riwayat pembayaran.</p>
						</div>
					{:else}
						<div>
							{#each payments as p (p.ID)}
								<div
									class="flex items-center justify-between gap-4 border-b border-ink/8 py-4 last:border-0"
								>
									<div class="flex items-center gap-4">
										<div
											class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-teal/25 text-teal"
										>
											<svg
												class="h-4 w-4"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="2.5"
												><path
													stroke-linecap="round"
													stroke-linejoin="round"
													d="M5 13l4 4L19 7"
												/></svg
											>
										</div>
										<div>
											<p class="font-serif text-lg text-ink tabular-nums">
												{formatCurrency(p.Amount)}
											</p>
											<p class="text-[12px] text-ink/45">{formatDate(p.PaymentDate)}</p>
										</div>
									</div>
									{#if p.Notes}
										<p class="max-w-[40%] truncate text-right text-xs text-ink/45 italic">
											"{p.Notes}"
										</p>
									{/if}
								</div>
							{/each}
						</div>
					{/if}
				</div>

				<!-- Form Catat Pembayaran -->
				<div class="lg:col-span-2">
					{#if selectedDebt.Status !== 'SETTLED'}
						<div class="sticky top-0 space-y-5 rounded-2xl border border-ink/10 bg-card p-6">
							<h3 class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
								{selectedDebt.Direction === 'RECEIVABLE' ? 'Terima Pembayaran' : 'Catat Cicilan'}
							</h3>

							<form onsubmit={handleRecordPayment} class="space-y-4">
								<div class="space-y-1.5">
									<label
										for="pay-amount"
										class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
										>Jumlah (Rp)</label
									>
									<input
										id="pay-amount"
										type="number"
										required
										min="1"
										max={selectedDebt.RemainingAmount}
										bind:value={payForm.amount}
										class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none focus:border-teal"
									/>
									{#if payForm.amount >= selectedDebt.RemainingAmount}
										<p class="text-[11px] font-semibold text-teal">Ini akan melunasi hutang</p>
									{:else if payForm.amount > 0}
										<p class="text-[11px] text-ink/45">
											Sisa setelah ini: {formatCurrency(
												selectedDebt.RemainingAmount - payForm.amount
											)}
										</p>
									{/if}
								</div>

								<div class="space-y-1.5">
									<label
										for="pay-date"
										class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
										>Tanggal</label
									>
									<input
										id="pay-date"
										type="date"
										required
										bind:value={payForm.payment_date}
										class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none focus:border-teal"
									/>
								</div>

								<div class="space-y-1.5">
									<label
										for="pay-notes"
										class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
										>Keterangan</label
									>
									<input
										id="pay-notes"
										type="text"
										bind:value={payForm.notes}
										placeholder="Opsional"
										class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none focus:border-teal"
									/>
								</div>

								<button
									type="submit"
									class="w-full rounded-full py-3 text-[13px] font-semibold text-card transition-colors {selectedDebt.Direction ===
									'RECEIVABLE'
										? 'bg-teal hover:bg-ink'
										: 'bg-clay hover:bg-ink'}"
								>
									Simpan
								</button>
							</form>
						</div>
					{:else}
						<div class="rounded-2xl border border-teal/25 bg-teal/5 p-7 text-center">
							<div
								class="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-full border border-teal/30 text-teal"
							>
								<svg
									class="h-7 w-7"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="2.5"
									><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" /></svg
								>
							</div>
							<p class="font-serif text-xl text-ink">Lunas!</p>
							<p class="mt-1 text-sm text-ink/55">Semua pembayaran sudah diterima.</p>
						</div>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	::-webkit-scrollbar {
		width: 6px;
	}
	::-webkit-scrollbar-track {
		background: transparent;
	}
	::-webkit-scrollbar-thumb {
		background: rgba(233, 237, 244, 0.15);
		border-radius: 10px;
	}
	::-webkit-scrollbar-thumb:hover {
		background: rgba(233, 237, 244, 0.25);
	}
</style>
