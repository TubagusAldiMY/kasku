<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { fade, fly } from 'svelte/transition';

	type CategoryType = 'INCOME' | 'EXPENSE' | 'BOTH';

	type Category = {
		id: string;
		name: string;
		category_type: CategoryType;
		icon: string;
		color: string;
		is_default?: boolean;
		updated_at?: string;
	};

	type ServerCategory = {
		id?: string;
		ID?: string;
		name?: string;
		Name?: string;
		category_type?: CategoryType;
		CategoryType?: CategoryType;
		icon?: string;
		Icon?: string;
		color?: string;
		Color?: string;
		is_default?: boolean;
		IsDefault?: boolean;
		updated_at?: string;
		UpdatedAt?: string;
	};

	let categories = $state<Category[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let errorMessage = $state('');
	let showModal = $state(false);

	let form = $state<Category>({
		id: '',
		name: '',
		category_type: 'EXPENSE',
		icon: '🏷️',
		color: '#6b7280'
	});

	const predefinedColors = [
		'#f59e0b',
		'#3b82f6',
		'#10b981',
		'#8b5cf6',
		'#14b8a6',
		'#ec4899',
		'#ef4444',
		'#64748b'
	];

	function normalizeCategory(item: ServerCategory): Category | null {
		const id = item.id ?? item.ID;
		const name = item.name ?? item.Name;
		const categoryType = item.category_type ?? item.CategoryType;

		if (!id || !name || !categoryType) return null;

		return {
			id,
			name,
			category_type: categoryType,
			icon: item.icon ?? item.Icon ?? '🏷️',
			color: item.color ?? item.Color ?? '#6b7280',
			is_default: item.is_default ?? item.IsDefault,
			updated_at: item.updated_at ?? item.UpdatedAt
		};
	}

	function normalizeCategories(data: unknown): Category[] {
		if (!Array.isArray(data)) return [];
		return data
			.map((item) => normalizeCategory(item as ServerCategory))
			.filter((item): item is Category => item !== null);
	}

	async function readApiResult(res: Response) {
		try {
			return await res.json();
		} catch {
			return {
				success: false,
				error: { message: `Response API tidak valid (HTTP ${res.status})` }
			};
		}
	}

	async function fetchCategories() {
		loading = true;
		errorMessage = '';

		try {
			const res = await apiFetch('/categories');
			const result = await readApiResult(res);
			if (res.ok && result.success) {
				categories = normalizeCategories(result.data);
			} else {
				errorMessage = result.error?.message || 'Gagal memuat kategori.';
			}
		} catch (err) {
			console.error(err);
			errorMessage = 'Gagal memuat kategori. Periksa koneksi atau service backend.';
		} finally {
			loading = false;
		}
	}

	async function handleSaveCategory(e: SubmitEvent) {
		e.preventDefault();
		errorMessage = '';

		try {
			saving = true;
			const method = form.id ? 'PUT' : 'POST';
			const url = form.id ? `/categories/${form.id}` : '/categories';
			const res = await apiFetch(url, {
				method,
				body: JSON.stringify({
					name: form.name.trim(),
					category_type: form.category_type,
					icon: form.icon.trim(),
					color: form.color
				})
			});
			const result = await readApiResult(res);
			if (res.ok && result.success) {
				showModal = false;
				await fetchCategories();
			} else {
				errorMessage = result.error?.message || 'Gagal menyimpan kategori.';
			}
		} catch (err) {
			console.error(err);
			errorMessage = 'Gagal menyimpan kategori. Periksa koneksi atau service backend.';
		} finally {
			saving = false;
		}
	}

	async function handleDeleteCategory(id: string) {
		if (!confirm('Hapus kategori ini? Transaksi terkait mungkin akan terpengaruh.')) return;

		try {
			saving = true;
			const res = await apiFetch(`/categories/${id}`, { method: 'DELETE' });
			const result = await readApiResult(res);
			if (res.ok && result.success) {
				showModal = false;
				await fetchCategories();
			} else {
				errorMessage = result.error?.message || 'Gagal menghapus kategori';
			}
		} catch (err) {
			console.error(err);
			errorMessage = 'Gagal menghapus kategori. Periksa koneksi atau service backend.';
		} finally {
			saving = false;
		}
	}

	function openAddModal(type: CategoryType) {
		errorMessage = '';
		form = { id: '', name: '', category_type: type, icon: '🏷️', color: predefinedColors[0] };
		showModal = true;
	}

	function openEditModal(cat: Category) {
		errorMessage = '';
		form = { ...cat };
		showModal = true;
	}

	onMount(fetchCategories);

	const expenseCategories = $derived(
		categories.filter((c) => c.category_type === 'EXPENSE' || c.category_type === 'BOTH')
	);
	const incomeCategories = $derived(
		categories.filter((c) => c.category_type === 'INCOME' || c.category_type === 'BOTH')
	);
</script>

<div>
	<!-- ═══════════ Header ═══════════ -->
	<div class="border-b border-ink/10 pb-8">
		<h1 class="font-serif text-4xl tracking-tight text-ink">Kategori</h1>
		<p class="mt-2.5 text-sm text-ink/60">
			Kelola kategori untuk pencatatan keuangan yang lebih rapi.
		</p>
	</div>

	{#if errorMessage && !showModal}
		<div class="mt-6 rounded-[10px] border border-clay/25 bg-clay/5 px-4 py-3 text-sm text-clay">
			{errorMessage}
		</div>
	{/if}

	<div class="grid grid-cols-1 gap-12 pt-8 md:grid-cols-2 md:gap-16">
		<!-- Expense Section -->
		<div>
			<div class="mb-1 flex items-center justify-between">
				<div class="flex items-center gap-2.5">
					<span class="h-1.5 w-1.5 rounded-full bg-clay"></span>
					<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">
						Pengeluaran
					</p>
				</div>
				<button
					onclick={() => openAddModal('EXPENSE')}
					class="text-[12.5px] font-semibold text-teal transition-colors hover:text-ink"
				>
					+ Tambah
				</button>
			</div>

			{#if loading}
				{#each [0, 1, 2, 3] as i (i)}
					<div class="border-t border-ink/10 py-4">
						<div class="h-5 w-40 animate-pulse rounded bg-ink/5"></div>
					</div>
				{/each}
			{:else if expenseCategories.length === 0}
				<p class="border-t border-ink/10 py-8 text-sm text-ink/45">Belum ada kategori.</p>
			{:else}
				{#each expenseCategories as cat (cat.id)}
					<button
						onclick={() => openEditModal(cat)}
						class="group flex w-full items-center gap-3.5 border-t border-ink/10 py-4 text-left transition-colors last:border-b hover:bg-ink/[0.02]"
					>
						<span
							class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-base"
							style="background-color: {cat.color}18; color: {cat.color}"
						>
							{cat.icon}
						</span>
						<span class="flex-1 truncate font-serif text-lg text-ink">{cat.name}</span>
						{#if cat.category_type === 'BOTH'}
							<span
								class="shrink-0 rounded-full border border-ink/15 px-2 py-px text-[11px] text-ink/55"
								>Keduanya</span
							>
						{/if}
						{#if cat.is_default}
							<span
								class="shrink-0 rounded-full border border-ink/15 px-2 py-px text-[11px] text-ink/45"
								>Bawaan</span
							>
						{/if}
						<svg
							class="h-4 w-4 shrink-0 text-ink/25 transition-colors group-hover:text-ink/50"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.8"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
							/>
						</svg>
					</button>
				{/each}
			{/if}
		</div>

		<!-- Income Section -->
		<div>
			<div class="mb-1 flex items-center justify-between">
				<div class="flex items-center gap-2.5">
					<span class="h-1.5 w-1.5 rounded-full bg-teal"></span>
					<p class="text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase">Pemasukan</p>
				</div>
				<button
					onclick={() => openAddModal('INCOME')}
					class="text-[12.5px] font-semibold text-teal transition-colors hover:text-ink"
				>
					+ Tambah
				</button>
			</div>

			{#if loading}
				{#each [0, 1, 2] as i (i)}
					<div class="border-t border-ink/10 py-4">
						<div class="h-5 w-40 animate-pulse rounded bg-ink/5"></div>
					</div>
				{/each}
			{:else if incomeCategories.length === 0}
				<p class="border-t border-ink/10 py-8 text-sm text-ink/45">Belum ada kategori.</p>
			{:else}
				{#each incomeCategories as cat (cat.id)}
					<button
						onclick={() => openEditModal(cat)}
						class="group flex w-full items-center gap-3.5 border-t border-ink/10 py-4 text-left transition-colors last:border-b hover:bg-ink/[0.02]"
					>
						<span
							class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-base"
							style="background-color: {cat.color}18; color: {cat.color}"
						>
							{cat.icon}
						</span>
						<span class="flex-1 truncate font-serif text-lg text-ink">{cat.name}</span>
						{#if cat.category_type === 'BOTH'}
							<span
								class="shrink-0 rounded-full border border-ink/15 px-2 py-px text-[11px] text-ink/55"
								>Keduanya</span
							>
						{/if}
						{#if cat.is_default}
							<span
								class="shrink-0 rounded-full border border-ink/15 px-2 py-px text-[11px] text-ink/45"
								>Bawaan</span
							>
						{/if}
						<svg
							class="h-4 w-4 shrink-0 text-ink/25 transition-colors group-hover:text-ink/50"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.8"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
							/>
						</svg>
					</button>
				{/each}
			{/if}
		</div>
	</div>
</div>

<!-- Modal Kategori -->
{#if showModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-4 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
	>
		<div
			class="w-full max-w-sm overflow-hidden rounded-2xl border border-ink/10 bg-card shadow-2xl"
			in:fly={{ y: 20, duration: 400 }}
		>
			<div class="space-y-6 p-8">
				<div class="flex items-center justify-between">
					<h2 class="font-serif text-2xl text-ink">
						{form.id ? 'Edit kategori' : 'Kategori baru'}
					</h2>
					<button
						aria-label="Tutup modal"
						onclick={() => (showModal = false)}
						class="rounded-full p-1.5 text-ink/40 transition-colors hover:bg-ink/5 hover:text-ink"
					>
						<svg
							class="h-5 w-5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"><path d="M6 18L18 6M6 6l12 12" /></svg
						>
					</button>
				</div>

				<form onsubmit={handleSaveCategory} class="space-y-5">
					<div class="space-y-1.5">
						<label
							for="name"
							class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
							>Nama kategori</label
						>
						<input
							id="name"
							type="text"
							required
							bind:value={form.name}
							placeholder="Misal: Belanja Bulanan"
							class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none focus:border-teal"
						/>
					</div>

					<div class="space-y-1.5">
						<label
							for="type"
							class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
							>Tipe</label
						>
						<select
							id="type"
							bind:value={form.category_type}
							class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-sm text-ink transition-colors outline-none focus:border-teal"
						>
							<option value="EXPENSE">Pengeluaran</option>
							<option value="INCOME">Pemasukan</option>
							<option value="BOTH">Keduanya</option>
						</select>
					</div>

					<div class="grid grid-cols-2 gap-4">
						<div class="space-y-1.5">
							<label
								for="icon"
								class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>Icon (emoji)</label
							>
							<input
								id="icon"
								type="text"
								required
								bind:value={form.icon}
								placeholder="🍔"
								class="w-full rounded-[10px] border border-ink/25 bg-field px-4 py-3 text-center text-xl transition-colors outline-none focus:border-teal"
							/>
						</div>

						<div class="space-y-1.5">
							<label
								for="color"
								class="block text-[11px] font-semibold tracking-[0.12em] text-ink/45 uppercase"
								>Warna</label
							>
							<input
								id="color"
								type="color"
								required
								bind:value={form.color}
								class="h-[50px] w-full cursor-pointer rounded-[10px] border border-ink/25 bg-field px-2 py-1"
							/>
						</div>
					</div>

					<!-- Preset Colors -->
					<div class="flex justify-between gap-1 pt-1">
						{#each predefinedColors as color (color)}
							<button
								type="button"
								aria-label="Pilih warna {color}"
								onclick={() => (form.color = color)}
								class="h-6 w-6 rounded-full border-2 transition-transform hover:scale-110 active:scale-90"
								class:border-ink={form.color === color}
								class:border-transparent={form.color !== color}
								style="background-color: {color}"
							></button>
						{/each}
					</div>

					{#if errorMessage}
						<div class="rounded-[10px] border border-clay/25 bg-clay/5 px-4 py-3 text-xs text-clay">
							{errorMessage}
						</div>
					{/if}

					<div class="flex gap-3 pt-4">
						{#if form.id}
							<button
								type="button"
								onclick={() => handleDeleteCategory(form.id)}
								disabled={saving}
								class="rounded-full border border-ink/15 px-5 py-2.5 text-[13px] font-semibold text-clay transition-colors hover:border-clay/30 hover:bg-clay/5 disabled:opacity-60"
							>
								Hapus
							</button>
						{/if}
						<button
							type="submit"
							disabled={saving}
							class="flex-1 rounded-full bg-teal py-2.5 text-[13px] font-semibold text-card transition-colors hover:bg-ink disabled:cursor-not-allowed disabled:opacity-60"
						>
							{saving ? 'Menyimpan…' : 'Simpan'}
						</button>
					</div>
				</form>
			</div>
		</div>
	</div>
{/if}
