package tech.tubsamy.kasku.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontVariation
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp
import tech.tubsamy.kasku.R

// Variable font helper — satu file TTF, bobot via axis wght (minSdk 26).
@OptIn(androidx.compose.ui.text.ExperimentalTextApi::class)
private fun variable(resId: Int, weight: FontWeight) =
    Font(resId, weight = weight, variationSettings = FontVariation.Settings(FontVariation.weight(weight.weight)))

/** Space Grotesk — wajah modern KasKu (wordmark, judul, angka uang). Geometric, techy. */
val SpaceGrotesk = FontFamily(
    variable(R.font.space_grotesk_var, FontWeight.Normal),
    variable(R.font.space_grotesk_var, FontWeight.Medium),
    variable(R.font.space_grotesk_var, FontWeight.SemiBold),
    variable(R.font.space_grotesk_var, FontWeight.Bold),
)

/** Manrope — body & data, sans netral yang bersih. */
val Manrope = FontFamily(
    variable(R.font.manrope_var, FontWeight.Normal),
    variable(R.font.manrope_var, FontWeight.Medium),
    variable(R.font.manrope_var, FontWeight.SemiBold),
    variable(R.font.manrope_var, FontWeight.Bold),
)

/** Font angka uang besar — dipakai MoneyText/PercentText di ui/components. */
val MoneyFont = SpaceGrotesk

private val base = Typography()

/**
 * Space Grotesk untuk display/headline/title (identitas), Manrope untuk body/label
 * (keterbacaan data). Angka uang besar memakai MoneyFont langsung.
 */
val KasKuTypography = Typography(
    displayLarge = base.displayLarge.copy(fontFamily = SpaceGrotesk, fontWeight = FontWeight.Bold),
    displayMedium = base.displayMedium.copy(fontFamily = SpaceGrotesk, fontWeight = FontWeight.Bold),
    displaySmall = base.displaySmall.copy(fontFamily = SpaceGrotesk, fontWeight = FontWeight.Bold),
    headlineLarge = base.headlineLarge.copy(fontFamily = SpaceGrotesk, fontWeight = FontWeight.Bold),
    headlineMedium = base.headlineMedium.copy(fontFamily = SpaceGrotesk, fontWeight = FontWeight.SemiBold),
    headlineSmall = base.headlineSmall.copy(fontFamily = SpaceGrotesk, fontWeight = FontWeight.SemiBold),
    titleLarge = base.titleLarge.copy(fontFamily = SpaceGrotesk, fontWeight = FontWeight.SemiBold),
    titleMedium = base.titleMedium.copy(fontFamily = Manrope, fontWeight = FontWeight.SemiBold),
    titleSmall = base.titleSmall.copy(fontFamily = Manrope, fontWeight = FontWeight.SemiBold),
    bodyLarge = base.bodyLarge.copy(fontFamily = Manrope),
    bodyMedium = base.bodyMedium.copy(fontFamily = Manrope),
    bodySmall = base.bodySmall.copy(fontFamily = Manrope),
    labelLarge = base.labelLarge.copy(fontFamily = Manrope, fontWeight = FontWeight.SemiBold),
    labelMedium = base.labelMedium.copy(fontFamily = Manrope, fontWeight = FontWeight.Medium),
    labelSmall = base.labelSmall.copy(fontFamily = Manrope, fontWeight = FontWeight.Medium),
)

/** Label seksi modern: sentence case, Manrope Medium, tracking halus (bukan eyebrow kuno). */
val LabelEyebrow = TextStyle(
    fontFamily = Manrope,
    fontSize = 13.sp,
    letterSpacing = 0.2.sp,
    fontWeight = FontWeight.Medium,
)
