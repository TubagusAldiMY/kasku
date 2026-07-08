<script lang="ts">
	import { resolve } from '$app/paths';

	const lastUpdated = '18 Mei 2026';
</script>

<svelte:head>
	<title>Syarat & Ketentuan — KasKu</title>
	<meta
		name="description"
		content="Syarat dan ketentuan penggunaan layanan KasKu — aplikasi pengelolaan keuangan personal multi-tenant."
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
					Syarat &amp; Ketentuan
				</h1>
				<p class="text-sm text-ink/60">Terakhir diperbarui: {lastUpdated}</p>
			</header>

			<section class="space-y-3 text-[15px] leading-relaxed text-ink/70">
				<p>
					Dengan mendaftarkan akun, mengakses, atau menggunakan KasKu (selanjutnya disebut
					"Layanan"), Anda dianggap telah membaca, memahami, dan menyetujui untuk terikat oleh
					syarat dan ketentuan ini. Jika tidak menyetujui, mohon hentikan penggunaan Layanan.
				</p>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">1. Definisi</h2>
				<ul class="list-disc space-y-2 pl-6 text-[15px] leading-relaxed text-ink/70">
					<li>
						<strong>"Layanan"</strong> berarti aplikasi KasKu beserta seluruh fitur, situs, dan API terkait.
					</li>
					<li>
						<strong>"Penyedia"</strong> berarti TubsAMY (admin@tubsamy.tech) sebagai operator Layanan.
					</li>
					<li><strong>"Pengguna"</strong> berarti individu yang telah membuat akun di Layanan.</li>
					<li><strong>"Konten"</strong> berarti data finansial yang Anda masukkan ke Layanan.</li>
				</ul>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">2. Akun &amp; Keamanan</h2>
				<ul class="list-disc space-y-2 pl-6 text-[15px] leading-relaxed text-ink/70">
					<li>Anda bertanggung jawab penuh atas kerahasiaan kredensial akun.</li>
					<li>Satu akun untuk satu individu; akun bersama tidak diizinkan.</li>
					<li>
						Aktivitas mencurigakan harap segera dilaporkan ke
						<a href="mailto:admin@tubsamy.tech" class="text-teal hover:underline"
							>admin@tubsamy.tech</a
						>.
					</li>
				</ul>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">3. Penggunaan yang Diizinkan</h2>
				<p class="text-[15px] leading-relaxed text-ink/70">
					Layanan disediakan untuk penggunaan personal. Anda dilarang:
				</p>
				<ul class="list-disc space-y-2 pl-6 text-[15px] leading-relaxed text-ink/70">
					<li>Mengakses atau mencoba mengakses data Pengguna lain.</li>
					<li>
						Melakukan reverse engineering, scraping, atau aktivitas otomatis terhadap API tanpa izin
						tertulis.
					</li>
					<li>Mengunggah konten ilegal, malware, atau yang melanggar hak pihak ketiga.</li>
					<li>
						Menggunakan Layanan untuk pencucian uang, pendanaan terorisme, atau aktivitas ilegal
						lainnya.
					</li>
				</ul>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">4. Langganan &amp; Pembayaran</h2>
				<ul class="list-disc space-y-2 pl-6 text-[15px] leading-relaxed text-ink/70">
					<li>Layanan tersedia dalam tier gratis (FREE) dan berbayar (PRO/PREMIUM).</li>
					<li>Pembayaran diproses oleh Midtrans; tagihan otomatis sesuai periode yang dipilih.</li>
					<li>
						Pengembalian dana mengikuti kebijakan refund pro-rata; ajukan melalui email dalam 7
						hari.
					</li>
					<li>Penyedia dapat mengubah harga dengan pemberitahuan minimal 30 hari sebelumnya.</li>
				</ul>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">5. Kepemilikan Konten</h2>
				<p class="text-[15px] leading-relaxed text-ink/70">
					Anda mempertahankan kepemilikan penuh atas Konten yang Anda masukkan. Penyedia hanya
					mendapat lisensi terbatas untuk memproses Konten guna menyediakan Layanan (mis.
					menampilkan, menyinkronkan antar perangkat, mencadangkan).
				</p>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">6. Disclaimer &amp; Tanggung Jawab</h2>
				<ul class="list-disc space-y-2 pl-6 text-[15px] leading-relaxed text-ink/70">
					<li>
						Layanan disediakan <em>as-is</em>. Penyedia tidak memberikan saran investasi atau pajak
						— informasi yang ditampilkan bersifat referensial.
					</li>
					<li>
						Harga aset (kripto, emas, saham) berasal dari pihak ketiga dan dapat tertunda. Jangan
						gunakan sebagai satu-satunya sumber untuk keputusan finansial material.
					</li>
					<li>
						Sejauh diizinkan hukum, tanggung jawab Penyedia atas kerugian terbatas pada nominal
						pembayaran terakhir Anda untuk Layanan.
					</li>
				</ul>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">7. Penghentian Akun</h2>
				<p class="text-[15px] leading-relaxed text-ink/70">
					Anda dapat menghapus akun kapan saja melalui menu Profil. Penyedia dapat menangguhkan atau
					menutup akun yang melanggar Syarat ini, dengan pemberitahuan sebelumnya jika memungkinkan.
				</p>
			</section>

			<section class="space-y-3">
				<h2 class="font-serif text-xl text-ink">8. Hukum yang Berlaku</h2>
				<p class="text-[15px] leading-relaxed text-ink/70">
					Syarat ini diatur oleh hukum Republik Indonesia. Sengketa yang timbul akan diselesaikan
					melalui musyawarah; bila gagal, melalui pengadilan yang berwenang di Jakarta.
				</p>
			</section>

			<section class="space-y-3 border-t border-ink/10 pt-8">
				<h2 class="font-serif text-xl text-ink">9. Perubahan Syarat</h2>
				<p class="text-[15px] leading-relaxed text-ink/70">
					Penyedia dapat memperbarui Syarat ini sewaktu-waktu. Perubahan akan diumumkan melalui
					email dan/atau notifikasi di aplikasi. Penggunaan berkelanjutan setelah perubahan
					diberlakukan merupakan persetujuan Anda terhadap Syarat yang baru.
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
