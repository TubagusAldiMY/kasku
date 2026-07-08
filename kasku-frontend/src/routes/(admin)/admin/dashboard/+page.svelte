<script lang="ts">
	import { onMount } from 'svelte';
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
				data?: UserListItem[];
				error?: { message?: string };
			};
			if (!usersRes.ok || !usersEnvelope.success) {
				throw new Error(usersEnvelope.error?.message ?? `users HTTP ${usersRes.status}`);
			}
			recentUsers = usersEnvelope.data ?? [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Gagal memuat data dashboard';
		} finally {
			loading = false;
		}
	}

	onMount(loadDashboard);

	const tierEntries = $derived(stats ? Object.entries(stats.tier_distribution) : []);
</script>

<div class="space-y-10">
	<div class="flex items-end justify-between gap-4">
		<div>
			<h1 class="font-serif text-3xl leading-tight tracking-tight text-ink">Ringkasan sistem</h1>
			<p class="mt-1.5 text-sm text-ink/55">
				Pantau performa global dan kesehatan infrastruktur KasKu.
			</p>
		</div>
		<button
			type="button"
			onclick={loadDashboard}
			disabled={loading}
			class="shrink-0 rounded-full border border-ink/20 px-4 py-2 text-[13px] font-semibold text-ink transition-colors hover:border-ink/40 disabled:opacity-60"
		>
			{loading ? 'Memuat…' : 'Muat ulang'}
		</button>
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

	<!-- Stat header -->
	<section class="grid grid-cols-2 border-b border-ink/10 pb-8 lg:grid-cols-4">
		{#each [{ label: 'Total pengguna', value: stats ? formatNumber(stats.total_users) : '…', delta: stats ? `+${formatNumber(stats.new_users_last_7_days)} minggu ini` : '', deltaCls: 'text-teal' }, { label: 'Aktif 7 hari', value: stats ? formatNumber(stats.total_active_users) : '…', delta: stats && stats.total_users > 0 ? `${Math.round((stats.total_active_users / stats.total_users) * 100)}% dari total` : '', deltaCls: 'text-ink/50' }, { label: 'Baru (7 hari)', value: stats ? formatNumber(stats.new_users_last_7_days) : '…', delta: 'Registrasi terbaru', deltaCls: 'text-ink/50' }, { label: 'MRR', value: stats ? formatCurrency(stats.mrr_idr) : '…', delta: stats ? `Churn ${stats.churn_rate_30d_pct.toFixed(1)}% / 30 hari` : '', deltaCls: 'text-ink/50' }] as item, i (item.label)}
			<div
				class="{i % 2 === 1 ? 'border-l border-ink/12 pl-6' : 'pr-6'} {i < 2
					? 'pb-6 lg:pb-0'
					: ''} lg:pr-6 lg:pl-6 {i > 0 ? 'lg:border-l lg:border-ink/12' : ''} {i === 0
					? 'lg:pl-0'
					: ''}"
			>
				<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
					{item.label}
				</p>
				<p class="mt-2 font-serif text-4xl leading-none text-ink tabular-nums">{item.value}</p>
				{#if item.delta}
					<p class="mt-1.5 text-xs {item.deltaCls}">{item.delta}</p>
				{/if}
			</div>
		{/each}
	</section>

	<section class="grid gap-12 lg:grid-cols-[1.5fr_1fr] lg:gap-16">
		<!-- Recent users -->
		<div>
			<div class="mb-1 flex items-baseline justify-between">
				<p class="text-[13px] font-semibold text-ink">Pengguna terbaru</p>
				<a
					href={resolve('/admin/users')}
					class="text-xs font-semibold text-teal transition-colors hover:text-ink"
				>
					Kelola semua →
				</a>
			</div>

			<div
				class="grid grid-cols-[1.6fr_0.7fr_0.9fr_0.6fr] gap-3 border-b border-ink/25 py-3 text-[10.5px] font-semibold tracking-[0.12em] text-ink/50 uppercase"
			>
				<span>Email</span><span>Paket</span><span>Status</span><span>Daftar</span>
			</div>

			{#if loading && recentUsers.length === 0}
				<div class="space-y-3 pt-4">
					{#each [0, 1, 2] as i (i)}
						<div class="h-6 animate-pulse rounded bg-ink/5"></div>
					{/each}
				</div>
			{:else if recentUsers.length === 0}
				<p class="pt-6 text-sm text-ink/45">Belum ada pengguna terdaftar.</p>
			{:else}
				{#each recentUsers as u (u.id)}
					{@const isPro = u.subscription_tier?.toUpperCase() === 'PRO'}
					<a
						href={resolve(`/admin/users/${u.id}`)}
						class="grid grid-cols-[1.6fr_0.7fr_0.9fr_0.6fr] items-baseline gap-3 border-b border-ink/8 py-4 text-[13px] transition-colors hover:bg-ink/[0.02]"
					>
						<span class="truncate font-medium text-ink">{u.email}</span>
						<span
							class="justify-self-start rounded-full border px-2 py-px text-[11px] {isPro
								? 'border-teal/30 text-teal'
								: 'border-ink/15 text-ink/55'}"
						>
							{isPro ? 'Pro' : 'Gratis'}
						</span>
						{#if !u.is_active}
							<span class="font-semibold text-clay">Ditangguhkan</span>
						{:else if u.email_verified}
							<span class="font-semibold text-teal">Terverifikasi</span>
						{:else}
							<span class="font-semibold text-gold">Menunggu email</span>
						{/if}
						<span class="text-ink/50">{relativeDate(u.created_at)}</span>
					</a>
				{/each}
			{/if}
		</div>

		<!-- Tier distribution -->
		<div>
			<p class="mb-4 text-[13px] font-semibold text-ink">Distribusi tier</p>

			{#if loading && tierEntries.length === 0}
				<div class="space-y-3">
					{#each [0, 1] as i (i)}
						<div class="h-6 animate-pulse rounded bg-ink/5"></div>
					{/each}
				</div>
			{:else if tierEntries.length === 0}
				<p class="text-sm text-ink/45">Belum ada subscription tercatat.</p>
			{:else}
				<div>
					{#each tierEntries as [tier, count] (tier)}
						<div class="flex items-baseline justify-between border-b border-ink/8 py-3 text-[13px]">
							<span class="font-medium text-ink">{tier}</span>
							<span class="font-semibold text-ink/70 tabular-nums">{formatNumber(count)}</span>
						</div>
					{/each}
				</div>
			{/if}

			{#if stats}
				<div class="mt-6 rounded-xl bg-ink p-5 text-card">
					<p class="text-[11px] font-semibold tracking-[0.12em] text-mint uppercase">
						Churn 30 hari
					</p>
					<p class="mt-1.5 font-serif text-3xl leading-none tabular-nums">
						{stats.churn_rate_30d_pct.toFixed(1)}%
					</p>
				</div>
			{/if}
		</div>
	</section>
</div>
