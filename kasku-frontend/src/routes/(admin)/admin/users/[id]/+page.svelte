<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import { adminApiFetch } from '$lib/api/admin_client';
	import { adminAuth } from '$lib/stores/admin_auth.svelte';

	const isSuperAdmin = $derived(adminAuth.admin?.role === 'SUPER_ADMIN');

	type UserDetail = {
		id: string;
		email: string;
		username: string;
		is_active: boolean;
		email_verified: boolean;
		subscription_tier: string;
		subscription_status: string;
		subscription_ends_at: string | null;
		created_at: string;
		last_login_at: string | null;
	};

	const userId = $derived(page.params.id);

	let user = $state<UserDetail | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let actionBusy = $state(false);
	let suspendReason = $state('');
	let activateReason = $state('');
	let overridePlanName = $state('PRO');
	let overrideReason = $state('');
	let deleteConfirm = $state('');
	let deleteReason = $state('');
	let deleted = $state(false);

	const DELETE_PHRASE = 'HAPUS';

	const tierOptions = ['FREE', 'BASIC', 'PRO'];

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
		if (suspendReason.trim().length < 3) {
			error = 'Alasan suspend minimal 3 karakter.';
			return;
		}
		if (!confirm('Suspend pengguna ini? Akun tidak bisa login sampai diaktifkan kembali.')) return;
		if (await postAction('suspend', { reason: suspendReason.trim() })) {
			suspendReason = '';
			await loadUser();
		}
	}

	async function activate() {
		if (activateReason.trim().length < 3) {
			error = 'Alasan aktivasi minimal 3 karakter.';
			return;
		}
		if (await postAction('activate', { reason: activateReason.trim() })) {
			activateReason = '';
			await loadUser();
		}
	}

	async function override() {
		if (overrideReason.trim().length < 3) {
			error = 'Alasan override minimal 3 karakter.';
			return;
		}
		if (!confirm(`Override subscription ke ${overridePlanName}? Aksi ini tercatat di audit log.`)) {
			return;
		}
		const ok = await postAction('override-subscription', {
			plan_name: overridePlanName,
			reason: overrideReason.trim()
		});
		if (ok) {
			overrideReason = '';
			await loadUser();
		}
	}

	async function deleteUser() {
		if (deleteReason.trim().length < 3) {
			error = 'Alasan penghapusan minimal 3 karakter.';
			return;
		}
		if (deleteConfirm.trim() !== DELETE_PHRASE) {
			error = `Ketik "${DELETE_PHRASE}" untuk konfirmasi.`;
			return;
		}
		if (
			!confirm(
				'Hapus permanen pengguna ini? Data akan di-anonimkan dan akun tidak bisa dipulihkan.'
			)
		)
			return;
		if (await postAction('delete', { reason: deleteReason.trim() })) {
			deleted = true;
			setTimeout(() => goto(resolve('/admin/users')), 1200);
		}
	}

	onMount(loadUser);
</script>

<div class="space-y-8">
	<div class="flex items-center justify-between">
		<button
			type="button"
			onclick={() => goto(resolve('/admin/users'))}
			class="text-[13px] font-semibold text-ink/55 transition-colors hover:text-ink"
		>
			← Kembali ke daftar
		</button>
		<button
			type="button"
			onclick={loadUser}
			disabled={loading}
			class="rounded-full border border-ink/20 px-4 py-2 text-[13px] font-semibold text-ink transition-colors hover:border-ink/40 disabled:opacity-60"
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

	{#if loading && !user}
		<div class="h-48 animate-pulse rounded-2xl bg-ink/5"></div>
	{:else if user}
		<div class="border-b border-ink/10 pb-8">
			<div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
				<div>
					<h1 class="font-serif text-3xl leading-tight tracking-tight text-ink">{user.username}</h1>
					<p class="mt-1.5 text-sm text-ink/55">{user.email}</p>
				</div>
				<div class="flex items-center gap-2">
					<span
						class="rounded-full border px-2.5 py-0.5 text-[11px] {user.is_active
							? 'border-teal/30 text-teal'
							: 'border-clay/30 text-clay'}"
					>
						{user.is_active ? 'Aktif' : 'Suspended'}
					</span>
					<span
						class="rounded-full border px-2.5 py-0.5 text-[11px] {user.email_verified
							? 'border-teal/30 text-teal'
							: 'border-gold/30 text-gold'}"
					>
						{user.email_verified ? 'Terverifikasi' : 'Belum verifikasi'}
					</span>
				</div>
			</div>

			<div class="mt-7 grid grid-cols-2 gap-x-6 gap-y-6 md:grid-cols-4">
				<div>
					<p class="text-[10.5px] font-semibold tracking-[0.12em] text-ink/45 uppercase">Tier</p>
					<p class="mt-1 text-sm font-medium text-ink">{user.subscription_tier}</p>
				</div>
				<div>
					<p class="text-[10.5px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
						Status subscription
					</p>
					<p class="mt-1 text-sm font-medium text-ink">{user.subscription_status}</p>
				</div>
				<div>
					<p class="text-[10.5px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
						Aktif hingga
					</p>
					<p class="mt-1 text-sm font-medium text-ink">
						{formatDate(user.subscription_ends_at)}
					</p>
				</div>
				<div>
					<p class="text-[10.5px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
						Terdaftar
					</p>
					<p class="mt-1 text-sm font-medium text-ink">{formatDate(user.created_at)}</p>
				</div>
				<div>
					<p class="text-[10.5px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
						Login terakhir
					</p>
					<p class="mt-1 text-sm font-medium text-ink">{formatDate(user.last_login_at)}</p>
				</div>
				<div class="md:col-span-2">
					<p class="text-[10.5px] font-semibold tracking-[0.12em] text-ink/45 uppercase">User ID</p>
					<p class="mt-1 font-mono text-xs text-ink/70">{user.id}</p>
				</div>
			</div>
		</div>

		<div class="grid grid-cols-1 gap-8 md:grid-cols-2">
			{#if isSuperAdmin}
			<div class="rounded-2xl border border-ink/12 bg-card p-6">
				<h3 class="text-[13px] font-semibold text-ink">Status akun</h3>
				<p class="mt-1.5 text-[13px] text-ink/55">
					Suspend mencegah login dan akses API; aktivasi mengembalikan akses penuh.
				</p>
				{#if user.is_active}
					<label
						class="mt-4 block text-[11px] font-semibold tracking-[0.1em] text-ink/50 uppercase"
					>
						Alasan suspend
						<input
							type="text"
							bind:value={suspendReason}
							placeholder="Contoh: Pelanggaran ToS pasal 3"
							class="mt-2 w-full rounded-[10px] border border-ink/25 bg-field px-3.5 py-2.5 text-sm font-normal text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-clay"
						/>
					</label>
					<button
						type="button"
						onclick={suspend}
						disabled={actionBusy || suspendReason.trim().length < 3}
						class="mt-4 rounded-full border border-clay/25 px-5 py-2.5 text-[13px] font-semibold text-clay transition-colors hover:bg-clay/5 disabled:opacity-40"
					>
						Suspend
					</button>
				{:else}
					<label
						class="mt-4 block text-[11px] font-semibold tracking-[0.1em] text-ink/50 uppercase"
					>
						Alasan aktivasi
						<input
							type="text"
							bind:value={activateReason}
							placeholder="Contoh: Masalah sudah diselesaikan"
							class="mt-2 w-full rounded-[10px] border border-ink/25 bg-field px-3.5 py-2.5 text-sm font-normal text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-teal"
						/>
					</label>
					<button
						type="button"
						onclick={activate}
						disabled={actionBusy || activateReason.trim().length < 3}
						class="mt-4 rounded-full bg-teal px-5 py-2.5 text-[13px] font-semibold text-card transition-colors hover:bg-ink disabled:opacity-40"
					>
						Aktifkan
					</button>
				{/if}
			</div>
			{/if}

			{#if isSuperAdmin}
			<div class="rounded-2xl border border-ink/12 bg-card p-6">
				<h3 class="text-[13px] font-semibold text-ink">Override subscription</h3>
				<p class="mt-1.5 text-[13px] text-ink/55">
					Ubah tier secara manual; berguna untuk kompensasi atau promo. Tercatat di audit log.
				</p>
				<div class="mt-4 grid grid-cols-1 gap-3.5">
					<label class="text-[11px] font-semibold tracking-[0.1em] text-ink/50 uppercase">
						Tier baru
						<select
							bind:value={overridePlanName}
							class="mt-2 w-full rounded-[10px] border border-ink/25 bg-field px-3.5 py-2.5 text-sm font-normal text-ink transition-colors outline-none focus:border-teal"
						>
							{#each tierOptions as t (t)}
								<option value={t}>{t}</option>
							{/each}
						</select>
					</label>
					<label class="text-[11px] font-semibold tracking-[0.1em] text-ink/50 uppercase">
						Alasan override
						<input
							type="text"
							bind:value={overrideReason}
							placeholder="Contoh: Kompensasi gangguan layanan"
							class="mt-2 w-full rounded-[10px] border border-ink/25 bg-field px-3.5 py-2.5 text-sm font-normal text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-teal"
						/>
					</label>
				</div>
				<button
					type="button"
					onclick={override}
					disabled={actionBusy || overrideReason.trim().length < 3}
					class="mt-4 rounded-full bg-teal px-5 py-2.5 text-[13px] font-semibold text-card transition-colors hover:bg-ink disabled:opacity-40"
				>
					Override sekarang
				</button>
			</div>
			{/if}
		</div>

		{#if isSuperAdmin}
			<div class="rounded-2xl border border-clay/30 bg-clay/[0.03] p-6">
				<h3 class="text-[13px] font-semibold text-clay">Zona berbahaya</h3>
				<p class="mt-1.5 text-[13px] text-ink/55">
					Menghapus pengguna akan meng-anonimkan seluruh data pribadinya secara permanen. Aksi ini
					tidak bisa dibatalkan dan tercatat di audit log.
				</p>
				{#if deleted}
					<p class="mt-4 text-[13px] font-semibold text-teal">
						Pengguna dihapus. Mengalihkan ke daftar…
					</p>
				{:else}
					<label class="mt-4 block text-[11px] font-semibold tracking-[0.1em] text-ink/50 uppercase">
						Alasan penghapusan
						<input
							type="text"
							bind:value={deleteReason}
							placeholder="Contoh: Permintaan penghapusan data (UU PDP)"
							class="mt-2 w-full rounded-[10px] border border-clay/30 bg-field px-3.5 py-2.5 text-sm font-normal text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-clay"
						/>
					</label>
					<label class="mt-4 block text-[11px] font-semibold tracking-[0.1em] text-ink/50 uppercase">
						Ketik <span class="font-mono text-clay">{DELETE_PHRASE}</span> untuk konfirmasi
						<input
							type="text"
							bind:value={deleteConfirm}
							placeholder={DELETE_PHRASE}
							autocomplete="off"
							class="mt-2 w-full max-w-xs rounded-[10px] border border-clay/30 bg-field px-3.5 py-2.5 text-sm font-normal text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-clay"
						/>
					</label>
					<button
						type="button"
						onclick={deleteUser}
						disabled={actionBusy ||
							deleteConfirm.trim() !== DELETE_PHRASE ||
							deleteReason.trim().length < 3}
						class="mt-4 rounded-full bg-clay px-5 py-2.5 text-[13px] font-semibold text-card transition-colors hover:bg-ink disabled:opacity-40"
					>
						{actionBusy ? 'Menghapus…' : 'Hapus user'}
					</button>
				{/if}
			</div>
		{/if}
	{/if}
</div>
