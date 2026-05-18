<script lang="ts">
	import { onMount } from 'svelte';
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
			const params = new URLSearchParams({
				page: String(page),
				page_size: String(pageSize)
			});
			if (search.trim()) params.set('q', search.trim());

			const res = await adminApiFetch(`/admin/users?${params.toString()}`);
			const envelope = (await res.json()) as {
				success: boolean;
				data?: { items: UserListItem[]; meta: ListMeta };
				error?: { message?: string };
			};
			if (!res.ok || !envelope.success || !envelope.data) {
				throw new Error(envelope.error?.message ?? `HTTP ${res.status}`);
			}
			users = envelope.data.items;
			meta = envelope.data.meta;
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
		<div class="space-y-1">
			<h1 class="text-3xl font-black text-[#0a2e31]">Pengguna</h1>
			<p class="font-medium text-gray-500">Kelola seluruh akun pelanggan KasKu.</p>
		</div>
		<form onsubmit={applySearch} class="flex items-center gap-2">
			<input
				type="search"
				bind:value={search}
				placeholder="Cari email / username…"
				class="rounded-full border border-gray-200 bg-white px-4 py-2 text-xs font-bold text-[#0a2e31] placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-teal-500/40"
			/>
			<button
				type="submit"
				class="rounded-full bg-[#0a2e31] px-4 py-2 text-[11px] font-black tracking-widest text-white uppercase transition-colors hover:bg-[#143f43]"
			>
				Cari
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
					<th class="px-6 py-4">Email</th>
					<th class="px-6 py-4">Username</th>
					<th class="px-6 py-4">Status</th>
					<th class="px-6 py-4">Tier</th>
					<th class="px-6 py-4">Terdaftar</th>
					<th class="px-6 py-4"></th>
				</tr>
			</thead>
			<tbody class="divide-y divide-gray-50">
				{#if loading && users.length === 0}
					{#each [0, 1, 2, 3, 4] as i (i)}
						<tr><td colspan="6" class="h-12 animate-pulse bg-gray-50/50"></td></tr>
					{/each}
				{:else if users.length === 0}
					<tr>
						<td colspan="6" class="px-6 py-10 text-center text-xs font-bold text-gray-400">
							Tidak ada pengguna ditemukan.
						</td>
					</tr>
				{:else}
					{#each users as u, i (u.id)}
						<tr in:fly={{ y: 8, delay: i * 20, duration: 200 }} class="transition-colors hover:bg-gray-50/60">
							<td class="px-6 py-4 text-sm font-bold text-[#0a2e31]">{u.email}</td>
							<td class="px-6 py-4 text-xs font-medium text-gray-600">{u.username}</td>
							<td class="px-6 py-4">
								<span
									class="rounded-full px-2 py-0.5 text-[9px] font-black tracking-widest uppercase
									{u.is_active ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}"
								>
									{u.is_active ? 'Aktif' : 'Suspended'}
								</span>
							</td>
							<td class="px-6 py-4">
								<span class="text-[10px] font-black tracking-widest text-teal-700 uppercase">
									{u.subscription_tier}
								</span>
							</td>
							<td class="px-6 py-4 text-xs font-medium text-gray-500">{formatDate(u.created_at)}</td>
							<td class="px-6 py-4 text-right">
								<a
									href={resolve(`/admin/users/${u.id}`)}
									class="text-xs font-black text-[#217b84] hover:underline"
								>
									Detail →
								</a>
							</td>
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
