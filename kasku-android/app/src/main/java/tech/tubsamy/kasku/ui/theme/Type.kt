package tech.tubsamy.kasku.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontVariation
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.em
import androidx.compose.ui.unit.sp
import tech.tubsamy.kasku.R

// Variable font helper — satu file TTF, bobot via axis wght (minSdk 26).
@OptIn(androidx.compose.ui.text.ExperimentalTextApi::class)
private fun variable(resId: Int, weight: FontWeight) =
    Font(resId, weight = weight, variationSettings = FontVariation.Settings(FontVariation.weight(weight.weight)))

/** Instrument Serif — wajah editorial KasKu (judul, angka uang hero). Hanya weight 400 + italic. */
val InstrumentSerif = FontFamily(
    Font(R.font.instrument_serif, weight = FontWeight.Normal),
    Font(R.font.instrument_serif_italic, weight = FontWeight.Normal, style = FontStyle.Italic),
)

/** Instrument Sans — body, label, data. */
val InstrumentSans = FontFamily(
    variable(R.font.instrument_sans_var, FontWeight.Normal),
    variable(R.font.instrument_sans_var, FontWeight.Medium),
    variable(R.font.instrument_sans_var, FontWeight.SemiBold),
    variable(R.font.instrument_sans_var, FontWeight.Bold),
)

/** Font angka uang besar — dipakai MoneyText/PercentText di ui/components. */
val MoneyFont = InstrumentSerif

private val base = Typography()

// Serif editorial selalu weight 400, tracking rapat (mengikuti desain -0.02em).
private fun TextStyle.serif() = copy(
    fontFamily = InstrumentSerif,
    fontWeight = FontWeight.Normal,
    letterSpacing = (-0.01).em,
)

/**
 * Instrument Serif untuk display/headline/titleLarge (identitas editorial),
 * Instrument Sans untuk body/label (keterbacaan data).
 */
val KasKuTypography = Typography(
    displayLarge = base.displayLarge.serif(),
    displayMedium = base.displayMedium.serif(),
    displaySmall = base.displaySmall.serif(),
    headlineLarge = base.headlineLarge.serif(),
    headlineMedium = base.headlineMedium.serif(),
    headlineSmall = base.headlineSmall.serif(),
    titleLarge = base.titleLarge.serif(),
    titleMedium = base.titleMedium.copy(fontFamily = InstrumentSans, fontWeight = FontWeight.SemiBold),
    titleSmall = base.titleSmall.copy(fontFamily = InstrumentSans, fontWeight = FontWeight.SemiBold),
    bodyLarge = base.bodyLarge.copy(fontFamily = InstrumentSans),
    bodyMedium = base.bodyMedium.copy(fontFamily = InstrumentSans),
    bodySmall = base.bodySmall.copy(fontFamily = InstrumentSans),
    labelLarge = base.labelLarge.copy(fontFamily = InstrumentSans, fontWeight = FontWeight.SemiBold),
    labelMedium = base.labelMedium.copy(fontFamily = InstrumentSans, fontWeight = FontWeight.Medium),
    labelSmall = base.labelSmall.copy(fontFamily = InstrumentSans, fontWeight = FontWeight.Medium),
)

/** Eyebrow editorial: UPPERCASE, tracking lebar (SectionLabel meng-uppercase teksnya). */
val LabelEyebrow = TextStyle(
    fontFamily = InstrumentSans,
    fontSize = 11.sp,
    letterSpacing = 0.12.em,
    fontWeight = FontWeight.SemiBold,
)
