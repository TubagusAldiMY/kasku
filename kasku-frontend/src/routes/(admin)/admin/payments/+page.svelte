<script lang="ts">
	import { onMount } from 'svelte';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
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
				return 'border-teal/30 text-teal';
			case 'FAILED':
				return 'border-clay/30 text-clay';
			case 'PENDING':
				return 'border-gold/30 text-gold';
			default:
				return 'border-ink/15 text-ink/55';
		}
	}

	async function loadPayments() {
		loading = true;
		error = null;
		try {
			const params = new SvelteURLSearchParams({
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
		<div>
			<h1 class="font-serif text-3xl leading-tight tracking-tight text-ink">Pembayaran</h1>
			<p class="mt-1.5 text-sm text-ink/55">Riwayat transaksi pelanggan via Midtrans.</p>
		</div>
		<div class="flex items-center gap-2">
			<select
				bind:value={statusFilter}
				onchange={applyFilter}
				class="rounded-[10px] border border-ink/25 bg-field px-4 py-2.5 text-sm text-ink transition-colors outline-none focus:border-teal"
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
				class="rounded-full border border-ink/20 px-4 py-2.5 text-[13px] font-semibold text-ink transition-colors hover:border-ink/40 disabled:opacity-60"
			>
				{loading ? 'Memuat…' : 'Muat ulang'}
			</button>
		</div>
	</div>

	{#if error}
		<div class="flex items-start gap-2.5 rounded-xl border border-clay/25 bg-clay/5 p-4">
			<svg
				class="mt-px h-4 w-4 shrink-0 text-clay"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				stroke-width="2"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
				/>
			</svg>
			<p class="text-[13px] font-medium text-clay">{error}</p>
		</div>
	{/if}

	<div>
		<div
			class="grid grid-cols-[1fr_1.4fr_0.7fr_0.9fr_0.7fr_1fr] gap-4 border-b border-ink/25 py-3 text-[10.5px] font-semibold tracking-[0.12em] text-ink/50 uppercase"
		>
			<span>Order ID</span><span>Pengguna</span><span>Paket</span><span class="text-right"
				>Nominal</span
			><span>Status</span><span>Dibayar</span>
		</div>

		{#if loading && payments.length === 0}
			<div class="space-y-3 pt-4">
				{#each [0, 1, 2, 3, 4] as i (i)}
					<div class="h-6 animate-pulse rounded bg-ink/5"></div>
				{/each}
			</div>
		{:else if payments.length === 0}
			<p class="py-8 text-center text-sm text-ink/45">Tidak ada transaksi.</p>
		{:else}
			{#each payments as p, i (p.id)}
				<div
					in:fly={{ y: 8, delay: i * 20, duration: 200 }}
					class="grid grid-cols-[1fr_1.4fr_0.7fr_0.9fr_0.7fr_1fr] items-baseline gap-4 border-b border-ink/8 py-4 text-[13px]"
				>
					<span class="truncate font-mono text-xs text-ink">{p.order_id}</span>
					<span class="truncate font-medium text-ink">{p.user_email}</span>
					<span class="truncate text-teal">{p.plan_name}</span>
					<span class="text-right font-semibold text-ink tabular-nums"
						>{formatCurrency(p.amount_idr)}</span
					>
					<span>
						<span class="rounded-full border px-2 py-px text-[11px] {statusBadge(p.status)}">
							{p.status}
						</span>
					</span>
					<span class="text-ink/50">{formatDate(p.paid_at)}</span>
				</div>
			{/each}
		{/if}
	</div>

	{#if meta && meta.total > 0}
		<div class="flex items-center justify-between text-[13px] text-ink/55">
			<span>
				Menampilkan {(meta.page - 1) * meta.page_size + 1}–{Math.min(
					meta.page * meta.page_size,
					meta.total
				)} dari {meta.total}
			</span>
			<div class="flex items-center gap-2">
				<button
					type="button"
					onclick={() => changePage(-1)}
					disabled={page === 1 || loading}
					class="rounded-full border border-ink/20 px-3.5 py-1.5 font-semibold text-ink transition-colors hover:border-ink/40 disabled:opacity-40"
				>
					← Sebelumnya
				</button>
				<span class="font-semibold text-ink">Halaman {page} / {totalPages}</span>
				<button
					type="button"
					onclick={() => changePage(1)}
					disabled={page >= totalPages || loading}
					class="rounded-full border border-ink/20 px-3.5 py-1.5 font-semibold text-ink transition-colors hover:border-ink/40 disabled:opacity-40"
				>
					Berikutnya →
				</button>
			</div>
		</div>
	{/if}
</div>
