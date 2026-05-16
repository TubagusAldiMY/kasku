<script lang="ts">
	import { fade, fly } from 'svelte/transition';

	// Mock Admin Data
	const stats = [
		{ label: 'Total Pengguna', value: '1,284', grow: '+12%', icon: '👥', color: 'blue' },
		{ label: 'Pendapatan (Mei)', value: 'Rp 42.500.000', grow: '+8.4%', icon: '💰', color: 'green' },
		{ label: 'Total Transaksi', value: '14,802', grow: '+24%', icon: '💳', color: 'purple' },
		{ label: 'Aset Dipantau', value: 'Rp 8,2 Miliar', grow: '+15%', icon: '📈', color: 'teal' }
	];

	const recentRegistrations = [
		{ id: 1, user: 'budi_hartono', email: 'budi@gmail.com', date: '2 menit lalu', status: 'Active' },
		{ id: 2, user: 'susi_susanti', email: 'susi.s@outlook.com', date: '1 jam lalu', status: 'Pending' },
		{ id: 3, user: 'eko_wijaya', email: 'eko.w@perusahaan.id', date: '3 jam lalu', status: 'Active' },
		{ id: 4, user: 'ani_lestari', email: 'ani_cute88@yahoo.com', date: '5 jam lalu', status: 'Active' }
	];
</script>

<div class="space-y-10 animate-in fade-in duration-700">
	<div class="flex justify-between items-end">
		<div class="space-y-1">
			<h1 class="text-3xl font-black text-[#0a2e31]">Ringkasan Sistem</h1>
			<p class="text-gray-500 font-medium">Pantau performa global dan kesehatan infrastruktur KasKu.</p>
		</div>
		<div class="flex gap-2">
			<span class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-green-50 text-green-700 rounded-full text-[10px] font-black uppercase tracking-widest border border-green-100">
				<div class="h-2 w-2 rounded-full bg-green-500 animate-pulse"></div>
				Backend: Healthy
			</span>
		</div>
	</div>

	<!-- Stats Grid -->
	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
		{#each stats as stat, i}
			<div 
				in:fly={{ y: 20, delay: i * 100, duration: 500 }}
				class="bg-white p-6 rounded-[2rem] border border-gray-100 shadow-sm space-y-4 hover:shadow-lg transition-all"
			>
				<div class="flex justify-between items-start">
					<div class="h-12 w-12 rounded-2xl bg-gray-50 flex items-center justify-center text-xl">
						{stat.icon}
					</div>
					<span class="text-[10px] font-black text-green-500 bg-green-50 px-2 py-1 rounded-lg">
						{stat.grow}
					</span>
				</div>
				<div>
					<p class="text-[11px] font-black text-gray-400 uppercase tracking-widest">{stat.label}</p>
					<p class="text-xl font-black text-[#0a2e31] mt-1">{stat.value}</p>
				</div>
			</div>
		{/each}
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
		<!-- Table -->
		<div class="lg:col-span-2 bg-white rounded-[2.5rem] border border-gray-100 shadow-sm overflow-hidden flex flex-col">
			<div class="p-8 border-b border-gray-50 flex justify-between items-center">
				<h3 class="font-black text-[#0a2e31] uppercase text-xs tracking-widest">Pendaftaran Terbaru</h3>
				<button class="text-[10px] font-black text-[#217b84] hover:underline">Lihat Semua</button>
			</div>
			<div class="overflow-x-auto flex-1">
				<table class="w-full text-left">
					<thead>
						<tr class="bg-gray-50/50">
							<th class="px-8 py-4 text-[10px] font-black text-gray-400 uppercase tracking-widest">Pengguna</th>
							<th class="px-8 py-4 text-[10px] font-black text-gray-400 uppercase tracking-widest">Waktu</th>
							<th class="px-8 py-4 text-[10px] font-black text-gray-400 uppercase tracking-widest">Status</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-gray-50">
						{#each recentRegistrations as reg}
							<tr class="hover:bg-gray-50/50 transition-colors">
								<td class="px-8 py-5">
									<p class="text-sm font-black text-[#0a2e31]">{reg.user}</p>
									<p class="text-[10px] text-gray-400 font-bold">{reg.email}</p>
								</td>
								<td class="px-8 py-5 text-xs font-bold text-gray-500">{reg.date}</td>
								<td class="px-8 py-5">
									<span class="px-3 py-1 rounded-full text-[9px] font-black uppercase tracking-tight {reg.status === 'Active' ? 'bg-green-50 text-green-600' : 'bg-orange-50 text-orange-600'}">
										{reg.status}
									</span>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>

		<!-- Infrastructure Status -->
		<div class="bg-[#0a2e31] p-10 rounded-[2.5rem] text-white shadow-xl space-y-8 relative overflow-hidden group">
			<div class="absolute -right-10 -top-10 h-40 w-40 bg-white/5 rounded-full group-hover:scale-110 transition-transform duration-1000"></div>
			
			<h3 class="font-black text-teal-400 uppercase text-xs tracking-[0.2em] relative z-10">Infrastruktur</h3>
			
			<div class="space-y-6 relative z-10">
				<div class="flex justify-between items-center">
					<span class="text-sm font-bold text-white/70">Database (Postgres)</span>
					<span class="h-2.5 w-2.5 rounded-full bg-green-500"></span>
				</div>
				<div class="flex justify-between items-center">
					<span class="text-sm font-bold text-white/70">Message Bus (RMQ)</span>
					<span class="h-2.5 w-2.5 rounded-full bg-green-500"></span>
				</div>
				<div class="flex justify-between items-center">
					<span class="text-sm font-bold text-white/70">Cache (Redis)</span>
					<span class="h-2.5 w-2.5 rounded-full bg-green-500"></span>
				</div>
				<div class="flex justify-between items-center">
					<span class="text-sm font-bold text-white/70">Auth Service</span>
					<span class="h-2.5 w-2.5 rounded-full bg-green-500"></span>
				</div>
				<div class="flex justify-between items-center">
					<span class="text-sm font-bold text-white/70">Price Scraper</span>
					<span class="h-2.5 w-2.5 rounded-full bg-orange-500"></span>
				</div>
			</div>

			<button class="w-full py-4 bg-[#217b84] hover:bg-teal-500 text-white text-[10px] font-black uppercase tracking-widest rounded-2xl transition-all relative z-10 mt-4 shadow-lg shadow-black/20">
				Buka Monitoring Grafana
			</button>
		</div>
	</div>
</div>
