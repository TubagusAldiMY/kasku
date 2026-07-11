package tech.tubsamy.kasku.ui.theme

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp

// ── Palet "Midnight" — fintech gelap premium (Revolut/Robinhood vibe) ──
private val Bg = Color(0xFF0B0F14) // canvas near-black
private val Card = Color(0xFF151B23) // kartu/panel
private val Elevated = Color(0xFF1C2530) // field, chip, kartu terangkat
private val Mint = Color(0xFF34D399) // aksi utama / pemasukan / positif
private val MintDeep = Color(0xFF0E3A2C) // container mint gelap
private val MintSoft = Color(0xFFA7F3D0) // teks di atas container mint
private val Coral = Color(0xFFFB7185) // bahaya / pengeluaran / negatif
private val Ink = Color(0xFFE6EDF3) // teks utama (near-white cool)
private val Muted = Color(0xFF8A97A6) // teks sekunder
private val Line = Color(0xFF232C36) // hairline / border kartu
private val Outline = Color(0xFF2E3A46)

// Warna brand tambahan untuk pemakaian langsung.
val KasKuMint = Mint
val KasKuCoral = Coral

private val DarkColors = darkColorScheme(
    primary = Mint,
    onPrimary = Color(0xFF04261B),
    primaryContainer = Mint, // FAB & tombol utama = mint solid
    onPrimaryContainer = Color(0xFF04261B),
    secondary = Coral,
    onSecondary = Color(0xFF2A0A10),
    secondaryContainer = MintDeep, // chip terpilih = mint gelap
    onSecondaryContainer = MintSoft,
    background = Bg,
    onBackground = Ink,
    surface = Card,
    onSurface = Ink,
    surfaceVariant = Elevated,
    onSurfaceVariant = Muted,
    surfaceTint = Mint,
    error = Coral,
    onError = Color(0xFF2A0A10),
    outline = Outline,
    outlineVariant = Line,
)

// Fallback light (jarang dipakai — app default gelap). Netral modern, bukan paper kuno.
private val LightColors = lightColorScheme(
    primary = Color(0xFF0F766E),
    onPrimary = Color(0xFFFFFFFF),
    secondary = Color(0xFFE11D48),
    background = Color(0xFFF7F8FA),
    onBackground = Color(0xFF0F172A),
    surface = Color(0xFFFFFFFF),
    onSurface = Color(0xFF0F172A),
    surfaceVariant = Color(0xFFEEF1F5),
    onSurfaceVariant = Color(0xFF64748B),
    error = Color(0xFFE11D48),
    outline = Color(0xFFCBD5E1),
    outlineVariant = Color(0xFFE2E8F0),
)

// Sudut membulat modern.
private val KasKuShapes = Shapes(
    extraSmall = RoundedCornerShape(10.dp),
    small = RoundedCornerShape(14.dp),
    medium = RoundedCornerShape(18.dp),
    large = RoundedCornerShape(24.dp),
    extraLarge = RoundedCornerShape(28.dp),
)

/** Default GELAP (midnight premium). Kirim darkTheme=false untuk fallback light. */
@Composable
fun KasKuTheme(
    darkTheme: Boolean = true,
    content: @Composable () -> Unit,
) {
    MaterialTheme(
        colorScheme = if (darkTheme) DarkColors else LightColors,
        typography = KasKuTypography,
        shapes = KasKuShapes,
        content = content,
    )
}
