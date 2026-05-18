<script lang="ts">
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import { adminApiFetch } from '$lib/api/admin_client';

	type AuditEntry = {
		id: string;
		admin_id: string;
		admin_email: string;
		action: string; // SUSPEND_USER | OVERRIDE_SUBSCRIPTION | LOGIN | LOGOUT | ...
		target_type: string; // user | subscription | system
		target_id: string | null;
		ip_address: string;
		user_agent: string;
		metadata: Record<string, unknown> | null;
		created_at: string;
	};

	type ListMeta = {
		page: number;
		page_size: number;
		total: number;
	};

	let entries = $state<AuditEntry[]>([]);
	let meta = $state<ListMeta | null>(null);
	let page = $state(1);
	const pageSize = 25;
	let actionFilter = $state('');
	let loading = $state(true);
	let error = $state<string | null>(null);

	const totalPages = $derived(meta ? Math.max(1, Math.ceil(meta.total / meta.page_size)) : 1);

	function formatDate(iso: string): string {
		try {
			return new Date(iso).toLocaleString('id-ID', {
				day: 'numeric',
				month: 'short',
				year: 'numeric',
				hour: '2-digit',
				minute: '2-digit',
				second: '2-digit'
			});
		} catch {
			return iso;
		}
	}

	function actionBadge(action: string) {
		if (action.startsWith('SUSPEND') || action.includes('DELETE')) return 'bg-red-50 text-red-700';
		if (action.startsWith('LOGIN') || action.startsWith('LOGOUT')) return 'bg-gray-50 text-gray-700';
		if (action.startsWith('OVERRIDE')) return 'bg-amber-50 text-amber-700';
		if (action.startsWith('ACTIVATE')) return 'bg-green-50 text-green-700';
		return 'bg-teal-50 text-teal-700';
	}

	async function loadEntries() {
		loading = true;
		error = null;
		try {
			const params = new URLSearchParams({
				page: String(page),
				page_size: String(pageSize)
			});
			if (actionFilter.trim()) params.set('action', actionFilter.trim());

			const res = await adminApiFetch(`/admin/audit-log?${params.toString()}`);
			const envelope = (await res.json()) as {
				success: boolean;
				data?: { items: AuditEntry[]; meta: ListMeta };
				error?: { message?: string };
			};
			if (!res.ok || !envelope.success || !envelope.data) {
				throw new Error(envelope.error?.message ?? `HTTP ${res.status}`);
			}
			entries = envelope.data.items;
			meta = envelope.data.meta;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Gagal memuat audit log';
		} finally {
			loading = false;
		}
	}

	function changePage(delta: number) {
		const next = page + delta;
		if (next < 1 || next > totalPages) return;
		page = next;
		loadEntries();
	}

	function applyFilter(e: SubmitEvent) {
		e.preventDefault();
		page = 1;
		loadEntries();
	}

	onMount(loadEntries);
</script>

<div class="space-y-8">
	<div class="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
		<div class="space-y-1">
			<h1 class="text-3xl font-black text-[#0a2e31]">Audit Log</h1>
			<p class="font-medium text-gray-500">
				Jejak semua aksi administratif. Disimpan permanen untuk kepatuhan audit.
			</p>
		</div>
		<form onsubmit={applyFilter} class="flex items-center gap-2">
			<input
				type="text"
				bind:value={actionFilter}
				placeholder="Filter action (e.g. SUSPEND_USER)…"
				class="rounded-full border border-gray-200 bg-white px-4 py-2 text-xs font-bold text-[#0a2e31] placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-teal-500/40"
			/>
			<button
				type="submit"
				class="rounded-full bg-[#0a2e31] px-4 py-2 text-[11px] font-black tracking-widest text-white uppercase hover:bg-[#143f43]"
			>
				Filter
			</button>
		</form>
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
					<th class="px-6 py-4">Waktu</th>
					<th class="px-6 py-4">Admin</th>
					<th class="px-6 py-4">Aksi</th>
					<th class="px-6 py-4">Target</th>
					<th class="px-6 py-4">IP</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-gray-50">
				{#if loading && entries.length === 0}
					{#each [0, 1, 2, 3, 4, 5] as i (i)}
						<tr><td colspan="5" class="h-12 animate-pulse bg-gray-50/50"></td></tr>
					{/each}
				{:else if entries.length === 0}
					<tr>
						<td colspan="5" class="px-6 py-10 text-center text-xs font-bold text-gray-400">
							Tidak ada entri audit.
						</td>
					</tr>
				{:else}
					{#each entries as e, i (e.id)}
						<tr in:fly={{ y: 8, delay: i * 10, duration: 180 }} class="transition-colors hover:bg-gray-50/60">
							<td class="px-6 py-3 text-xs font-mono text-gray-600">{formatDate(e.created_at)}</td>
							<td class="px-6 py-3 text-xs font-bold text-[#0a2e31]">{e.admin_email}</td>
							<td class="px-6 py-3">
								<span class="rounded-full px-2 py-0.5 text-[9px] font-black tracking-widest uppercase {actionBadge(e.action)}">
									{e.action}
								</span>
							</td>
							<td class="px-6 py-3 text-xs text-gray-600">
								{#if e.target_id}
									<span class="font-mono">{e.target_type}/{e.target_id.slice(0, 8)}…</span>
								{:else}
									<span class="font-mono">{e.target_type}</span>
								{/if}
							</td>
							<td class="px-6 py-3 text-xs font-mono text-gray-500">{e.ip_address}</td>
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
