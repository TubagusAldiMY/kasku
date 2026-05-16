<script lang="ts">
	import { fade, fly } from 'svelte/transition';
	import { apiFetch } from '$lib/api/client';

	let generating = $state(false);
	let progress = $state(0);
	let message = $state<{ type: 'success' | 'info', text: string } | null>(null);

	// Report Config State
	let reportConfig = $state({
		type: 'SUMMARY',
		format: 'PDF',
		dateStart: new Date(new Date().getFullYear(), new Date().getMonth(), 1).toISOString().split('T')[0],
		dateEnd: new Date().toISOString().split('T')[0],
		includeCharts: true
	});

	// Mock History
	let reportHistory = $state([
		{ id: '1', name: 'Laporan_Mei_2026.pdf', date: '10 Mei 2026', type: 'Ringkasan', size: '1.2 MB' },
		{ id: '2', name: 'Transaksi_April.csv', date: '01 Mei 2026', type: 'Detail Transaksi', size: '450 KB' }
	]);

	const reportTypes = [
		{ id: 'SUMMARY', label: 'Ringkasan Bulanan', desc: 'Gabungan arus kas, saldo akhir, dan performa aset.' },
		{ id: 'DETAILED', label: 'Detail Transaksi', desc: 'Daftar seluruh transaksi masuk dan keluar secara rinci.' },
		{ id: 'CATEGORY', label: 'Analisis Kategori', desc: 'Breakdown pengeluaran berdasarkan kategori terbanyak.' }
	];

	async function generateReport() {
		generating = true;
		progress = 0;
		message = null;

		// Jika format CSV, panggil backend asli
		if (reportConfig.format === 'CSV') {
			try {
				const res = await apiFetch('/transactions/export');
				if (!res.ok) throw new Error('Gagal mengambil data dari server.');
				
				const blob = await res.blob();
				const url = window.URL.createObjectURL(blob);
				const a = document.createElement('a');
				a.href = url;
				a.download = `kasku_export_${new Date().toISOString().split('T')[0]}.csv`;
				document.body.appendChild(a);
				a.click();
				window.URL.revokeObjectURL(url);
				document.body.removeChild(a);

				message = { type: 'success', text: 'Laporan CSV berhasil diunduh!' };
				
				// Tambah ke history
				const newReport = {
					id: Math.random().toString(),
					name: `kasku_export_${new Date().toISOString().split('T')[0]}.csv`,
					date: 'Baru saja',
					type: 'Detail Transaksi',
					size: `${(blob.size / 1024).toFixed(1)} KB`
				};
				reportHistory = [newReport, ...reportHistory];
			} catch (err: any) {
				message = { type: 'info', text: err.message || 'Gagal ekspor CSV.' };
			} finally {
				generating = false;
			}
			return;
		}

		// Simulasi proses pembuatan laporan untuk format lain (FE Team style)
		const interval = setInterval(() => {
			progress += Math.random() * 30;
			if (progress >= 100) {
				progress = 100;
				clearInterval(interval);
				setTimeout(() => {
					generating = false;
					message = { type: 'success', text: `Laporan ${reportConfig.format} berhasil dibuat!` };
					
					// Tambah ke history (mock)
					const newReport = {
						id: Math.random().toString(),
						name: `Laporan_${reportConfig.type}_${reportConfig.dateStart}.${reportConfig.format.toLowerCase()}`,
						date: 'Baru saja',
						type: reportTypes.find(t => t.id === reportConfig.type)?.label || 'Custom',
						size: '850 KB'
					};
					reportHistory = [newReport, ...reportHistory];
				}, 500);
			}
		}, 400);
	}
</script>

<div class="space-y-10 animate-in fade-in duration-700 pb-20">
	<div class="space-y-1">
		<h1 class="text-3xl font-black text-[#0a2e31]">Laporan Keuangan</h1>
		<p class="text-gray-500 font-medium">Ekspor data finansial Anda ke berbagai format profesional.</p>
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-3 gap-10">
		<!-- Configuration Panel -->
		<div class="lg:col-span-2 space-y-8">
			<div class="bg-white p-10 rounded-[2.5rem] border border-gray-100 shadow-sm space-y-8">
				<h3 class="text-xl font-black text-[#0a2e31]">Konfigurasi Laporan</h3>
				
				<div class="space-y-8">
					<!-- Report Type Selection -->
					<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
						{#each reportTypes as type}
							<button 
								onclick={() => reportConfig.type = type.id}
								class="flex flex-col text-left p-5 rounded-3xl border-2 transition-all {reportConfig.type === type.id ? 'border-[#217b84] bg-teal-50 shadow-md' : 'border-gray-50 hover:border-gray-200'}"
							>
								<span class="text-sm font-bold text-[#0a2e31] mb-2">{type.label}</span>
								<span class="text-[10px] text-gray-500 leading-relaxed">{type.desc}</span>
							</button>
						{/each}
					</div>

					<div class="grid grid-cols-1 md:grid-cols-2 gap-8">
						<!-- Date Range -->
						<div class="space-y-4">
							<span class="block text-[11px] font-bold text-gray-400 uppercase tracking-widest px-1">Rentang Waktu</span>
							<div class="flex items-center gap-3">
								<input 
									type="date" 
									bind:value={reportConfig.dateStart}
									class="flex-1 px-4 py-3 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 outline-none transition-all text-sm font-medium"
								/>
								<span class="text-gray-300 font-bold">s/d</span>
								<input 
									type="date" 
									bind:value={reportConfig.dateEnd}
									class="flex-1 px-4 py-3 bg-gray-50 border border-gray-100 rounded-2xl focus:ring-4 focus:ring-teal-50 outline-none transition-all text-sm font-medium"
								/>
							</div>
						</div>

						<!-- Format -->
						<div class="space-y-4">
							<span class="block text-[11px] font-bold text-gray-400 uppercase tracking-widest px-1">Format Berkas</span>
							<div class="flex p-1.5 bg-gray-50 rounded-2xl border border-gray-100">
								{#each ['PDF', 'CSV', 'XLSX'] as fmt}
									<button 
										onclick={() => reportConfig.format = fmt}
										class="flex-1 py-2 text-xs font-bold rounded-xl transition-all {reportConfig.format === fmt ? 'bg-[#0a2e31] text-white shadow-lg' : 'text-gray-400 hover:text-[#0a2e31]'}"
									>
										{fmt}
									</button>
								{/each}
							</div>
						</div>
					</div>

					<!-- Options -->
					<div class="flex items-center gap-3 px-1">
						<input type="checkbox" id="charts" bind:checked={reportConfig.includeCharts} class="h-5 w-5 rounded-lg border-gray-200 text-[#217b84] focus:ring-[#217b84]" />
						<label for="charts" class="text-sm font-bold text-[#0a2e31]">Sertakan Visualisasi Grafik (Rekomendasi untuk PDF)</label>
					</div>

					<div class="pt-6 border-t border-gray-50">
						{#if generating}
							<div class="space-y-4">
								<div class="flex justify-between items-center text-xs font-bold uppercase tracking-widest text-[#217b84]">
									<span>Menyusun Berkas...</span>
									<span>{Math.round(progress)}%</span>
								</div>
								<div class="h-3 w-full bg-gray-50 rounded-full overflow-hidden border border-gray-100">
									<div class="h-full bg-gradient-to-r from-[#217b84] to-[#4fd1c5] transition-all duration-300" style="width: {progress}%"></div>
								</div>
							</div>
						{:else}
							<button 
								onclick={generateReport}
								class="w-full py-4 bg-[#217b84] hover:bg-[#1a5f66] text-white font-bold rounded-[1.5rem] shadow-xl shadow-teal-900/10 transition-all active:scale-[0.98] flex items-center justify-center gap-3"
							>
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" /></svg>
								Buat Laporan Sekarang
							</button>
						{/if}
					</div>
				</div>
			</div>
		</div>

		<!-- History Sidebar -->
		<div class="space-y-6">
			<div class="flex items-center justify-between px-2">
				<h3 class="text-xl font-bold text-[#0a2e31]">Riwayat Unduhan</h3>
			</div>

			{#if message}
				<div 
					in:fly={{ x: 20, duration: 400 }}
					class="p-4 rounded-[1.5rem] bg-green-50 border border-green-100 flex items-center gap-3 text-green-800"
				>
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" /></svg>
					<span class="text-xs font-bold">{message.text}</span>
				</div>
			{/if}

			<div class="space-y-4">
				{#each reportHistory as report}
					<div class="bg-white p-5 rounded-[1.8rem] border border-gray-100 shadow-sm group hover:border-[#217b84]/30 transition-all">
						<div class="flex items-start justify-between mb-4">
							<div class="h-10 w-10 rounded-2xl bg-gray-50 flex items-center justify-center text-gray-400 group-hover:bg-teal-50 group-hover:text-teal-600 transition-colors">
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									{#if report.name.endsWith('pdf')}
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
									{:else}
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17v-2a4 4 0 00-4-4H5a2 2 0 00-2 2v2a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-2a2 2 0 00-2-2H9a2 2 0 00-2 2v2a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z" />
									{/if}
								</svg>
							</div>
							<button class="p-2 text-gray-300 hover:text-[#0a2e31] transition-colors" aria-label={`Unduh ${report.name}`}>
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" /></svg>
							</button>
						</div>
						<h4 class="text-sm font-bold text-[#0a2e31] truncate mb-1">{report.name}</h4>
						<div class="flex justify-between items-center text-[10px] font-bold uppercase tracking-wider text-gray-400">
							<span>{report.type}</span>
							<span>{report.size}</span>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</div>
</div>
