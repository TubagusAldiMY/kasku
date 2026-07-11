package tech.tubsamy.kasku.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import tech.tubsamy.kasku.ui.formatIdr
import tech.tubsamy.kasku.ui.theme.LabelEyebrow
import tech.tubsamy.kasku.ui.theme.MoneyFont

/** Label seksi modern: sentence case, muted, tracking halus. */
@Composable
fun SectionLabel(text: String, modifier: Modifier = Modifier) {
    Text(
        text = text,
        style = LabelEyebrow,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = modifier,
    )
}

/** Angka uang — Instrument Serif, elemen pahlawan. sign=true → +/- untuk income/expense. */
@Composable
fun MoneyText(
    amount: Long,
    modifier: Modifier = Modifier,
    fontSize: Int = 28,
    color: androidx.compose.ui.graphics.Color = MaterialTheme.colorScheme.onBackground,
    signed: Boolean = false,
) {
    val prefix = if (signed && amount != 0L) (if (amount > 0) "+" else "−") else ""
    Text(
        text = prefix + formatIdr(kotlin.math.abs(amount)),
        fontFamily = MoneyFont,
        fontWeight = FontWeight.Bold,
        fontSize = fontSize.sp,
        color = color,
        modifier = modifier,
    )
}

/** Garis hairline tipis pemisah baris ledger. */
@Composable
fun Hairline(modifier: Modifier = Modifier) {
    HorizontalDivider(
        modifier = modifier,
        thickness = 1.dp,
        color = MaterialTheme.colorScheme.outlineVariant,
    )
}

/** Baris ledger: keterangan kiri (sans), angka serif rata-kanan. */
@Composable
fun LedgerRow(
    label: String,
    amount: Long,
    color: androidx.compose.ui.graphics.Color = MaterialTheme.colorScheme.onBackground,
    sub: String? = null,
    signed: Boolean = false,
) {
    Row(
        modifier = Modifier.fillMaxWidth().padding(vertical = 14.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column {
            Text(label, style = MaterialTheme.typography.bodyLarge)
            if (sub != null) {
                Text(sub, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
        MoneyText(amount, fontSize = 22, color = color, signed = signed, modifier = Modifier)
    }
}

/** Angka persentase serif (untuk savings rate). */
@Composable
fun PercentText(value: Int, color: androidx.compose.ui.graphics.Color) {
    Text(
        text = "$value%",
        fontFamily = MoneyFont,
        fontWeight = FontWeight.Bold,
        fontSize = 22.sp,
        color = color,
        textAlign = TextAlign.End,
    )
}
