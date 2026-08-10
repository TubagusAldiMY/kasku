package tech.tubsamy.kasku.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.em
import androidx.compose.ui.unit.sp
import tech.tubsamy.kasku.R

/*
 * Pasangan huruf sama persis dengan web (`--font-sans` / `--font-serif` di layout.css):
 *   Poppins          → body, label, data. Geometric sans, jadi wajah "SaaS" produk.
 *   Instrument Serif → judul & angka uang hero. Satu-satunya sisa identitas editorial.
 * Keduanya di-bundle sebagai aset (bukan Downloadable Fonts) supaya tampilan tidak
 * bergantung pada Google Play Services maupun koneksi saat first launch.
 */

/** Poppins — hanya 4 bobot yang benar-benar dipakai, sisanya cuma menambah ukuran APK. */
val Poppins = FontFamily(
    Font(R.font.poppins_regular, FontWeight.Normal),
    Font(R.font.poppins_medium, FontWeight.Medium),
    Font(R.font.poppins_semibold, FontWeight.SemiBold),
    Font(R.font.poppins_bold, FontWeight.Bold),
)

/** Instrument Serif — judul & angka uang. Hanya tersedia weight 400 + italic. */
val InstrumentSerif = FontFamily(
    Font(R.font.instrument_serif, weight = FontWeight.Normal),
    Font(R.font.instrument_serif_italic, weight = FontWeight.Normal, style = FontStyle.Italic),
)

/** Font angka uang besar — dipakai MoneyText/PercentText di ui/components. */
val MoneyFont = InstrumentSerif

private val base = Typography()

// Serif selalu weight 400 dengan tracking rapat — web memakai `tracking-tight` di tiap
// angka/judul serif; memaksa bold di sini justru menghasilkan faux-bold yang buram.
private fun TextStyle.serif() = copy(
    fontFamily = InstrumentSerif,
    fontWeight = FontWeight.Normal,
    letterSpacing = (-0.02).em,
)

// Poppins sudah cukup lapang secara natural; letterSpacing bawaan Material (0.1–0.5sp)
// membuatnya terbaca renggang dan tidak sepadan dengan web yang memakai tracking normal.
private fun TextStyle.poppins(weight: FontWeight = FontWeight.Normal) = copy(
    fontFamily = Poppins,
    fontWeight = weight,
    letterSpacing = 0.sp,
)

/**
 * Instrument Serif untuk display/headline/titleLarge (angka & judul),
 * Poppins untuk title kecil/body/label (keterbacaan data).
 */
val KasKuTypography = Typography(
    displayLarge = base.displayLarge.serif(),
    displayMedium = base.displayMedium.serif(),
    displaySmall = base.displaySmall.serif(),
    headlineLarge = base.headlineLarge.serif(),
    headlineMedium = base.headlineMedium.serif(),
    headlineSmall = base.headlineSmall.serif(),
    titleLarge = base.titleLarge.serif(),
    titleMedium = base.titleMedium.poppins(FontWeight.SemiBold),
    titleSmall = base.titleSmall.poppins(FontWeight.SemiBold),
    bodyLarge = base.bodyLarge.poppins(),
    bodyMedium = base.bodyMedium.poppins(),
    bodySmall = base.bodySmall.poppins(),
    labelLarge = base.labelLarge.poppins(FontWeight.SemiBold),
    labelMedium = base.labelMedium.poppins(FontWeight.Medium),
    labelSmall = base.labelSmall.poppins(FontWeight.Medium),
)

/** Eyebrow: UPPERCASE, tracking lebar — padanan `text-[11px] tracking-[0.12em] uppercase` di web. */
val LabelEyebrow = TextStyle(
    fontFamily = Poppins,
    fontSize = 11.sp,
    letterSpacing = 0.12.em,
    fontWeight = FontWeight.SemiBold,
)
