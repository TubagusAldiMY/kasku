package tech.tubsamy.kasku.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

// ── Palet "Buku Kas" — hex PERSIS dari web (kasku-frontend/src/routes/layout.css) ──
private val Paper = Color(0xFFECEAE4) // canvas
private val Card = Color(0xFFF7F6F2) // kartu/panel
private val Field = Color(0xFFFDFCFA) // input, teks di atas warna gelap
private val Ink = Color(0xFF12312E) // teks utama (dark teal-green)
private val Teal = Color(0xFF1A5F66) // aksi utama / pemasukan / positif
private val Mint = Color(0xFF7FC7BD) // teal terang di latar gelap
private val Clay = Color(0xFFA4502F) // bahaya / pengeluaran / negatif
private val Gold = Color(0xFF8A6A1F) // caution / tag kategori
private val Muted = Color(0xFF5F6E6A) // teks sekunder (ink diredam)
private val Hairline = Color(0xFFDAD7CE) // garis pemisah

// Warna brand tambahan untuk pemakaian langsung (tag/transfer) — di luar colorScheme.
val KasKuGold = Gold
val KasKuMint = Mint

private val LightColors = lightColorScheme(
    primary = Teal,
    onPrimary = Field,
    primaryContainer = Teal, // FAB & container = teal, bukan lavender Material default
    onPrimaryContainer = Field,
    surfaceTint = Teal, // tonal overlay kartu ke arah teal (on-brand), bukan ungu
    secondary = Clay,
    onSecondary = Field,
    secondaryContainer = Mint, // chip terpilih = teal lembut, bukan lavender Material default
    onSecondaryContainer = Ink,
    background = Paper,
    onBackground = Ink,
    surface = Card,
    onSurface = Ink,
    surfaceVariant = Card,
    onSurfaceVariant = Muted,
    error = Clay,
    onError = Field,
    outline = Muted,
    outlineVariant = Hairline,
)

// Varian gelap konsisten brand (teal→mint). Default app tetap light (samakan web).
private val InkBg = Color(0xFF0F1F1D)
private val InkCard = Color(0xFF16302C)
private val DarkColors = darkColorScheme(
    primary = Mint,
    onPrimary = InkBg,
    secondary = Color(0xFFC6714C),
    background = InkBg,
    onBackground = Paper,
    surface = InkCard,
    onSurface = Paper,
    surfaceVariant = InkCard,
    onSurfaceVariant = Color(0xFF8CA39D),
    error = Color(0xFFC6714C),
    onError = InkBg,
    outline = Color(0xFF8CA39D),
    outlineVariant = Color(0xFF29403B),
)

/**
 * Default LIGHT (paper) agar identik dengan web yang light-only. Kirim darkTheme=true
 * bila nanti ingin mode gelap.
 */
@Composable
fun KasKuTheme(
    darkTheme: Boolean = false,
    content: @Composable () -> Unit,
) {
    MaterialTheme(
        colorScheme = if (darkTheme) DarkColors else LightColors,
        typography = KasKuTypography,
        content = content,
    )
}
