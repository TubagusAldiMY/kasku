<script lang="ts">
	import { onMount } from 'svelte';
	import { syncStatus, triggerManualSync, enqueueUpdate } from '$lib/sync';
	import { syncQueueRepo, conflictsRepo } from '$lib/db';
	import type {
		SyncQueueRow,
		SyncConflictRow,
		AccountRow,
		TransactionRow,
		InvestmentRow
	} from '$lib/db';

	type AnyRow = AccountRow | TransactionRow | InvestmentRow;

	const RESOURCE_LABEL: Record<SyncConflictRow['resource'], string> = {
		accounts: 'Akun',
		transactions: 'Transaksi',
		investments: 'Investasi'
	};

	// Field internal yang tak perlu ditampilkan saat membandingkan nilai.
	const HIDDEN_FIELDS = new Set(['_local_dirty', '_deleted', 'sync_id']);

	let failed = $state<SyncQueueRow[]>([]);
	let conflicts = $state<SyncConflictRow[]>([]);
	let loading = $state(true);
	let busy = $state(false);

	async function refresh() {
		[failed, conflicts] = await Promise.all([syncQueueRepo.failed(), conflictsRepo.list()]);
		syncStatus.setConflictCount(conflicts.length);
		syncStatus.setQueuedCount(await syncQueueRepo.count());
	}

	onMount(async () => {
		await refresh();
		loading = false;
	});

	async function handleSyncNow() {
		busy = true;
		try {
			await triggerManualSync();
			await refresh();
		} finally {
			busy = false;
		}
	}

	// "Pakai versi saya": re-push nilai lokal sebagai update baru (updated_at = now,
	// jadi menang atas server lewat LWW). ponytail: tak menghidupkan ulang entity yang
	// sudah dihapus server (enqueueUpdate no-op bila lokal sudah hilang) — cukup untuk v1.
	async function keepLocal(c: SyncConflictRow) {
		busy = true;
		try {
			await enqueueUpdate<AnyRow>(c.resource, c.entity_id, c.local_value as Partial<AnyRow>);
			await conflictsRepo.resolve(c.id);
			await triggerManualSync();
			await refresh();
		} finally {
			busy = false;
		}
	}

	async function acceptServer(c: SyncConflictRow) {
		busy = true;
		try {
			await conflictsRepo.resolve(c.id);
			await refresh();
		} finally {
			busy = false;
		}
	}

	function fields(value: unknown): [string, string][] {
		if (!value || typeof value !== 'object') return [];
		return Object.entries(value as Record<string, unknown>)
			.filter(([k]) => !HIDDEN_FIELDS.has(k))
			.map(([k, v]) => [k, typeof v === 'object' ? JSON.stringify(v) : String(v)]);
	}

	function formatTime(iso: string): string {
		const d = new Date(iso);
		return Number.isNaN(d.getTime()) ? iso : d.toLocaleString('id-ID');
	}
</script>

<div class="mx-auto max-w-2xl space-y-8 px-1 py-2">
	<div>
		<h1 class="font-serif text-[30px] leading-tight tracking-tight text-ink">Sinkronisasi</h1>
		<p class="mt-2 text-sm text-ink/60">Status data offline, antrean, dan konflik.</p>
	</div>

	<!-- Status ringkas -->
	<div class="rounded-2xl border border-ink/10 bg-card p-5">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-2.5">
				<span
					class="h-2.5 w-2.5 rounded-full {syncStatus.online ? 'bg-teal' : 'bg-clay'}"
					aria-hidden="true"
				></span>
				<span class="text-sm font-semibold text-ink">
					{syncStatus.online ? 'Tersambung' : 'Mode offline'}
				</span>
			</div>
			<button
				onclick={handleSyncNow}
				disabled={busy || syncStatus.running || !syncStatus.online}
				class="rounded-full bg-teal px-4 py-2 text-[13px] font-semibold text-card transition-colors hover:bg-ink disabled:opacity-50"
			>
				{syncStatus.running || busy ? 'Menyinkronkan…' : 'Sinkronkan sekarang'}
			</button>
		</div>
		<dl class="mt-4 grid grid-cols-2 gap-3 text-[13px]">
			<div>
				<dt class="text-ink/45">Terakhir sinkron</dt>
				<dd class="font-medium text-ink">
					{syncStatus.lastSyncAt ? formatTime(syncStatus.lastSyncAt) : '—'}
				</dd>
			</div>
			<div>
				<dt class="text-ink/45">Menunggu terkirim</dt>
				<dd class="font-medium text-ink">{syncStatus.queuedCount}</dd>
			</div>
		</dl>
		{#if syncStatus.error}
			<p class="mt-3 rounded-lg border border-clay/25 bg-clay/5 px-3 py-2 text-[12px] text-clay">
				Kesalahan terakhir: {syncStatus.error}
			</p>
		{/if}
	</div>

	{#if loading}
		<p class="text-sm text-ink/50">Memuat…</p>
	{:else}
		<!-- Konflik -->
		<section class="space-y-3">
			<h2 class="font-serif text-[20px] tracking-tight text-ink">
				Konflik {#if conflicts.length > 0}<span class="text-clay">({conflicts.length})</span>{/if}
			</h2>
			{#if conflicts.length === 0}
				<p class="rounded-xl border border-ink/10 bg-field px-4 py-3 text-[13px] text-ink/50">
					Tak ada konflik. Perubahanmu tersinkron mulus.
				</p>
			{:else}
				<p class="text-[13px] text-ink/60">
					Perubahanmu ditimpa versi server yang lebih baru. Pilih mana yang dipertahankan.
				</p>
				{#each conflicts as c (c.id)}
					<div class="rounded-2xl border border-clay/20 bg-clay/5 p-4">
						<p class="text-[13px] font-semibold text-ink">
							{RESOURCE_LABEL[c.resource]} · <span class="font-normal text-ink/50"
								>{formatTime(c.detected_at)}</span
							>
						</p>
						<div class="mt-3 grid grid-cols-2 gap-3">
							<div class="rounded-xl border border-ink/10 bg-card p-3">
								<p class="mb-1.5 text-[11px] font-semibold tracking-wide text-teal uppercase">
									Versi saya
								</p>
								<dl class="space-y-0.5 text-[12px]">
									{#each fields(c.local_value) as [k, v] (k)}
										<div class="flex justify-between gap-2">
											<dt class="text-ink/45">{k}</dt>
											<dd class="truncate font-medium text-ink" title={v}>{v}</dd>
										</div>
									{/each}
								</dl>
							</div>
							<div class="rounded-xl border border-ink/10 bg-card p-3">
								<p class="mb-1.5 text-[11px] font-semibold tracking-wide text-ink/40 uppercase">
									Versi server
								</p>
								<dl class="space-y-0.5 text-[12px]">
									{#each fields(c.server_value) as [k, v] (k)}
										<div class="flex justify-between gap-2">
											<dt class="text-ink/45">{k}</dt>
											<dd class="truncate font-medium text-ink" title={v}>{v}</dd>
										</div>
									{/each}
								</dl>
							</div>
						</div>
						<div class="mt-3 flex gap-2">
							<button
								onclick={() => keepLocal(c)}
								disabled={busy}
								class="flex-1 rounded-full bg-teal py-2 text-[12px] font-semibold text-card transition-colors hover:bg-ink disabled:opacity-50"
							>
								Pakai versi saya
							</button>
							<button
								onclick={() => acceptServer(c)}
								disabled={busy}
								class="flex-1 rounded-full border border-ink/20 py-2 text-[12px] font-semibold text-ink/70 transition-colors hover:bg-ink/5 disabled:opacity-50"
							>
								Terima server
							</button>
						</div>
					</div>
				{/each}
			{/if}
		</section>

		<!-- Gagal terkirim -->
		<section class="space-y-3">
			<h2 class="font-serif text-[20px] tracking-tight text-ink">
				Gagal terkirim {#if failed.length > 0}<span class="text-clay">({failed.length})</span>{/if}
			</h2>
			{#if failed.length === 0}
				<p class="rounded-xl border border-ink/10 bg-field px-4 py-3 text-[13px] text-ink/50">
					Tak ada yang gagal.
				</p>
			{:else}
				<p class="text-[13px] text-ink/60">
					Operasi ini gagal terkirim dan akan dicoba ulang otomatis. Kamu juga bisa coba sekarang.
				</p>
				{#each failed as f (f.sync_id)}
					<div class="rounded-xl border border-ink/10 bg-card px-4 py-3">
						<div class="flex items-center justify-between">
							<p class="text-[13px] font-medium text-ink">
								{RESOURCE_LABEL[f.resource as SyncConflictRow['resource']] ?? f.resource} · {f.operation}
							</p>
							<span class="text-[11px] text-ink/40">{f.attempts}× gagal</span>
						</div>
						{#if f.last_error}
							<p class="mt-1 truncate text-[11px] text-clay" title={f.last_error}>{f.last_error}</p>
						{/if}
					</div>
				{/each}
				<button
					onclick={handleSyncNow}
					disabled={busy || !syncStatus.online}
					class="w-full rounded-full border border-ink/20 py-2.5 text-[13px] font-semibold text-ink transition-colors hover:bg-ink/5 disabled:opacity-50"
				>
					Coba kirim ulang semua
				</button>
			{/if}
		</section>
	{/if}
</div>
