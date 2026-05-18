<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import { adminApiFetch } from '$lib/api/admin_client';

	type UserDetail = {
		id: string;
		email: string;
		username: string;
		is_active: boolean;
		email_verified: boolean;
		subscription_tier: string;
		subscription_status: string;
		subscription_expires_at: string | null;
		created_at: string;
		last_login_at: string | null;
	};

	const userId = $derived(page.params.id);

	let user = $state<UserDetail | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let actionBusy = $state(false);
	let overrideTier = $state('PRO');
	let overrideDays = $state(30);

	const tierOptions = ['FREE', 'PRO', 'PREMIUM'];

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

	async function loadUser() {
		loading = true;
		error = null;
		try {
			const res = await adminApiFetch(`/admin/users/${userId}`);
			const envelope = (await res.json()) as {
				success: boolean;
				data?: UserDetail;
				error?: { message?: string };
			};
			if (!res.ok || !envelope.success || !envelope.data) {
				throw new Error(envelope.error?.message ?? `HTTP ${res.status}`);
			}
			user = envelope.data;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Gagal memuat detail pengguna';
		} finally {
			loading = false;
		}
	}

	async function postAction(path: string, body: unknown = {}): Promise<boolean> {
		actionBusy = true;
		error = null;
		try {
			const res = await adminApiFetch(`/admin/users/${userId}/${path}`, {
				method: 'POST',
				body: JSON.stringify(body)
			});
			const envelope = (await res.json()) as {
				success: boolean;
				error?: { message?: string };
			};
			if (!res.ok || !envelope.success) {
				throw new Error(envelope.error?.message ?? `HTTP ${res.status}`);
			}
			return true;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Aksi gagal';
			return false;
		} finally {
			actionBusy = false;
		}
	}

	async function suspend() {
		if (!confirm('Suspend pengguna ini? Akun tidak bisa login sampai diaktifkan kembali.')) return;
		if (await postAction('suspend')) await loadUser();
	}

	async function activate() {
		if (await postAction('activate')) await loadUser();
	}

	async function override() {
		if (
			!confirm(
				`Override subscription ke ${overrideTier} selama ${overrideDays} hari? Aksi ini tercatat di audit log.`
			)
		) {
			return;
		}
		const ok = await postAction('override-subscription', {
			tier: overrideTier,
			duration_days: overrideDays
		});
		if (ok) await loadUser();
	}

	onMount(loadUser);
</script>

<div class="space-y-8">
	<div class="flex items-center justify-between">
		<button
			type="button"
			onclick={() => goto(resolve('/admin/users'))}
			class="text-xs font-black text-gray-500 hover:text-[#0a2e31]"
		>
			← Kembali ke daftar
		</button>
		<button
			type="button"
			onclick={loadUser}
			disabled={loading}
			class="rounded-full border border-gray-200 bg-white px-4 py-2 text-[11px] font-bold text-[#0a2e31] hover:bg-gray-50 disabled:opacity-60"
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

	{#if loading && !user}
		<div class="h-48 animate-pulse rounded-[2rem] bg-gray-50"></div>
	{:else if user}
		<div class="space-y-6 rounded-[2.5rem] border border-gray-100 bg-white p-8 shadow-sm">
			<div class="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
				<div class="space-y-1">
					<h1 class="text-3xl font-black text-[#0a2e31]">{user.username}</h1>
					<p class="text-sm font-medium text-gray-500">{user.email}</p>
				</div>
				<div class="flex items-center gap-2">
					<span
						class="rounded-full px-3 py-1 text-[10px] font-black tracking-widest uppercase
						{user.is_active ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}"
					>
						{user.is_active ? 'Aktif' : 'Suspended'}
					</span>
					<span
						class="rounded-full px-3 py-1 text-[10px] font-black tracking-widest uppercase
						{user.email_verified ? 'bg-teal-50 text-teal-700' : 'bg-amber-50 text-amber-700'}"
					>
						{user.email_verified ? 'Terverifikasi' : 'Belum Verifikasi'}
					</span>
				</div>
			</div>

			<div class="grid grid-cols-2 gap-6 md:grid-cols-4">
				<div class="space-y-1">
					<p class="text-[10px] font-black tracking-widest text-gray-400 uppercase">Tier</p>
					<p class="text-sm font-bold text-[#0a2e31]">{user.subscription_tier}</p>
				</div>
				<div class="space-y-1">
					<p class="text-[10px] font-black tracking-widest text-gray-400 uppercase">Status Subscription</p>
					<p class="text-sm font-bold text-[#0a2e31]">{user.subscription_status}</p>
				</div>
				<div class="space-y-1">
					<p class="text-[10px] font-black tracking-widest text-gray-400 uppercase">Aktif Hingga</p>
					<p class="text-sm font-bold text-[#0a2e31]">{formatDate(user.subscription_expires_at)}</p>
				</div>
				<div class="space-y-1">
					<p class="text-[10px] font-black tracking-widest text-gray-400 uppercase">Terdaftar</p>
					<p class="text-sm font-bold text-[#0a2e31]">{formatDate(user.created_at)}</p>
				</div>
				<div class="space-y-1">
					<p class="text-[10px] font-black tracking-widest text-gray-400 uppercase">Login Terakhir</p>
					<p class="text-sm font-bold text-[#0a2e31]">{formatDate(user.last_login_at)}</p>
				</div>
				<div class="space-y-1 md:col-span-2">
					<p class="text-[10px] font-black tracking-widest text-gray-400 uppercase">User ID</p>
					<p class="font-mono text-xs text-[#0a2e31]">{user.id}</p>
				</div>
			</div>
		</div>

		<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
			<div class="space-y-4 rounded-[2rem] border border-gray-100 bg-white p-6 shadow-sm">
				<h3 class="text-sm font-black text-[#0a2e31]">Status Akun</h3>
				<p class="text-xs text-gray-500">
					Suspend mencegah login dan akses API; aktivasi mengembalikan akses penuh.
				</p>
				<div class="flex items-center gap-2">
					{#if user.is_active}
						<button
							type="button"
							onclick={suspend}
							disabled={actionBusy}
							class="rounded-full bg-red-600 px-4 py-2 text-[11px] font-black tracking-widest text-white uppercase hover:bg-red-700 disabled:opacity-60"
						>
							Suspend
						</button>
					{:else}
						<button
							type="button"
							onclick={activate}
							disabled={actionBusy}
							class="rounded-full bg-green-600 px-4 py-2 text-[11px] font-black tracking-widest text-white uppercase hover:bg-green-700 disabled:opacity-60"
						>
							Aktifkan
						</button>
					{/if}
				</div>
			</div>

			<div class="space-y-4 rounded-[2rem] border border-gray-100 bg-white p-6 shadow-sm">
				<h3 class="text-sm font-black text-[#0a2e31]">Override Subscription</h3>
				<p class="text-xs text-gray-500">
					Ubah tier secara manual; berguna untuk kompensasi atau promo. Tercatat di audit log.
				</p>
				<div class="grid grid-cols-2 gap-3">
					<label class="space-y-1 text-xs font-bold text-gray-600">
						Tier
						<select
							bind:value={overrideTier}
							class="w-full rounded-full border border-gray-200 px-3 py-2 text-xs font-bold text-[#0a2e31] focus:outline-none focus:ring-2 focus:ring-teal-500/40"
						>
							{#each tierOptions as t (t)}
								<option value={t}>{t}</option>
							{/each}
						</select>
					</label>
					<label class="space-y-1 text-xs font-bold text-gray-600">
						Durasi (hari)
						<input
							type="number"
							bind:value={overrideDays}
							min="1"
							max="365"
							class="w-full rounded-full border border-gray-200 px-3 py-2 text-xs font-bold text-[#0a2e31] focus:outline-none focus:ring-2 focus:ring-teal-500/40"
						/>
					</label>
				</div>
				<button
					type="button"
					onclick={override}
					disabled={actionBusy}
					class="rounded-full bg-[#0a2e31] px-4 py-2 text-[11px] font-black tracking-widest text-white uppercase hover:bg-[#143f43] disabled:opacity-60"
				>
					Override Sekarang
				</button>
			</div>
		</div>
	{/if}
</div>
