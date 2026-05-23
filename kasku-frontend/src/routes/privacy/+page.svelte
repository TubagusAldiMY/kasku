<script lang="ts">
	const lastUpdated = '18 Mei 2026';
</script>

<svelte:head>
	<title>Kebijakan Privasi — KasKu</title>
	<meta
		name="description"
		content="Kebijakan privasi KasKu — bagaimana kami mengumpulkan, menyimpan, dan melindungi data finansial Anda."
	/>
</svelte:head>

<main class="mx-auto max-w-3xl space-y-10 px-6 py-16 text-[#0a2e31]">
	<header class="space-y-2 border-b border-gray-100 pb-6">
		<p class="text-[10px] font-black tracking-widest text-gray-400 uppercase">Dokumen Hukum</p>
		<h1 class="text-4xl font-black">Kebijakan Privasi</h1>
		<p class="text-sm font-medium text-gray-500">Terakhir diperbarui: {lastUpdated}</p>
	</header>

	<section class="space-y-4 text-sm leading-relaxed text-gray-700">
		<p>
			KasKu (dioperasikan oleh TubsAMY) menghormati privasi Anda dan berkomitmen melindungi data
			pribadi Anda sesuai dengan UU Perlindungan Data Pribadi No. 27 Tahun 2022 (UU PDP) Republik
			Indonesia dan, jika berlaku, General Data Protection Regulation (GDPR).
		</p>
		<p>
			Dokumen ini menjelaskan jenis data yang kami kumpulkan, tujuan pemrosesan, dasar hukum, hak
			Anda sebagai subjek data, serta cara kami menjaga keamanan dan kerahasiaan informasi Anda.
		</p>
	</section>

	<section class="space-y-3">
		<h2 class="text-xl font-black">1. Data yang Kami Kumpulkan</h2>
		<ul class="list-disc space-y-2 pl-6 text-sm text-gray-700">
			<li>
				<strong>Data akun</strong>: nama pengguna, alamat email, kata sandi (di-hash dengan
				Argon2id).
			</li>
			<li>
				<strong>Data finansial</strong>: akun keuangan, transaksi, kategori, instrumen investasi —
				disimpan terisolasi per pengguna dalam schema multi-tenant.
			</li>
			<li>
				<strong>Data teknis</strong>: alamat IP, user agent, log aktivitas (untuk keamanan dan
				audit, tanpa konten transaksi mendetail).
			</li>
			<li>
				<strong>Data pembayaran</strong>: kami tidak menyimpan nomor kartu — diproses oleh Midtrans
				(PCI-DSS compliant).
			</li>
		</ul>
	</section>

	<section class="space-y-3">
		<h2 class="text-xl font-black">2. Tujuan Pemrosesan</h2>
		<ul class="list-disc space-y-2 pl-6 text-sm text-gray-700">
			<li>Menyediakan dan memelihara layanan tracking keuangan.</li>
			<li>Autentikasi, otorisasi, dan keamanan akun.</li>
			<li>Pemrosesan pembayaran langganan.</li>
			<li>Komunikasi terkait layanan (verifikasi email, reset password, notifikasi tagihan).</li>
			<li>Kepatuhan audit dan kewajiban hukum.</li>
		</ul>
	</section>

	<section class="space-y-3">
		<h2 class="text-xl font-black">3. Hak Anda</h2>
		<p class="text-sm text-gray-700">Sebagai subjek data, Anda berhak untuk:</p>
		<ul class="list-disc space-y-2 pl-6 text-sm text-gray-700">
			<li>Mengakses, memperbaiki, atau menghapus data pribadi Anda.</li>
			<li>Membatasi atau menolak pemrosesan tertentu.</li>
			<li>Portabilitas data — mengunduh data Anda dalam format CSV.</li>
			<li>Mencabut persetujuan kapan saja (akun akan dinonaktifkan).</li>
			<li>Mengajukan keluhan ke otoritas pelindung data.</li>
		</ul>
	</section>

	<section class="space-y-3">
		<h2 class="text-xl font-black">4. Keamanan Data</h2>
		<p class="text-sm text-gray-700">
			Kami menerapkan langkah-langkah keamanan teknis dan organisasi yang sesuai: enkripsi TLS untuk
			data dalam transit, password di-hash dengan Argon2id, isolasi data per tenant, audit log untuk
			semua aksi administratif, dan review kerentanan berkala terhadap OWASP Top 10.
		</p>
	</section>

	<section class="space-y-3">
		<h2 class="text-xl font-black">5. Retensi Data</h2>
		<p class="text-sm text-gray-700">
			Data finansial disimpan selama akun Anda aktif. Setelah penghapusan akun, data dihapus dalam
			waktu 30 hari kecuali ada kewajiban hukum untuk menyimpannya lebih lama (mis. catatan
			pembayaran untuk keperluan pajak/audit).
		</p>
	</section>

	<section class="space-y-3">
		<h2 class="text-xl font-black">6. Pihak Ketiga</h2>
		<ul class="list-disc space-y-2 pl-6 text-sm text-gray-700">
			<li><strong>Midtrans</strong> — pemroses pembayaran.</li>
			<li>
				<strong>CoinGecko, metals.live</strong> — sumber harga pasar (tidak menerima data Anda).
			</li>
			<li>
				<strong>Penyedia infrastruktur cloud</strong> — hosting (terikat kontrak pemrosesan data).
			</li>
		</ul>
	</section>

	<section class="space-y-3">
		<h2 class="text-xl font-black">7. Kontak</h2>
		<p class="text-sm text-gray-700">
			Pertanyaan, permintaan akses data, atau keluhan dapat diajukan ke:
			<a href="mailto:admin@tubsamy.tech" class="font-bold text-[#217b84] hover:underline"
				>admin@tubsamy.tech</a
			>.
		</p>
	</section>

	<section class="space-y-3 border-t border-gray-100 pt-6">
		<h2 class="text-xl font-black">Perubahan Kebijakan</h2>
		<p class="text-sm text-gray-700">
			Kami dapat memperbarui kebijakan ini dari waktu ke waktu. Perubahan material akan
			diberitahukan melalui email dan/atau notifikasi di aplikasi.
		</p>
	</section>
</main>
