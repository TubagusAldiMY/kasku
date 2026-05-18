<script lang="ts">
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import { resolve } from '$app/paths';
	import { adminApiFetch } from '$lib/api/admin_client';

	type DashboardStats = {
		total_users: number;
		total_active_users: number;
		new_users_last_7_days: number;
		tier_distribution: Record<string, number>;
		mrr_idr: number;
		churn_rate_30d_pct: number;
	};

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

	let stats = $state<DashboardStats | null>(null);
	let recentUsers = $state<UserListItem[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(val);
	}

	function formatNumber(val: number) {
		return new Intl.NumberFormat('id-ID').format(val);
	}

	function relativeDate(iso: string): string {
		try {
			const dt = new Date(iso);
			return dt.toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' });
		} catch {
			return iso;
		}
	}

	async function loadDashboard() {
		loading = true;
		error = null;
		try {
			const [statsRes, usersRes] = await Promise.all([
				adminApiFetch('/admin/stats/dashboard'),
				adminApiFetch('/admin/users?page=1&page_size=5')
			]);

			const statsEnvelope = (await statsRes.json()) as {
				success: boolean;
				data?: DashboardStats;
				error?: { message?: string };
			};
			if (!statsRes.ok || !statsEnvelope.success || !statsEnvelope.data) {
				throw new Error(statsEnvelope.error?.message ?? `stats HTTP ${statsRes.status}`);
			}
			stats = statsEnvelope.data;

			const usersEnvelope = (await usersRes.json()) as {
				success: boolean;
				data?: { items: UserListItem[]; meta?: unknown };
				error?: { message?: string };
			};
			if (!usersRes.ok || !usersEnvelope.success) {
				throw new Error(usersEnvelope.error?.message ?? `users HTTP ${usersRes.status}`);
			}
			recentUsers = usersEnvelope.data?.items ?? [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Gagal memuat data dashboard';
		} finally {
			loading = false;
		}
	}

	onMount(loadDashboard);

	const tierEntries = $derived(stats ? Object.entries(stats.tier_distribution) : []);
</script>

<div class="animate-in fade-in space-y-10 duration-700">
	<div class="flex items-end justify-between">
		<div class="space-y-1">
			<h1 class="text-3xl font-black text-[#0a2e31]">Ringkasan Sistem</h1>
			<p class="font-medium text-gray-500">
				Pantau performa global dan kesehatan infrastruktur KasKu.
			</p>
		</div>
		<button
			type="button"
			onclick={loadDashboard}
			disabled={loading}
			class="rounded-full border border-gray-200 bg-white px-4 py-2 text-[11px] font-bold text-[#0a2e31] transition-colors hover:bg-gray-50 disabled:opacity-60"
		>
			{loading ? 'Memuat…' : 'Muat Ulang'}
		</button>
	</div>

	{#if error}
		<div
			class="rounded-2xl border border-red-100 bg-red-50 px-5 py-4 text-xs font-bold text-red-700"
		>
			{error}
		</div>
	{/if}

	<!-- Stats Grid -->
	<div class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4">
		{#each [{ label: 'Total Pengguna', value: stats ? formatNumber(stats.total_users) : '…' }, { label: 'Pengguna Aktif', value: stats ? formatNumber(stats.total_active_users) : '…' }, { label: 'Baru (7 hari)', value: stats ? formatNumber(stats.new_users_last_7_days) : '…' }, { label: 'MRR', value: stats ? formatCurrency(stats.mrr_idr) : '…' }] as item, i (item.label)}
			<div
				in:fly={{ y: 20, delay: i * 80, duration: 400 }}
				class="space-y-3 rounded-[2rem] border border-gray-100 bg-white p-6 shadow-sm transition-all hover:shadow-lg"
			>
				<p class="text-[11px] font-black tracking-widest text-gray-400 uppercase">{item.label}</p>
				<p class="text-2xl font-black text-[#0a2e31] tabular-nums">{item.value}</p>
			</div>
		{/each}
	</div>

	<div class="grid grid-cols-1 gap-8 lg:grid-cols-3">
		<!-- Recent users -->
		<div
			class="flex flex-col overflow-hidden rounded-[2.5rem] border border-gray-100 bg-white shadow-sm lg:col-span-2"
		>
			<div class="flex items-center justify-between border-b border-gray-50 p-8">
				<div class="space-y-1">
					<h3 class="text-lg font-black text-[#0a2e31]">Registrasi Terbaru</h3>
					<p class="text-[10px] font-bold tracking-widest text-gray-400 uppercase">
						5 pengguna terakhir
					</p>
				</div>
				<a href={resolve('/admin/users')} class="text-xs font-bold text-[#217b84] hover:underline"
					>Kelola Semua →</a
				>
			</div>
			<div class="divide-y divide-gray-50">
				{#if loading && recentUsers.length === 0}
					{#each [0, 1, 2] as i (i)}
						<div class="h-16 animate-pulse bg-gray-50"></div>
					{/each}
				{:else if recentUsers.length === 0}
					<div class="px-8 py-10 text-center text-xs font-bold text-gray-400">
						Belum ada pengguna terdaftar.
					</div>
				{:else}
					{#each recentUsers as u (u.id)}
						<a
							href={resolve(`/admin/users/${u.id}`)}
							class="flex items-center justify-between gap-4 px-8 py-5 transition-colors hover:bg-gray-50"
						>
							<div class="min-w-0">
								<p class="truncate text-sm font-black text-[#0a2e31]">{u.username}</p>
								<p class="truncate text-xs font-medium text-gray-500">{u.email}</p>
							</div>
							<div class="flex flex-col items-end gap-1">
								<span
									class="rounded-full px-2 py-0.5 text-[9px] font-black tracking-widest uppercase
									{u.is_active ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}"
								>
									{u.is_active ? 'Aktif' : 'Suspended'}
								</span>
								<span class="text-[10px] font-bold text-gray-400">{relativeDate(u.created_at)}</span
								>
							</div>
						</a>
					{/each}
				{/if}
			</div>
		</div>

		<!-- Tier distribution -->
		<div class="space-y-6 rounded-[2.5rem] border border-gray-100 bg-white p-8 shadow-sm">
			<div class="space-y-1">
				<h3 class="text-lg font-black text-[#0a2e31]">Distribusi Tier</h3>
				<p class="text-[10px] font-bold tracking-widest text-gray-400 uppercase">
					Snapshot saat ini
				</p>
			</div>

			{#if loading && tierEntries.length === 0}
				<div class="h-32 animate-pulse rounded-2xl bg-gray-50"></div>
			{:else if tierEntries.length === 0}
				<p class="text-xs font-bold text-gray-400">Belum ada subscription tercatat.</p>
			{:else}
				<div class="space-y-3">
					{#each tierEntries as [tier, count] (tier)}
						<div class="flex items-center justify-between">
							<span class="text-xs font-black tracking-widest text-[#0a2e31] uppercase">{tier}</span
							>
							<span class="rounded-full bg-teal-50 px-3 py-1 text-xs font-black text-teal-700"
								>{formatNumber(count)}</span
							>
						</div>
					{/each}
				</div>
			{/if}

			{#if stats}
				<div class="rounded-2xl bg-[#0a2e31] p-5 text-white">
					<p class="text-[10px] font-bold tracking-widest text-teal-300/80 uppercase">
						Churn 30 hari
					</p>
					<p class="mt-1 text-2xl font-black tabular-nums">
						{stats.churn_rate_30d_pct.toFixed(1)}%
					</p>
				</div>
			{/if}
		</div>
	</div>
</div>
