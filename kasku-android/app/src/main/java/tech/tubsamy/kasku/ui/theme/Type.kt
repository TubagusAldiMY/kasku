package tech.tubsamy.kasku.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp
import tech.tubsamy.kasku.R

/** Instrument Serif — wajah editorial KasKu (wordmark, judul, angka uang besar). */
val InstrumentSerif = FontFamily(
    Font(R.font.instrument_serif_regular, FontWeight.Normal),
    Font(R.font.instrument_serif_italic, FontWeight.Normal, FontStyle.Italic),
)

// Body & data pakai sans default sistem (kontras dengan serif display) — tetap FontFamily.Default.
private val base = Typography()

/**
 * Serif untuk display/headline/title (editorial), sans untuk body/label (keterbacaan data).
 * Angka uang besar memakai style display (lihat MoneyText di ui/components).
 */
val KasKuTypography = Typography(
    displayLarge = base.displayLarge.copy(fontFamily = InstrumentSerif, fontWeight = FontWeight.Normal),
    displayMedium = base.displayMedium.copy(fontFamily = InstrumentSerif, fontWeight = FontWeight.Normal),
    displaySmall = base.displaySmall.copy(fontFamily = InstrumentSerif, fontWeight = FontWeight.Normal),
    headlineLarge = base.headlineLarge.copy(fontFamily = InstrumentSerif, fontWeight = FontWeight.Normal),
    headlineMedium = base.headlineMedium.copy(fontFamily = InstrumentSerif, fontWeight = FontWeight.Normal),
    headlineSmall = base.headlineSmall.copy(fontFamily = InstrumentSerif, fontWeight = FontWeight.Normal),
    titleLarge = base.titleLarge.copy(fontFamily = InstrumentSerif, fontWeight = FontWeight.Normal),
    // title/body/label lain tetap sans default.
)

/** Label seksi ala ledger: UPPERCASE, tracking lebar, kecil. */
val LabelEyebrow = TextStyle(
    fontSize = 11.sp,
    letterSpacing = 1.5.sp,
    fontWeight = FontWeight.Medium,
)
