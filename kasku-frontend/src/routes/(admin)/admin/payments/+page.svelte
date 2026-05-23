<script lang="ts">
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import { adminApiFetch } from '$lib/api/admin_client';

	type PaymentItem = {
		id: string;
		order_id: string;
		user_email: string;
		amount_idr: number;
		status: string; // SUCCESS | FAILED | PENDING
		plan_name: string;
		paid_at: string | null;
		created_at: string;
	};

	type ListMeta = {
		page: number;
		page_size: number;
		total: number;
	};

	let payments = $state<PaymentItem[]>([]);
	let meta = $state<ListMeta | null>(null);
	let page = $state(1);
	const pageSize = 20;
	let statusFilter = $state<'all' | 'SUCCESS' | 'FAILED' | 'PENDING'>('all');
	let loading = $state(true);
	let error = $state<string | null>(null);

	const totalPages = $derived(meta ? Math.max(1, Math.ceil(meta.total / meta.page_size)) : 1);

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(val);
	}

	function formatDate(iso: string | null): string {
		if (!iso) return '—';
		try {
			return new Date(iso).toLocaleString('id-ID', {
				day: 'numeric',
				month: 'short',
				year: 'numeric',
				hour: '2-digit',
				minute: '2-digit'
			});
		} catch {
			return iso;
		}
	}

	function statusBadge(status: string) {
		switch (status) {
			case 'SUCCESS':
				return 'bg-green-50 text-green-700';
			case 'FAILED':
				return 'bg-red-50 text-red-700';
			case 'PENDING':
				return 'bg-amber-50 text-amber-700';
			default:
				return 'bg-gray-50 text-gray-700';
		}
	}

	async function loadPayments() {
		loading = true;
		error = null;
		try {
			const params = new URLSearchParams({
				page: String(page),
				page_size: String(pageSize)
			});
			if (statusFilter !== 'all') params.set('status', statusFilter);

			const res = await adminApiFetch(`/admin/payments?${params.toString()}`);
			const envelope = (await res.json()) as {
				success: boolean;
				data?: PaymentItem[];
				meta?: ListMeta;
				error?: { message?: string };
			};
			if (!res.ok || !envelope.success || !envelope.data) {
				throw new Error(envelope.error?.message ?? `HTTP ${res.status}`);
			}
			payments = envelope.data;
			meta = envelope.meta ?? null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Gagal memuat pembayaran';
		} finally {
			loading = false;
		}
	}

	function changePage(delta: number) {
		const next = page + delta;
		if (next < 1 || next > totalPages) return;
		page = next;
		loadPayments();
	}

	function applyFilter() {
		page = 1;
		loadPayments();
	}

	onMount(loadPayments);
</script>

<div class="space-y-8">
	<div class="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
		<div class="space-y-1">
			<h1 class="text-3xl font-black text-[#0a2e31]">Pembayaran</h1>
			<p class="font-medium text-gray-500">Riwayat transaksi pelanggan via Midtrans.</p>
		</div>
		<div class="flex items-center gap-2">
			<select
				bind:value={statusFilter}
				onchange={applyFilter}
				class="rounded-full border border-gray-200 bg-white px-4 py-2 text-xs font-bold text-[#0a2e31] focus:outline-none focus:ring-2 focus:ring-teal-500/40"
			>
				<option value="all">Semua status</option>
				<option value="SUCCESS">Sukses</option>
				<option value="FAILED">Gagal</option>
				<option value="PENDING">Pending</option>
			</select>
			<button
				type="button"
				onclick={loadPayments}
				disabled={loading}
				class="rounded-full border border-gray-200 bg-white px-4 py-2 text-[11px] font-bold text-[#0a2e31] hover:bg-gray-50 disabled:opacity-60"
			>
				{loading ? 'Memuat…' : 'Muat Ulang'}
			</button>
		</div>
	</div>

	{#if error}
		<div
			class="rounded-2xl border border-red-100 bg-red-50 px-5 py-4 text-xs font-bold text-red-700"
		>
			{error}
		</div>
	{/if}

	<div class="overflow-hidden rounded-[2rem] border border-gray-100 bg-white shadow-sm">
		<table class="w-full text-left text-sm">
			<thead class="border-b border-gray-100 bg-gray-50/60">
				<tr class="text-[10px] font-black tracking-widest text-gray-500 uppercase">
					<th class="px-6 py-4">Order ID</th>
					<th class="px-6 py-4">Pengguna</th>
					<th class="px-6 py-4">Paket</th>
					<th class="px-6 py-4 text-right">Nominal</th>
					<th class="px-6 py-4">Status</th>
					<th class="px-6 py-4">Dibayar</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-gray-50">
				{#if loading && payments.length === 0}
					{#each [0, 1, 2, 3, 4] as i (i)}
						<tr><td colspan="6" class="h-12 animate-pulse bg-gray-50/50"></td></tr>
					{/each}
				{:else if payments.length === 0}
					<tr>
						<td colspan="6" class="px-6 py-10 text-center text-xs font-bold text-gray-400">
							Tidak ada transaksi.
						</td>
					</tr>
				{:else}
					{#each payments as p, i (p.id)}
						<tr in:fly={{ y: 8, delay: i * 20, duration: 200 }} class="transition-colors hover:bg-gray-50/60">
							<td class="px-6 py-4 font-mono text-xs text-[#0a2e31]">{p.order_id}</td>
							<td class="px-6 py-4 text-sm font-bold text-[#0a2e31]">{p.user_email}</td>
							<td class="px-6 py-4 text-xs font-bold tracking-widest text-teal-700 uppercase">
								{p.plan_name}
							</td>
							<td class="px-6 py-4 text-right text-sm font-black text-[#0a2e31] tabular-nums">
								{formatCurrency(p.amount_idr)}
							</td>
							<td class="px-6 py-4">
								<span class="rounded-full px-2 py-0.5 text-[9px] font-black tracking-widest uppercase {statusBadge(p.status)}">
									{p.status}
								</span>
							</td>
							<td class="px-6 py-4 text-xs font-medium text-gray-500">{formatDate(p.paid_at)}</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>

	{#if meta && meta.total > 0}
		<div class="flex items-center justify-between text-xs font-bold text-gray-500">
			<span>
				Menampilkan {(meta.page - 1) * meta.page_size + 1}–{Math.min(meta.page * meta.page_size, meta.total)} dari {meta.total}
			</span>
			<div class="flex items-center gap-2">
				<button
					type="button"
					onclick={() => changePage(-1)}
					disabled={page === 1 || loading}
					class="rounded-full border border-gray-200 px-3 py-1 transition-colors hover:bg-gray-50 disabled:opacity-40"
				>
					← Sebelumnya
				</button>
				<span class="text-[#0a2e31]">Halaman {page} / {totalPages}</span>
				<button
					type="button"
					onclick={() => changePage(1)}
					disabled={page >= totalPages || loading}
					class="rounded-full border border-gray-200 px-3 py-1 transition-colors hover:bg-gray-50 disabled:opacity-40"
				>
					Berikutnya →
				</button>
			</div>
		</div>
	{/if}
</div>
