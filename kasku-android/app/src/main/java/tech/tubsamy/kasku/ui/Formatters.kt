package tech.tubsamy.kasku.ui

import java.text.NumberFormat
import java.util.Locale

private val idFormat = NumberFormat.getInstance(Locale.forLanguageTag("id-ID"))

/** 7690000 → "Rp 7.690.000". */
fun formatIdr(amount: Long): String = "Rp " + idFormat.format(amount)
