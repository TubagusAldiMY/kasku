package tech.tubsamy.kasku.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

// Token warna KasKu (port dari editorial theme web).
private val Teal = Color(0xFF1F6F6B)
private val Ink = Color(0xFF1A1A1A)
private val Paper = Color(0xFFF7F5F0)
private val Clay = Color(0xFFB4533A)

private val LightColors = lightColorScheme(
    primary = Teal,
    onPrimary = Color.White,
    background = Paper,
    onBackground = Ink,
    surface = Color.White,
    onSurface = Ink,
    error = Clay,
)

private val DarkColors = darkColorScheme(
    primary = Teal,
    error = Clay,
)

@Composable
fun KasKuTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit,
) {
    MaterialTheme(
        colorScheme = if (darkTheme) DarkColors else LightColors,
        content = content,
    )
}
