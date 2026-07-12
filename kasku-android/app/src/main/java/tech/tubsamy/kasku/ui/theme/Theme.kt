package tech.tubsamy.kasku.ui.theme

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp

// ── Palet "Editorial" — paper cream + serif, mengikuti handoff ReDesign/ (web kasku.id) ──
private val Paper = Color(0xFFF7F6F2) // canvas krem
private val Bright = Color(0xFFFDFCFA) // kartu/field lebih terang
private val Ink = Color(0xFF12312E) // teks utama, hijau-hitam pekat
private val Teal = Color(0xFF1A5F66) // aksi utama / pemasukan / positif
private val TealSoft = Color(0xFF7FC7BD) // aksen di atas ink gelap
private val Clay = Color(0xFFA4502F) // bahaya / pengeluaran / lewat anggaran
private val Gold = Color(0xFF8A6A1F) // peringatan / cache / anggaran mendekati limit
private val Muted = Color(0xFF6E807C) // teks sekunder (ink 60% di atas paper)
private val Line = Color(0xFFE3E4DF) // hairline pemisah baris (ink ~10%)
private val Border = Color(0xFFBEC5C1) // border field/chip (ink ~25%)

// Warna brand tambahan untuk pemakaian langsung.
val KasKuTeal = Teal
val KasKuClay = Clay
val KasKuGold = Gold
val KasKuInk = Ink // panel brand gelap (login, kartu saldo)
val KasKuTealSoft = TealSoft // aksen "Ku" di atas ink

/** Default: editorial terang. */
private val LightColors = lightColorScheme(
    primary = Teal,
    onPrimary = Paper,
    primaryContainer = Teal, // FAB & tombol utama = teal solid
    onPrimaryContainer = Paper,
    secondary = Clay,
    onSecondary = Paper,
    secondaryContainer = Ink, // chip/filter terpilih = ink solid (desain)
    onSecondaryContainer = Paper,
    tertiary = Gold,
    onTertiary = Paper,
    background = Paper,
    onBackground = Ink,
    surface = Bright,
    onSurface = Ink,
    surfaceVariant = Color(0xFFECEAE4), // field/chip netral
    onSurfaceVariant = Muted,
    surfaceTint = Color.Transparent, // flat: tanpa tonal overlay
    error = Clay,
    onError = Paper,
    outline = Border,
    outlineVariant = Line,
)

/** Fallback gelap: panel ink (seperti panel brand di desain login). */
private val DarkColors = darkColorScheme(
    primary = TealSoft,
    onPrimary = Ink,
    primaryContainer = Teal,
    onPrimaryContainer = Paper,
    secondary = Color(0xFFD98E6B),
    onSecondary = Ink,
    secondaryContainer = Paper,
    onSecondaryContainer = Ink,
    background = Ink,
    onBackground = Paper,
    surface = Color(0xFF1A3C38),
    onSurface = Paper,
    surfaceVariant = Color(0xFF224844),
    onSurfaceVariant = Color(0xFFA9BDB8),
    error = Color(0xFFD98E6B),
    onError = Ink,
    outline = Color(0xFF3C5D58),
    outlineVariant = Color(0xFF2A4A46),
)

// Sudut editorial: field lembut, tombol pill via komponen.
private val KasKuShapes = Shapes(
    extraSmall = RoundedCornerShape(6.dp),
    small = RoundedCornerShape(10.dp),
    medium = RoundedCornerShape(14.dp),
    large = RoundedCornerShape(18.dp),
    extraLarge = RoundedCornerShape(24.dp),
)

/** Default TERANG (editorial paper). Kirim darkTheme=true untuk fallback ink. */
@Composable
fun KasKuTheme(
    darkTheme: Boolean = false,
    content: @Composable () -> Unit,
) {
    MaterialTheme(
        colorScheme = if (darkTheme) DarkColors else LightColors,
        typography = KasKuTypography,
        shapes = KasKuShapes,
        content = content,
    )
}
