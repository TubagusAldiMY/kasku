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

<div class="flex min-h-screen items-center justify-center bg-[#0a2e31] px-6">
	<div class="w-full max-w-sm space-y-8 rounded-3xl bg-white p-10 shadow-2xl">
		<div class="space-y-2 text-center">
			<div class="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-red-500">
				<svg
					class="h-6 w-6 text-white"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					aria-hidden="true"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
					/>
				</svg>
			</div>
			<h1 class="text-xl font-black text-[#0a2e31]">Administrator Panel</h1>
			<p class="text-xs font-medium text-gray-500">
				Akses dibatasi — semua aksi tercatat di audit log.
			</p>
		</div>

		{#if error}
			<div
				class="rounded-xl border border-red-100 bg-red-50 px-4 py-3 text-xs font-bold text-red-700"
			>
				{error}
			</div>
		{/if}

		<form onsubmit={handleSubmit} class="space-y-5">
			<div class="space-y-2">
				<label
					for="admin-username"
					class="block text-[11px] font-black tracking-widest text-gray-400 uppercase"
				>
					Username
				</label>
				<input
					id="admin-username"
					type="text"
					required
					autocomplete="username"
					bind:value={username}
					class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-4 py-3.5 text-sm font-bold text-[#0a2e31] outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
				/>
			</div>

			<div class="space-y-2">
				<label
					for="admin-password"
					class="block text-[11px] font-black tracking-widest text-gray-400 uppercase"
				>
					Password
				</label>
				<input
					id="admin-password"
					type="password"
					required
					autocomplete="current-password"
					bind:value={password}
					class="w-full rounded-2xl border border-gray-100 bg-gray-50 px-4 py-3.5 text-sm font-bold text-[#0a2e31] outline-none focus:border-[#217b84] focus:ring-4 focus:ring-teal-50"
				/>
			</div>

			<button
				type="submit"
				disabled={loading}
				class="w-full rounded-2xl bg-red-500 py-3.5 text-xs font-black tracking-widest text-white uppercase shadow-lg transition-all hover:bg-red-600 active:scale-[0.98] disabled:opacity-60"
			>
				{loading ? 'Memproses…' : 'Masuk Panel Admin'}
			</button>
		</form>
	</div>
</div>
