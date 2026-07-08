<script lang="ts">
	import { onMount } from 'svelte';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
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
		if (action.startsWith('SUSPEND') || action.includes('DELETE'))
			return 'border-clay/30 text-clay';
		if (action.startsWith('LOGIN') || action.startsWith('LOGOUT'))
			return 'border-ink/15 text-ink/55';
		if (action.startsWith('OVERRIDE')) return 'border-gold/30 text-gold';
		if (action.startsWith('ACTIVATE')) return 'border-teal/30 text-teal';
		return 'border-teal/30 text-teal';
	}

	async function loadEntries() {
		loading = true;
		error = null;
		try {
			const params = new SvelteURLSearchParams({
				page: String(page),
				page_size: String(pageSize)
			});
			if (actionFilter.trim()) params.set('action', actionFilter.trim());

			const res = await adminApiFetch(`/admin/audit-log?${params.toString()}`);
			const envelope = (await res.json()) as {
				success: boolean;
				data?: AuditEntry[];
				meta?: ListMeta;
				error?: { message?: string };
			};
			if (!res.ok || !envelope.success || !envelope.data) {
				throw new Error(envelope.error?.message ?? `HTTP ${res.status}`);
			}
			entries = envelope.data;
			meta = envelope.meta ?? null;
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
		<div>
			<h1 class="font-serif text-3xl leading-tight tracking-tight text-ink">Audit log</h1>
			<p class="mt-1.5 text-sm text-ink/55">
				Jejak semua aksi administratif. Disimpan permanen untuk kepatuhan audit.
			</p>
		</div>
		<form onsubmit={applyFilter} class="flex items-center gap-2">
			<input
				type="text"
				bind:value={actionFilter}
				placeholder="Filter action (e.g. SUSPEND_USER)…"
				class="rounded-[10px] border border-ink/25 bg-field px-4 py-2.5 text-sm text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-teal"
			/>
			<button
				type="submit"
				class="rounded-full bg-teal px-5 py-2.5 text-[13px] font-semibold text-card transition-colors hover:bg-ink"
			>
				Filter
			</button>
		</form>
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

	<div class="border-t border-ink/25">
		{#if loading && entries.length === 0}
			<div class="space-y-3 pt-4">
				{#each [0, 1, 2, 3, 4, 5] as i (i)}
					<div class="h-8 animate-pulse rounded bg-ink/5"></div>
				{/each}
			</div>
		{:else if entries.length === 0}
			<p class="py-8 text-center text-sm text-ink/45">Tidak ada entri audit.</p>
		{:else}
			{#each entries as e, i (e.id)}
				<div
					in:fly={{ y: 8, delay: i * 10, duration: 180 }}
					class="flex items-start justify-between gap-4 border-b border-ink/8 py-4"
				>
					<div class="min-w-0">
						<p class="text-[13px] text-ink">
							<span class="font-semibold text-ink">{e.admin_email}</span>
							<span class="ml-2 rounded-full border px-2 py-px text-[11px] {actionBadge(e.action)}"
								>{e.action}</span
							>
							{#if e.target_id}
								<span class="ml-2 font-mono text-xs text-ink/60"
									>{e.target_type}/{e.target_id.slice(0, 8)}…</span
								>
							{:else}
								<span class="ml-2 font-mono text-xs text-ink/60">{e.target_type}</span>
							{/if}
						</p>
						<p class="mt-1 text-[11.5px] text-ink/45">
							{formatDate(e.created_at)} · <span class="font-mono">{e.ip_address}</span>
						</p>
					</div>
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
