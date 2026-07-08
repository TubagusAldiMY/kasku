<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { adminApiFetch } from '$lib/api/admin_client';
	import { adminAuth, type AdminUser } from '$lib/stores/admin_auth.svelte';

	type LoginResponseData = {
		access_token: string;
		token_type: string;
		expires_in: number;
		admin: AdminUser;
	};

	let username = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state<string | null>(null);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		loading = true;
		error = null;

		try {
			const res = await adminApiFetch('/admin/auth/login', {
				method: 'POST',
				skipAuth: true,
				body: JSON.stringify({ username, password })
			});

			const result = (await res.json()) as
				| { success: true; data: LoginResponseData }
				| { success: false; error?: { message?: string } };

			if (!res.ok || !('data' in result) || !result.success) {
				error = ('error' in result && result.error?.message) || 'Login admin gagal.';
				return;
			}

			adminAuth.setSession(result.data.access_token, result.data.expires_in, result.data.admin);
			goto(resolve('/admin/dashboard'));
		} catch (err) {
			console.error('Admin login error:', err);
			error = 'Tidak dapat menghubungi server admin.';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Login Admin · KasKu</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-ink px-6 py-16">
	<div class="w-full max-w-sm">
		<div class="mb-9 text-center">
			<span class="inline-flex items-baseline gap-2.5">
				<span class="font-serif text-[30px] leading-none tracking-tight text-card"
					>Kas<em class="text-mint">Ku</em></span
				>
				<span
					class="rounded-full bg-mint/15 px-2.5 py-0.5 text-[10px] font-semibold tracking-[0.14em] text-mint uppercase"
				>
					Admin
				</span>
			</span>
			<p class="mt-4 text-[13px] text-card/55">
				Akses dibatasi — semua aksi tercatat di audit log.
			</p>
		</div>

		<div class="rounded-[18px] bg-card p-8">
			{#if error}
				<div class="mb-5 flex items-start gap-2.5 rounded-xl border border-clay/25 bg-clay/5 p-4">
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

			<form onsubmit={handleSubmit} class="space-y-5">
				<div>
					<label
						for="admin-username"
						class="mb-2 block text-[11px] font-semibold tracking-[0.12em] text-ink/50 uppercase"
					>
						Username
					</label>
					<input
						id="admin-username"
						type="text"
						required
						autocomplete="username"
						bind:value={username}
						class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-teal"
					/>
				</div>

				<div>
					<label
						for="admin-password"
						class="mb-2 block text-[11px] font-semibold tracking-[0.12em] text-ink/50 uppercase"
					>
						Password
					</label>
					<input
						id="admin-password"
						type="password"
						required
						autocomplete="current-password"
						bind:value={password}
						class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none placeholder:text-ink/30 focus:border-teal"
					/>
				</div>

				<button
					type="submit"
					disabled={loading}
					class="flex w-full items-center justify-center gap-2.5 rounded-full bg-ink py-3.5 text-sm font-semibold text-card transition-colors hover:bg-teal disabled:opacity-70"
				>
					{#if loading}
						<span class="h-4 w-4 animate-spin rounded-full border-2 border-card/40 border-t-card"
						></span>
					{/if}
					{loading ? 'Memproses…' : 'Masuk Panel Admin'}
				</button>
			</form>
		</div>
	</div>
</div>
