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

<div class="animate-in fade-in space-y-8 pb-20 duration-500">
	<div class="space-y-1">
		<h1 class="text-3xl font-black text-[#0a2e31]">Kategori Transaksi</h1>
		<p class="text-sm font-medium text-gray-500">
			Kelola kategori untuk pencatatan keuangan yang lebih rapi.
		</p>
	</div>

	{#if errorMessage && !showModal}
		<div class="rounded-2xl border border-red-100 bg-red-50 px-4 py-3 text-sm font-bold text-red-600">
			{errorMessage}
		</div>
	{/if}

	<div class="grid grid-cols-1 gap-8 md:grid-cols-2">
		<!-- Expense Section -->
		<div class="space-y-6">
			<div class="flex items-center justify-between">
				<h2
					class="flex items-center gap-2 text-lg font-black tracking-widest text-red-500 uppercase"
				>
					<div class="h-2 w-2 rounded-full bg-red-500"></div>
					Pengeluaran
				</h2>
				<button
					onclick={() => openAddModal('EXPENSE')}
					class="rounded-full px-3 py-1.5 text-[10px] font-black tracking-widest text-[#217b84] uppercase transition-colors hover:bg-teal-50"
				>
					+ Tambah Kategori
				</button>
			</div>

			<div
				class="grid grid-cols-2 gap-3 rounded-[2.5rem] border border-gray-100 bg-white p-4 shadow-sm sm:grid-cols-3"
			>
				{#if loading}
					{#each [0, 1, 2, 3, 4, 5] as i (i)}
						<div class="h-24 animate-pulse rounded-3xl bg-gray-50"></div>
					{/each}
				{:else if expenseCategories.length === 0}
					<div class="col-span-full py-10 text-center text-xs font-bold text-gray-400">
						Belum ada kategori.
					</div>
				{:else}
					{#each expenseCategories as cat (cat.id)}
						<button
							onclick={() => openEditModal(cat)}
							class="group flex flex-col items-center justify-center gap-2 rounded-3xl border-2 border-transparent p-4 text-center transition-colors hover:border-gray-100 hover:bg-gray-50"
						>
							<div
								class="flex h-12 w-12 items-center justify-center rounded-2xl text-2xl shadow-sm transition-transform group-hover:scale-110"
								style="background-color: {cat.color}15; color: {cat.color}"
							>
								{cat.icon}
							</div>
							<span class="px-1 text-[11px] leading-tight font-bold text-[#0a2e31]">{cat.name}</span
							>
						</button>
					{/each}
				{/if}
			</div>
		</div>

		<!-- Income Section -->
		<div class="space-y-6">
			<div class="flex items-center justify-between">
				<h2
					class="flex items-center gap-2 text-lg font-black tracking-widest text-green-600 uppercase"
				>
					<div class="h-2 w-2 rounded-full bg-green-500"></div>
					Pemasukan
				</h2>
				<button
					onclick={() => openAddModal('INCOME')}
					class="rounded-full px-3 py-1.5 text-[10px] font-black tracking-widest text-[#217b84] uppercase transition-colors hover:bg-teal-50"
				>
					+ Tambah Kategori
				</button>
			</div>

			<div
				class="grid grid-cols-2 gap-3 rounded-[2.5rem] border border-gray-100 bg-white p-4 shadow-sm sm:grid-cols-3"
			>
				{#if loading}
					{#each [0, 1, 2] as i (i)}
						<div class="h-24 animate-pulse rounded-3xl bg-gray-50"></div>
					{/each}
				{:else if incomeCategories.length === 0}
					<div class="col-span-full py-10 text-center text-xs font-bold text-gray-400">
						Belum ada kategori.
					</div>
				{:else}
					{#each incomeCategories as cat (cat.id)}
						<button
							onclick={() => openEditModal(cat)}
							class="group flex flex-col items-center justify-center gap-2 rounded-3xl border-2 border-transparent p-4 text-center transition-colors hover:border-gray-100 hover:bg-gray-50"
						>
							<div
								class="flex h-12 w-12 items-center justify-center rounded-2xl text-2xl shadow-sm transition-transform group-hover:scale-110"
								style="background-color: {cat.color}15; color: {cat.color}"
							>
								{cat.icon}
							</div>
							<span class="px-1 text-[11px] leading-tight font-bold text-[#0a2e31]">{cat.name}</span
							>
						</button>
					{/each}
				{/if}
			</div>
		</div>
	</div>
</div>

<!-- Modal Kategori -->
{#if showModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-[#0a2e31]/40 p-4 backdrop-blur-sm"
		in:fade={{ duration: 200 }}
	>
		<div
			class="w-full max-w-sm overflow-hidden rounded-[2.5rem] bg-white shadow-2xl"
			in:fly={{ y: 20, duration: 400 }}
		>
			<div class="space-y-6 p-8">
				<div class="flex items-center justify-between">
					<h2 class="text-xl font-black text-[#0a2e31]">
						{form.id ? 'Edit Kategori' : 'Kategori Baru'}
					</h2>
					<button
						aria-label="Tutup modal"
						onclick={() => (showModal = false)}
						class="text-gray-300 transition-colors hover:text-gray-500"
					>
						<svg
							class="h-6 w-6"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2.5"><path d="M6 18L18 6M6 6l12 12" /></svg
						>
					</button>
				</div>

				<form onsubmit={handleSaveCategory} class="space-y-5">
					<div class="space-y-1.5">
						<label
							for="name"
							class="block px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
							>Nama Kategori</label
						>
						<input
							id="name"
							type="text"
							required
							bind:value={form.name}
							placeholder="Misal: Belanja Bulanan"
							class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] transition-all outline-none focus:ring-2 focus:ring-teal-500"
						/>
					</div>

					<div class="space-y-1.5">
						<label
							for="type"
							class="block px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
							>Tipe</label
						>
						<select
							id="type"
							bind:value={form.category_type}
							class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-sm font-bold text-[#0a2e31] transition-all outline-none focus:ring-2 focus:ring-teal-500"
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
								class="block px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
								>Icon (Emoji)</label
							>
							<input
								id="icon"
								type="text"
								required
								bind:value={form.icon}
								placeholder="🍔"
								class="w-full rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 text-center text-xl transition-all outline-none focus:ring-2 focus:ring-teal-500"
							/>
						</div>

						<div class="space-y-1.5">
							<label
								for="color"
								class="block px-1 text-[10px] font-black tracking-widest text-gray-400 uppercase"
								>Warna</label
							>
							<input
								id="color"
								type="color"
								required
								bind:value={form.color}
								class="h-[52px] w-full cursor-pointer rounded-xl border border-gray-100 bg-gray-50 px-2 py-1"
							/>
						</div>
					</div>

					<!-- Preset Colors -->
					<div class="flex justify-between gap-1 pt-2">
						{#each predefinedColors as color (color)}
							<button
								type="button"
								aria-label="Pilih warna {color}"
								onclick={() => (form.color = color)}
								class="h-6 w-6 rounded-full border-2 transition-transform hover:scale-110 active:scale-90"
								class:border-gray-900={form.color === color}
								class:border-transparent={form.color !== color}
								style="background-color: {color}"
							></button>
						{/each}
					</div>

					{#if errorMessage}
						<div
							class="rounded-2xl border border-red-100 bg-red-50 px-4 py-3 text-xs font-bold text-red-600"
						>
							{errorMessage}
						</div>
					{/if}

					<div class="flex gap-3 pt-4">
						{#if form.id}
							<button
								type="button"
								onclick={() => handleDeleteCategory(form.id)}
								disabled={saving}
								class="rounded-xl border border-red-100 px-5 py-3 text-[10px] font-black tracking-widest text-red-500 uppercase transition-all hover:bg-red-50"
							>
								Hapus
							</button>
						{/if}
						<button
							type="submit"
							disabled={saving}
							class="flex-1 rounded-xl bg-[#0a2e31] py-3 text-[10px] font-black tracking-widest text-white uppercase shadow-lg transition-all hover:bg-black active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-60"
						>
							{saving ? 'Menyimpan...' : 'Simpan'}
						</button>
					</div>
				</form>
			</div>
		</div>
	</div>
{/if}
