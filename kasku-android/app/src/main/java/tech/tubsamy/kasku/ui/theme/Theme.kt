package tech.tubsamy.kasku.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

// ── Palet "Buku Kas" ──────────────────────────────────────────────────────────
// Light (paper)
private val Paper = Color(0xFFF7F5F0)
private val PaperCard = Color(0xFFFCFBF8)
private val Ink = Color(0xFF1A1A1A)
private val Teal = Color(0xFF1F6F6B) // pemasukan / primary
private val Clay = Color(0xFFB4533A) // pengeluaran / alert
private val Muted = Color(0xFF8A8378)
private val Hairline = Color(0xFFDED9CE)

// Dark (ink)
private val InkBg = Color(0xFF141311)
private val InkCard = Color(0xFF201E1B)
private val PaperText = Color(0xFFEDEAE3)
private val TealDark = Color(0xFF5AA8A2)
private val ClayDark = Color(0xFFD3805F)
private val MutedDark = Color(0xFFA8A196)
private val HairlineDark = Color(0xFF34302B)

private val LightColors = lightColorScheme(
    primary = Teal,
    onPrimary = Paper,
    secondary = Clay,
    background = Paper,
    onBackground = Ink,
    surface = PaperCard,
    onSurface = Ink,
    surfaceVariant = PaperCard,
    onSurfaceVariant = Muted,
    error = Clay,
    onError = Paper,
    outline = Muted,
    outlineVariant = Hairline,
)

private val DarkColors = darkColorScheme(
    primary = TealDark,
    onPrimary = InkBg,
    secondary = ClayDark,
    background = InkBg,
    onBackground = PaperText,
    surface = InkCard,
    onSurface = PaperText,
    surfaceVariant = InkCard,
    onSurfaceVariant = MutedDark,
    error = ClayDark,
    onError = InkBg,
    outline = MutedDark,
    outlineVariant = HairlineDark,
)

@Composable
fun KasKuTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit,
) {
    MaterialTheme(
        colorScheme = if (darkTheme) DarkColors else LightColors,
        typography = KasKuTypography,
        content = content,
    )
}
