package tech.tubsamy.kasku.ui.theme

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp

/*
 * ── Palet "Modern Dark / cinematic" — port 1:1 dari token web kasku-frontend ──
 * Sumber kebenaran: `kasku-frontend/src/routes/layout.css` (@theme). Setiap nilai di bawah
 * harus sama persis dengan CSS custom property yang namanya sepadan, supaya app dan web
 * tidak pelan-pelan drift jadi dua produk yang beda rasa.
 *
 * Catatan semantik yang gampang menjebak (ikut konvensi web):
 *   - `Ink` BUKAN warna gelap. Ink = teks utama (terang) SEKALIGUS warna panel "inverted"
 *     (di web: `bg-ink` = panel terang dengan teks gelap — dipakai di panel brand login).
 *   - `Card` = permukaan gelap SEKALIGUS warna teks di atas tombol berwarna
 *     (di web: `bg-teal text-card`). Makanya onPrimary/onError = Card, bukan putih.
 *   - `Mint` = teal gelap, khusus dipakai DI ATAS panel terang (bg-ink). Jangan dipakai
 *     di atas paper/card — kontrasnya gagal.
 */

// Permukaan gelap — sengaja hindari #000 murni supaya elevasi masih terbaca.
private val Paper = Color(0xFF0A0E14) // kanvas app  (--color-paper)
private val Card = Color(0xFF10161F) // kartu/panel  (--color-card)
private val Field = Color(0xFF161E29) // input/inset  (--color-field)

// Ink + aksen — semua sudah dicek kontras AA di atas Paper/Card.
private val Ink = Color(0xFFE9EDF4) // teks utama   (--color-ink)
private val Teal = Color(0xFF2DD4BF) // aksi utama / positif (--color-teal)
private val Mint = Color(0xFF0F766E) // teal gelap di atas panel terang (--color-mint)
private val Clay = Color(0xFFF4795B) // danger / lewat anggaran (--color-clay)
private val Gold = Color(0xFFE0B357) // caution / anggaran mendekati limit (--color-gold)

/*
 * Turunan opacity ink di-flatten jadi warna solid: Compose menggambar teks/garis dengan
 * alpha per-elemen, tapi menyetel colorScheme dengan warna ber-alpha bikin hasil tak
 * konsisten saat elemen ditumpuk. Nilai di bawah = hasil komposit ink di atas paper.
 */
private val Muted = Color(0xFF98A2B3) // ≈ ink/60 — teks sekunder, kontras 7.3:1 vs paper
private val Line = Color(0xFF232A35) // ≈ ink/10 — hairline pemisah baris
private val Border = Color(0xFF313B49) // ≈ ink/18 — border field/chip

// Warna brand untuk pemakaian langsung (di luar colorScheme).
val KasKuTeal = Teal
val KasKuClay = Clay
val KasKuGold = Gold
val KasKuField = Field
val KasKuInk = Ink // panel "inverted" terang (panel brand login)
val KasKuMint = Mint // aksen "Ku" DI ATAS panel terang KasKuInk

/**
 * Satu-satunya skema warna. Web mengunci `color-scheme: dark` tanpa varian terang, jadi app
 * ikut: tidak ada light mode yang perlu dirawat, dan tidak ada permukaan yang lolos tanpa
 * dicek kontrasnya.
 */
private val KasKuColors = darkColorScheme(
    primary = Teal,
    onPrimary = Card,
    primaryContainer = Teal, // FAB & tombol utama = teal solid (web: bg-teal)
    onPrimaryContainer = Card,
    secondary = Clay,
    onSecondary = Card,
    // Chip/filter terpilih = panel ink terang — idiom "inverted" milik web.
    secondaryContainer = Ink,
    onSecondaryContainer = Card,
    tertiary = Gold,
    onTertiary = Card,
    background = Paper,
    onBackground = Ink,
    surface = Card,
    onSurface = Ink,
    surfaceVariant = Field, // field/inset
    onSurfaceVariant = Muted,
    surfaceTint = Color.Transparent, // flat: tanpa tonal overlay ungu bawaan M3
    // Tangga surfaceContainer diisi manual; default M3 memberi abu bersemu ungu yang
    // langsung terlihat asing di menu, dropdown, dan AlertDialog.
    surfaceContainerLowest = Paper,
    surfaceContainerLow = Card,
    surfaceContainer = Card,
    surfaceContainerHigh = Field,
    surfaceContainerHighest = Field,
    inverseSurface = Ink, // snackbar = panel terang
    inverseOnSurface = Card,
    error = Clay,
    onError = Card,
    errorContainer = Field,
    onErrorContainer = Clay,
    outline = Border,
    outlineVariant = Line,
    scrim = Color.Black,
)

// Radius mengikuti skala web: input rounded-xl (12), kartu rounded-2xl (16),
// sheet/panel rounded-3xl (24). Tombol pill diatur per-komponen (CircleShape).
private val KasKuShapes = Shapes(
    extraSmall = RoundedCornerShape(8.dp),
    small = RoundedCornerShape(12.dp),
    medium = RoundedCornerShape(16.dp),
    large = RoundedCornerShape(20.dp),
    extraLarge = RoundedCornerShape(28.dp),
)

/**
 * Tema KasKu — gelap permanen, sepadan dengan web.
 *
 * Parameter [darkTheme] dipertahankan supaya call site & @Preview lama tidak pecah, tapi
 * nilainya diabaikan: app hanya punya satu skema, jadi tidak ada jalur render yang bisa
 * lolos tanpa teruji.
 */
@Composable
@Suppress("UNUSED_PARAMETER")
fun KasKuTheme(
    darkTheme: Boolean = true,
    content: @Composable () -> Unit,
) {
    MaterialTheme(
        colorScheme = KasKuColors,
        typography = KasKuTypography,
        shapes = KasKuShapes,
        content = content,
    )
}
