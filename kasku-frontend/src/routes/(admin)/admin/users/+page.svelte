<script lang="ts">
	import { onMount } from 'svelte';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import { fly } from 'svelte/transition';
	import { resolve } from '$app/paths';
	import { adminApiFetch } from '$lib/api/admin_client';

	type UserListItem = {
		id: string;
		email: string;
		username: string;
		is_active: boolean;
		email_verified: boolean;
		subscription_tier: string;
		subscription_status: string;
		created_at: string;
	};

	type ListMeta = {
		page: number;
		page_size: number;
		total: number;
	};

	let users = $state<UserListItem[]>([]);
	let meta = $state<ListMeta | null>(null);
	let page = $state(1);
	const pageSize = 20;
	let loading = $state(true);
	let error = $state<string | null>(null);
	let search = $state('');

	const totalPages = $derived(meta ? Math.max(1, Math.ceil(meta.total / meta.page_size)) : 1);

	function formatDate(iso: string): string {
		try {
			return new Date(iso).toLocaleDateString('id-ID', {
				day: 'numeric',
				month: 'short',
				year: 'numeric'
			});
		} catch {
			return iso;
		}
	}

	async function loadUsers() {
		loading = true;
		error = null;
		try {
			const params = new SvelteURLSearchParams({
				page: String(page),
				page_size: String(pageSize)
			});
			if (search.trim()) params.set('q', search.trim());

			const res = await adminApiFetch(`/admin/users?${params.toString()}`);
			const envelope = (await res.json()) as {
				success: boolean;
				data?: UserListItem[];
				meta?: ListMeta;
				error?: { message?: string };
			};
			if (!res.ok || !envelope.success || !envelope.data) {
				throw new Error(envelope.error?.message ?? `HTTP ${res.status}`);
			}
			users = envelope.data;
			meta = envelope.meta ?? null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Gagal memuat pengguna';
		} finally {
			loading = false;
		}
	}

	function changePage(delta: number) {
		const next = page + delta;
		if (next < 1 || next > totalPages) return;
		page = next;
		loadUsers();
	}

	function applySearch(e: SubmitEvent) {
		e.preventDefault();
		page = 1;
		loadUsers();
	}

	onMount(loadUsers);
</script>

<div class="space-y-8">
	<div class="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
		<div>
			<h1 class="font-serif text-3xl leading-tight tracking-tight text-ink">Pengguna</h1>
			<p class="mt-1.5 text-sm text-ink/55">Kelola seluruh akun pelanggan KasKu.</p>
		</div>
		<form onsubmit={applySearch} class="flex items-center gap-2">
			<input
				type="search"
				bind:value={search}
				placeholder="Cari email / username…"
				class="rounded-[10px] border border-ink/25 bg-field px-4 py-2.5 text-sm text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-teal"
			/>
			<button
				type="submit"
				class="rounded-full bg-teal px-5 py-2.5 text-[13px] font-semibold text-card transition-colors hover:bg-ink"
			>
				Cari
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

	<div>
		<div
			class="grid grid-cols-[1.5fr_1fr_0.8fr_0.6fr_0.7fr_0.5fr] gap-4 border-b border-ink/25 py-3 text-[10.5px] font-semibold tracking-[0.12em] text-ink/50 uppercase"
		>
			<span>Email</span><span>Username</span><span>Status</span><span>Tier</span><span
				>Terdaftar</span
			><span></span>
		</div>

		{#if loading && users.length === 0}
			<div class="space-y-3 pt-4">
				{#each [0, 1, 2, 3, 4] as i (i)}
					<div class="h-6 animate-pulse rounded bg-ink/5"></div>
				{/each}
			</div>
		{:else if users.length === 0}
			<p class="py-8 text-center text-sm text-ink/45">Tidak ada pengguna ditemukan.</p>
		{:else}
			{#each users as u, i (u.id)}
				{@const isPro = u.subscription_tier?.toUpperCase() === 'PRO'}
				<div
					in:fly={{ y: 8, delay: i * 20, duration: 200 }}
					class="grid grid-cols-[1.5fr_1fr_0.8fr_0.6fr_0.7fr_0.5fr] items-baseline gap-4 border-b border-ink/8 py-4 text-[13px]"
				>
					<span class="truncate font-medium text-ink">{u.email}</span>
					<span class="truncate text-ink/60">{u.username}</span>
					<span>
						<span
							class="rounded-full border px-2 py-px text-[11px] {u.is_active
								? 'border-teal/30 text-teal'
								: 'border-clay/30 text-clay'}"
						>
							{u.is_active ? 'Aktif' : 'Suspended'}
						</span>
					</span>
					<span>
						<span
							class="rounded-full border px-2 py-px text-[11px] {isPro
								? 'border-teal/30 text-teal'
								: 'border-ink/15 text-ink/55'}"
						>
							{isPro ? 'Pro' : u.subscription_tier}
						</span>
					</span>
					<span class="text-ink/50">{formatDate(u.created_at)}</span>
					<a
						href={resolve(`/admin/users/${u.id}`)}
						class="justify-self-end font-semibold text-teal transition-colors hover:text-ink"
					>
						Detail →
					</a>
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
