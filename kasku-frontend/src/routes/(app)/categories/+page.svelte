<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api/client';
	import { fade, fly } from 'svelte/transition';

	let categories = $state<any[]>([]);
	let loading = $state(true);
	let showModal = $state(false);

	let form = $state({
		id: '',
		name: '',
		category_type: 'EXPENSE',
		icon: '🏷️',
		color: '#6b7280'
	});

	// Mock Data
	const mockCategories = [
		{ id: '1', name: 'Makanan & Minuman', category_type: 'EXPENSE', icon: '🍔', color: '#f59e0b' },
		{ id: '2', name: 'Transportasi', category_type: 'EXPENSE', icon: '🚗', color: '#3b82f6' },
		{ id: '3', name: 'Gaji', category_type: 'INCOME', icon: '💰', color: '#10b981' },
		{ id: '4', name: 'Hiburan', category_type: 'EXPENSE', icon: '🎬', color: '#8b5cf6' },
		{ id: '5', name: 'Investasi', category_type: 'EXPENSE', icon: '📈', color: '#14b8a6' },
		{ id: '6', name: 'Bonus', category_type: 'INCOME', icon: '🎁', color: '#ec4899' }
	];

	const predefinedColors = [
		'#f59e0b', '#3b82f6', '#10b981', '#8b5cf6', '#14b8a6', '#ec4899', '#ef4444', '#64748b'
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
				categories = categories.map(c => c.id === form.id ? { ...c, ...form } : c);
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
			categories = categories.filter(c => c.id !== id);
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

	function openEditModal(cat: any) {
		form = { ...cat };
		showModal = true;
	}

	onMount(fetchCategories);

	const expenseCategories = $derived(categories.filter(c => c.category_type === 'EXPENSE'));
	const incomeCategories = $derived(categories.filter(c => c.category_type === 'INCOME'));
</script>

<div class="space-y-8 animate-in fade-in duration-500 pb-20">
	<div class="space-y-1">
		<h1 class="text-3xl font-black text-[#0a2e31]">Kategori Transaksi</h1>
		<p class="text-gray-500 font-medium text-sm">Kelola kategori untuk pencatatan keuangan yang lebih rapi.</p>
	</div>

	<div class="grid grid-cols-1 md:grid-cols-2 gap-8">
		<!-- Expense Section -->
		<div class="space-y-6">
			<div class="flex items-center justify-between">
				<h2 class="text-lg font-black text-red-500 uppercase tracking-widest flex items-center gap-2">
					<div class="h-2 w-2 rounded-full bg-red-500"></div>
					Pengeluaran
				</h2>
				<button 
					onclick={() => openAddModal('EXPENSE')}
					class="text-[10px] font-black uppercase tracking-widest text-[#217b84] hover:bg-teal-50 px-3 py-1.5 rounded-full transition-colors"
				>
					+ Tambah Kategori
				</button>
			</div>

			<div class="bg-white rounded-[2.5rem] border border-gray-100 shadow-sm p-4 grid grid-cols-2 sm:grid-cols-3 gap-3">
				{#if loading}
					{#each Array(6) as _}
						<div class="h-24 bg-gray-50 rounded-3xl animate-pulse"></div>
					{/each}
				{:else if expenseCategories.length === 0}
					<div class="col-span-full py-10 text-center text-gray-400 font-bold text-xs">Belum ada kategori.</div>
				{:else}
					{#each expenseCategories as cat}
						<button 
							onclick={() => openEditModal(cat)}
							class="flex flex-col items-center justify-center p-4 rounded-3xl hover:bg-gray-50 transition-colors border-2 border-transparent hover:border-gray-100 group text-center gap-2"
						>
							<div class="h-12 w-12 rounded-2xl flex items-center justify-center text-2xl shadow-sm transition-transform group-hover:scale-110" style="background-color: {cat.color}15; color: {cat.color}">
								{cat.icon}
							</div>
							<span class="text-[11px] font-bold text-[#0a2e31] leading-tight px-1">{cat.name}</span>
						</button>
					{/each}
				{/if}
			</div>
		</div>

		<!-- Income Section -->
		<div class="space-y-6">
			<div class="flex items-center justify-between">
				<h2 class="text-lg font-black text-green-600 uppercase tracking-widest flex items-center gap-2">
					<div class="h-2 w-2 rounded-full bg-green-500"></div>
					Pemasukan
				</h2>
				<button 
					onclick={() => openAddModal('INCOME')}
					class="text-[10px] font-black uppercase tracking-widest text-[#217b84] hover:bg-teal-50 px-3 py-1.5 rounded-full transition-colors"
				>
					+ Tambah Kategori
				</button>
			</div>

			<div class="bg-white rounded-[2.5rem] border border-gray-100 shadow-sm p-4 grid grid-cols-2 sm:grid-cols-3 gap-3">
				{#if loading}
					{#each Array(3) as _}
						<div class="h-24 bg-gray-50 rounded-3xl animate-pulse"></div>
					{/each}
				{:else if incomeCategories.length === 0}
					<div class="col-span-full py-10 text-center text-gray-400 font-bold text-xs">Belum ada kategori.</div>
				{:else}
					{#each incomeCategories as cat}
						<button 
							onclick={() => openEditModal(cat)}
							class="flex flex-col items-center justify-center p-4 rounded-3xl hover:bg-gray-50 transition-colors border-2 border-transparent hover:border-gray-100 group text-center gap-2"
						>
							<div class="h-12 w-12 rounded-2xl flex items-center justify-center text-2xl shadow-sm transition-transform group-hover:scale-110" style="background-color: {cat.color}15; color: {cat.color}">
								{cat.icon}
							</div>
							<span class="text-[11px] font-bold text-[#0a2e31] leading-tight px-1">{cat.name}</span>
						</button>
					{/each}
				{/if}
			</div>
		</div>
	</div>
</div>

<!-- Modal Kategori -->
{#if showModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-[#0a2e31]/40 backdrop-blur-sm" in:fade={{ duration: 200 }}>
		<div class="bg-white w-full max-w-sm rounded-[2.5rem] shadow-2xl overflow-hidden" in:fly={{ y: 20, duration: 400 }}>
			<div class="p-8 space-y-6">
				<div class="flex justify-between items-center">
					<h2 class="text-xl font-black text-[#0a2e31]">{form.id ? 'Edit Kategori' : 'Kategori Baru'}</h2>
					<button aria-label="Tutup modal" onclick={() => showModal = false} class="text-gray-300 hover:text-gray-500 transition-colors">
						<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path d="M6 18L18 6M6 6l12 12" /></svg>
					</button>
				</div>

				<form onsubmit={handleSaveCategory} class="space-y-5">
					<div class="space-y-1.5">
						<label for="name" class="block text-[10px] font-black text-gray-400 uppercase tracking-widest px-1">Nama Kategori</label>
						<input id="name" type="text" required bind:value={form.name} placeholder="Misal: Belanja Bulanan" class="w-full px-4 py-3 bg-gray-50 border border-gray-100 rounded-xl focus:ring-2 focus:ring-teal-500 outline-none transition-all font-bold text-sm text-[#0a2e31]" />
					</div>

					<div class="space-y-1.5">
						<label for="type" class="block text-[10px] font-black text-gray-400 uppercase tracking-widest px-1">Tipe</label>
						<select id="type" bind:value={form.category_type} class="w-full px-4 py-3 bg-gray-50 border border-gray-100 rounded-xl focus:ring-2 focus:ring-teal-500 outline-none transition-all font-bold text-sm text-[#0a2e31]">
							<option value="EXPENSE">Pengeluaran</option>
							<option value="INCOME">Pemasukan</option>
						</select>
					</div>

					<div class="grid grid-cols-2 gap-4">
						<div class="space-y-1.5">
							<label for="icon" class="block text-[10px] font-black text-gray-400 uppercase tracking-widest px-1">Icon (Emoji)</label>
							<input id="icon" type="text" required bind:value={form.icon} placeholder="🍔" class="w-full px-4 py-3 bg-gray-50 border border-gray-100 rounded-xl focus:ring-2 focus:ring-teal-500 outline-none transition-all text-xl text-center" />
						</div>

						<div class="space-y-1.5">
							<label for="color" class="block text-[10px] font-black text-gray-400 uppercase tracking-widest px-1">Warna</label>
							<input id="color" type="color" required bind:value={form.color} class="w-full h-[52px] px-2 py-1 bg-gray-50 border border-gray-100 rounded-xl cursor-pointer" />
						</div>
					</div>

					<!-- Preset Colors -->
					<div class="flex justify-between gap-1 pt-2">
						{#each predefinedColors as color}
							<button
								type="button"
								aria-label="Pilih warna {color}"
								onclick={() => form.color = color}
								class="h-6 w-6 rounded-full border-2 transition-transform active:scale-90 hover:scale-110"
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
								class="px-5 py-3 border border-red-100 text-red-500 hover:bg-red-50 font-black text-[10px] uppercase tracking-widest rounded-xl transition-all"
							>
								Hapus
							</button>
						{/if}
						<button 
							type="submit"
							class="flex-1 py-3 bg-[#0a2e31] hover:bg-black text-white font-black text-[10px] uppercase tracking-widest rounded-xl shadow-lg transition-all active:scale-[0.98]"
						>
							Simpan
						</button>
					</div>
				</form>
			</div>
		</div>
	</div>
{/if}
