<script lang="ts">
	import { resolve } from '$app/paths';

	const lastUpdated = '18 Mei 2026';
</script>

<svelte:head>
	<title>Kebijakan Privasi — KasKu</title>
	<meta
		name="description"
		content="Kebijakan privasi KasKu — bagaimana kami mengumpulkan, menyimpan, dan melindungi data finansial Anda."
	/>
</svelte:head>

<div class="min-h-screen bg-paper">
	<div class="mx-auto max-w-3xl px-6 lg:px-10">
		<!-- Nav -->
		<header class="flex items-center justify-between border-b border-ink/10 py-6">
			<a href={resolve('/')} class="font-serif text-[26px] leading-none tracking-tight text-ink">
				Kas<em class="text-teal">Ku</em>
			</a>
			<a href={resolve('/')} class="text-[13.5px] font-medium text-ink/60 hover:text-ink">
				← Beranda
			</a>
		</header>

		<main class="max-w-2xl space-y-10 py-16">
			<header class="space-y-3 border-b border-ink/10 pb-8">
				<p class="text-[12px] font-semibold tracking-[0.16em] text-teal uppercase">Dokumen Hukum</p>
				<h1 class="font-serif text-4xl leading-[1.05] tracking-tight text-ink sm:text-5xl">
					Kebijakan Privasi
				</h1>
				<p class="text-sm text-ink/60">Terakhir diperbarui: {lastUpdated}</p>
			</header>

			<section class="space-y-4 text-[15px] leading-relaxed text-ink/70">
				<p>
					KasKu (dioperasikan oleh TubsAMY) menghormati privasi Anda dan berkomitmen melindungi data
					pribadi Anda sesuai dengan UU Perlindungan Data Pribadi No. 27 Tahun 2022 (UU PDP)
					Republik Indonesia dan, jika berlaku, General Data Protection Regulation (GDPR).
				</p>
				<p>
					Dokumen ini menjelaskan jenis data yang kami kumpulkan, tujuan pemrosesan, dasar hukum,
					hak Anda sebagai subjek data, serta cara kami menjaga keamanan dan kerahasiaan informasi
					Anda.
				</p>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">1. Data yang Kami Kumpulkan</h2>
				<ul class="list-disc space-y-2 pl-6 text-[15px] leading-relaxed text-ink/70">
					<li>
						<strong>Data akun</strong>: nama pengguna, alamat email, kata sandi (di-hash dengan
						Argon2id).
					</li>
					<li>
						<strong>Data finansial</strong>: akun keuangan, transaksi, kategori, instrumen investasi
						— disimpan terisolasi per pengguna dalam schema multi-tenant.
					</li>
					<li>
						<strong>Data teknis</strong>: alamat IP, user agent, log aktivitas (untuk keamanan dan
						audit, tanpa konten transaksi mendetail).
					</li>
					<li>
						<strong>Data pembayaran</strong>: kami tidak menyimpan nomor kartu — diproses oleh
						Midtrans (PCI-DSS compliant).
					</li>
				</ul>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">2. Tujuan Pemrosesan</h2>
				<ul class="list-disc space-y-2 pl-6 text-[15px] leading-relaxed text-ink/70">
					<li>Menyediakan dan memelihara layanan tracking keuangan.</li>
					<li>Autentikasi, otorisasi, dan keamanan akun.</li>
					<li>Pemrosesan pembayaran langganan.</li>
					<li>
						Komunikasi terkait layanan (verifikasi email, reset password, notifikasi tagihan).
					</li>
					<li>Kepatuhan audit dan kewajiban hukum.</li>
				</ul>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">3. Hak Anda</h2>
				<p class="text-[15px] leading-relaxed text-ink/70">
					Sebagai subjek data, Anda berhak untuk:
				</p>
				<ul class="list-disc space-y-2 pl-6 text-[15px] leading-relaxed text-ink/70">
					<li>Mengakses, memperbaiki, atau menghapus data pribadi Anda.</li>
					<li>Membatasi atau menolak pemrosesan tertentu.</li>
					<li>Portabilitas data — mengunduh data Anda dalam format CSV.</li>
					<li>Mencabut persetujuan kapan saja (akun akan dinonaktifkan).</li>
					<li>Mengajukan keluhan ke otoritas pelindung data.</li>
				</ul>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">4. Keamanan Data</h2>
				<p class="text-[15px] leading-relaxed text-ink/70">
					Kami menerapkan langkah-langkah keamanan teknis dan organisasi yang sesuai: enkripsi TLS
					untuk data dalam transit, password di-hash dengan Argon2id, isolasi data per tenant, audit
					log untuk semua aksi administratif, dan review kerentanan berkala terhadap OWASP Top 10.
				</p>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">5. Retensi Data</h2>
				<p class="text-[15px] leading-relaxed text-ink/70">
					Data finansial disimpan selama akun Anda aktif. Setelah penghapusan akun, data dihapus
					dalam waktu 30 hari kecuali ada kewajiban hukum untuk menyimpannya lebih lama (mis.
					catatan pembayaran untuk keperluan pajak/audit).
				</p>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">6. Pihak Ketiga</h2>
				<ul class="list-disc space-y-2 pl-6 text-[15px] leading-relaxed text-ink/70">
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
				<h2 class="font-serif text-xl text-ink">7. Kontak</h2>
				<p class="text-[15px] leading-relaxed text-ink/70">
					Pertanyaan, permintaan akses data, atau keluhan dapat diajukan ke:
					<a href="mailto:admin@tubsamy.tech" class="text-teal hover:underline"
						>admin@tubsamy.tech</a
					>.
				</p>
			</section>

			<section class="space-y-3 border-t border-ink/10 pt-8">
				<h2 class="font-serif text-xl text-ink">Perubahan Kebijakan</h2>
				<p class="text-[15px] leading-relaxed text-ink/70">
					Kami dapat memperbarui kebijakan ini dari waktu ke waktu. Perubahan material akan
					diberitahukan melalui email dan/atau notifikasi di aplikasi.
				</p>
			</section>
		</main>
	</div>

	<!-- Footer -->
	<footer class="border-t border-ink/10">
		<div
			class="mx-auto flex max-w-3xl items-center justify-between px-6 py-5 text-xs text-ink/45 lg:px-10"
		>
			<span>© 2026 KasKu</span>
			<span class="flex gap-5">
				<a href={resolve('/privacy')} class="hover:text-teal">Privasi</a>
				<a href={resolve('/terms')} class="hover:text-teal">Ketentuan</a>
			</span>
		</div>
	</footer>
</div>
