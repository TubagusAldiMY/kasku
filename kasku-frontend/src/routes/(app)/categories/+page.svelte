<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { fade, fly } from 'svelte/transition';

	type Category = {
		id: string;
		name: string;
		category_type: 'INCOME' | 'EXPENSE';
		icon: string;
		color: string;
	};

	let categories = $state<Category[]>([]);
	let loading = $state(true);
	let showModal = $state(false);

	let form = $state<Category>({
		id: '',
		name: '',
		category_type: 'EXPENSE',
		icon: '🏷️',
		color: '#6b7280'
	});

	// Mock Data
	const mockCategories: Category[] = [
		{ id: '1', name: 'Makanan & Minuman', category_type: 'EXPENSE', icon: '🍔', color: '#f59e0b' },
		{ id: '2', name: 'Transportasi', category_type: 'EXPENSE', icon: '🚗', color: '#3b82f6' },
		{ id: '3', name: 'Gaji', category_type: 'INCOME', icon: '💰', color: '#10b981' },
		{ id: '4', name: 'Hiburan', category_type: 'EXPENSE', icon: '🎬', color: '#8b5cf6' },
		{ id: '5', name: 'Investasi', category_type: 'EXPENSE', icon: '📈', color: '#14b8a6' },
		{ id: '6', name: 'Bonus', category_type: 'INCOME', icon: '🎁', color: '#ec4899' }
	];

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

	async function fetchCategories() {
		loading = true;
		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';

		if (isMock) {
			setTimeout(() => {
				categories = mockCategories;
				loading = false;
			}, 600);
			return;
		}

		try {
			const res = await apiFetch('/categories');
			const result = await res.json();
			if (result.success) categories = result.data || [];
		} catch (err) {
			console.error(err);
		} finally {
			loading = false;
		}
	}

	async function handleSaveCategory(e: SubmitEvent) {
		e.preventDefault();
		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';

		if (isMock) {
			if (form.id) {
				categories = categories.map((c) => (c.id === form.id ? { ...c, ...form } : c));
			} else {
				categories = [...categories, { ...form, id: Math.random().toString() }];
			}
			showModal = false;
			return;
		}

		try {
			const method = form.id ? 'PUT' : 'POST';
			const url = form.id ? `/categories/${form.id}` : '/categories';
			const res = await apiFetch(url, {
				method,
				body: JSON.stringify(form)
			});
			const result = await res.json();
			if (result.success) {
				showModal = false;
				fetchCategories();
			}
		} catch (err) {
			console.error(err);
		}
	}

	async function handleDeleteCategory(id: string) {
		if (!confirm('Hapus kategori ini? Transaksi terkait mungkin akan terpengaruh.')) return;

		const isMock = localStorage.getItem('kasku_mock_mode') === 'true';
		if (isMock) {
			categories = categories.filter((c) => c.id !== id);
			showModal = false;
			return;
		}

		try {
			const res = await apiFetch(`/categories/${id}`, { method: 'DELETE' });
			const result = await res.json();
			if (result.success) {
				showModal = false;
				fetchCategories();
			} else {
				alert(result.error?.message || 'Gagal menghapus kategori');
			}
		} catch (err) {
			console.error(err);
		}
	}

	function openAddModal(type: 'INCOME' | 'EXPENSE') {
		form = { id: '', name: '', category_type: type, icon: '🏷️', color: predefinedColors[0] };
		showModal = true;
	}

	function openEditModal(cat: Category) {
		form = { ...cat };
		showModal = true;
	}

	onMount(fetchCategories);

	const expenseCategories = $derived(categories.filter((c) => c.category_type === 'EXPENSE'));
	const incomeCategories = $derived(categories.filter((c) => c.category_type === 'INCOME'));
</script>

<div class="animate-in fade-in space-y-8 pb-20 duration-500">
	<div class="space-y-1">
		<h1 class="text-3xl font-black text-[#0a2e31]">Kategori Transaksi</h1>
		<p class="text-sm font-medium text-gray-500">
			Kelola kategori untuk pencatatan keuangan yang lebih rapi.
		</p>
	</div>

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

					<div class="flex gap-3 pt-4">
						{#if form.id}
							<button
								type="button"
								onclick={() => handleDeleteCategory(form.id)}
								class="rounded-xl border border-red-100 px-5 py-3 text-[10px] font-black tracking-widest text-red-500 uppercase transition-all hover:bg-red-50"
							>
								Hapus
							</button>
						{/if}
						<button
							type="submit"
							class="flex-1 rounded-xl bg-[#0a2e31] py-3 text-[10px] font-black tracking-widest text-white uppercase shadow-lg transition-all hover:bg-black active:scale-[0.98]"
						>
							Simpan
						</button>
					</div>
				</form>
			</div>
		</div>
	</div>
{/if}
