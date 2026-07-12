package tech.tubsamy.kasku.ui.dashboard

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.VerticalDivider
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.data.BudgetItem
import tech.tubsamy.kasku.data.CategorySlice
import tech.tubsamy.kasku.data.MonthlyPoint
import tech.tubsamy.kasku.data.TransactionItem
import tech.tubsamy.kasku.ui.components.Hairline
import tech.tubsamy.kasku.ui.components.MoneyText
import tech.tubsamy.kasku.ui.components.SectionLabel
import tech.tubsamy.kasku.ui.formatIdr
import tech.tubsamy.kasku.ui.theme.KasKuGold

@Composable
fun DashboardScreen(
    vm: DashboardViewModel,
    onBack: () -> Unit,
    onSeeAllHistory: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val summary by vm.summary.collectAsState()
    val charts by vm.charts.collectAsState()
    val recent by vm.recent.collectAsState()
    val budgets by vm.budgets.collectAsState()

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(24.dp),
    ) {
        // Hero editorial: eyebrow berbulan + angka serif besar + delta arus kas (desain ReDesign/).
        SectionLabel("Kekayaan bersih · ${vm.monthLabel}")
        Spacer(Modifier.height(10.dp))
        MoneyText(summary.netWorth, fontSize = 46)
        CashFlowDelta(charts.trend)

        Spacer(Modifier.height(28.dp))

        // Grid statistik 2×2 dengan pemisah hairline (signature desain).
        Hairline()
        Row(Modifier.height(IntrinsicSize.Min)) {
            StatCell("Pemasukan", summary.monthIncome, MaterialTheme.colorScheme.primary, Modifier.weight(1f))
            VerticalDivider(color = MaterialTheme.colorScheme.outlineVariant)
            StatCell(
                "Pengeluaran", summary.monthExpense, MaterialTheme.colorScheme.error,
                Modifier.weight(1f).padding(start = 20.dp),
            )
        }
        Hairline()
        Row(Modifier.height(IntrinsicSize.Min)) {
            Column(Modifier.weight(1f).padding(vertical = 16.dp)) {
                SectionLabel("Menabung")
                Spacer(Modifier.height(6.dp))
                Text(
                    "${summary.savingsRate}%",
                    style = MaterialTheme.typography.headlineSmall,
                    color = MaterialTheme.colorScheme.primary,
                )
            }
            VerticalDivider(color = MaterialTheme.colorScheme.outlineVariant)
            StatCell(
                "Arus kas bersih", summary.monthIncome - summary.monthExpense,
                if (summary.monthIncome >= summary.monthExpense) {
                    MaterialTheme.colorScheme.onBackground
                } else {
                    MaterialTheme.colorScheme.error
                },
                Modifier.weight(1f).padding(start = 20.dp),
            )
        }
        Hairline()

        Spacer(Modifier.height(36.dp))
        SectionLabel("Tren bulanan")
        Spacer(Modifier.height(16.dp))
        MonthlyTrendChart(charts.trend)

        if (recent.isNotEmpty()) {
            Spacer(Modifier.height(36.dp))
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                SectionLabel("Aktivitas terakhir")
                TextButton(onClick = onSeeAllHistory) { Text("Lihat semua →") }
            }
            Spacer(Modifier.height(4.dp))
            recent.forEach { tx ->
                RecentRow(tx)
                Hairline()
            }
        }

        if (budgets.isNotEmpty()) {
            Spacer(Modifier.height(36.dp))
            SectionLabel("Anggaran bulan ini")
            Spacer(Modifier.height(4.dp))
            budgets.forEach { BudgetRow(it) }
        }

        Spacer(Modifier.height(36.dp))
        SectionLabel("Pengeluaran per kategori")
        Spacer(Modifier.height(16.dp))
        CategoryDonut(charts.expenseByCategory)

        if (summary.insights.isNotEmpty()) {
            Spacer(Modifier.height(36.dp))
            SectionLabel("Insight")
            Spacer(Modifier.height(4.dp))
            summary.insights.forEachIndexed { i, insight ->
                if (i > 0) Hairline()
                Text(
                    insight,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.padding(vertical = 14.dp),
                )
            }
        }
    }
}

/** Sel grid statistik: eyebrow + angka serif. */
@Composable
private fun StatCell(label: String, amount: Long, color: Color, modifier: Modifier = Modifier) {
    Column(modifier.padding(vertical = 16.dp)) {
        SectionLabel(label)
        Spacer(Modifier.height(6.dp))
        MoneyText(amount, fontSize = 24, color = color)
    }
}

/**
 * Delta arus kas vs bulan lalu ("Arus kas naik +Rp X dari bulan lalu") — dihitung dari
 * dua titik trend terakhir. Disembunyikan bila belum ada data dua bulan.
 */
@Composable
private fun CashFlowDelta(trend: List<MonthlyPoint>) {
    if (trend.size < 2) return
    val (prev, cur) = trend.takeLast(2)
    if (prev.income + prev.expense == 0L && cur.income + cur.expense == 0L) return
    val delta = (cur.income - cur.expense) - (prev.income - prev.expense)
    val up = delta >= 0
    val accent = if (up) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.error
    Spacer(Modifier.height(10.dp))
    Text(
        buildAnnotatedString {
            append("Arus kas ${if (up) "naik" else "turun"} ")
            withStyle(SpanStyle(color = accent, fontWeight = FontWeight.SemiBold)) {
                append((if (up) "+" else "−") + formatIdr(kotlin.math.abs(delta)))
            }
            append(" dari bulan lalu.")
        },
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
}

/** Baris aktivitas ringkas: tanggal · keterangan · nominal tabular. */
@Composable
private fun RecentRow(tx: TransactionItem) {
    val isIncome = tx.type == "INCOME"
    Row(
        Modifier.fillMaxWidth().padding(vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            tx.date.takeLast(5).split("-").reversed().joinToString("/"),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.width(44.dp),
        )
        Text(
            tx.notes?.takeIf { it.isNotBlank() } ?: tx.categoryName,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.Medium,
            modifier = Modifier.weight(1f).padding(end = 8.dp),
            maxLines = 1,
        )
        Text(
            (if (isIncome) "+" else "−") + formatIdr(tx.amountIdr).removePrefix("Rp "),
            style = MaterialTheme.typography.bodyMedium.copy(
                fontFeatureSettings = "tnum",
                fontWeight = if (isIncome) FontWeight.SemiBold else FontWeight.Normal,
            ),
            color = if (isIncome) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onBackground,
        )
    }
}

/**
 * Baris anggaran (desain ReDesign/): nama + persen berwarna, bar tipis, caption terpakai/limit.
 * Warna: lewat limit = clay, ≥ ambang alert = gold, sisanya teal.
 */
@Composable
private fun BudgetRow(b: BudgetItem) {
    val accent = when {
        b.isOverBudget -> MaterialTheme.colorScheme.error
        b.alertThreshold in 1..b.progressPercent -> KasKuGold
        else -> MaterialTheme.colorScheme.primary
    }
    Column(Modifier.fillMaxWidth().padding(vertical = 12.dp)) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(b.name, style = MaterialTheme.typography.bodyLarge, fontWeight = FontWeight.Medium)
            Text(
                "${b.progressPercent}%",
                style = MaterialTheme.typography.titleSmall,
                color = accent,
            )
        }
        Spacer(Modifier.height(8.dp))
        // Bar progres editorial 3dp — track hairline, isi warna status.
        Box(
            Modifier
                .fillMaxWidth()
                .height(3.dp)
                .background(MaterialTheme.colorScheme.outlineVariant),
        ) {
            Box(
                Modifier
                    .fillMaxWidth((b.progressPercent / 100f).coerceIn(0f, 1f))
                    .height(3.dp)
                    .background(accent),
            )
        }
        Spacer(Modifier.height(8.dp))
        Text(
            "${formatIdr(b.spentIdr)} terpakai · limit ${formatIdr(b.limitIdr)} · " +
                if (b.isOverBudget) "lewat ${formatIdr(-b.remainingIdr)}" else "sisa ${formatIdr(b.remainingIdr)}",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

/** Palet irisan donat — tinta editorial redup di atas paper (deterministik, di-index modulo). */
private val sliceColors = listOf(
    Color(0xFF1A5F66), Color(0xFFA4502F), Color(0xFF8A6A1F), Color(0xFF2F5583),
    Color(0xFF12312E), Color(0xFF5B7B4F), Color(0xFF7A4A66), Color(0xFF4F7B77),
)

/**
 * Tren income vs expense per bulan — grouped bar Canvas native (ponytail: ganti Vico
 * yang API-nya rewel antar-versi; Canvas ~30 baris, nol dependensi, nol risiko versi).
 * Data lokal dari Room. Kosong → placeholder. Flat + gridline editorial.
 */
@Composable
private fun MonthlyTrendChart(trend: List<MonthlyPoint>) {
    if (trend.all { it.income == 0L && it.expense == 0L }) {
        EmptyChartHint("Belum ada transaksi untuk ditampilkan.")
        return
    }
    val maxVal = trend.maxOf { maxOf(it.income, it.expense) }.coerceAtLeast(1L)
    val incomeColor = MaterialTheme.colorScheme.primary
    val expenseColor = MaterialTheme.colorScheme.error
    val gridColor = MaterialTheme.colorScheme.outlineVariant
    Column(Modifier.fillMaxWidth()) {
        androidx.compose.foundation.Canvas(
            modifier = Modifier.fillMaxWidth().height(180.dp),
        ) {
            // Gridline horizontal tipis di 1/3 & 2/3 + baseline (gaya grafik desain).
            listOf(size.height / 3f, size.height * 2f / 3f).forEach { y ->
                drawLine(gridColor, Offset(0f, y), Offset(size.width, y), strokeWidth = 1f)
            }
            drawLine(gridColor, Offset(0f, size.height), Offset(size.width, size.height), strokeWidth = 2f)
            val groupW = size.width / trend.size
            val barW = groupW * 0.3f
            val gap = groupW * 0.08f
            val h = size.height
            trend.forEachIndexed { i, p ->
                val cx = groupW * i + groupW / 2f
                val incH = (p.income.toFloat() / maxVal) * h
                val expH = (p.expense.toFloat() / maxVal) * h
                drawRect(
                    color = incomeColor,
                    topLeft = Offset(cx - barW - gap / 2f, h - incH),
                    size = Size(barW, incH),
                )
                drawRect(
                    color = expenseColor,
                    topLeft = Offset(cx + gap / 2f, h - expH),
                    size = Size(barW, expH),
                )
            }
        }
        Spacer(Modifier.height(6.dp))
        // Label bulan ("YYYY-MM" → "MM") sejajar tiap grup.
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceAround) {
            trend.forEach { p ->
                Text(
                    p.month.substring(5, 7),
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
        Spacer(Modifier.height(10.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(16.dp)) {
            LegendDot(incomeColor, "Pemasukan")
            LegendDot(expenseColor, "Pengeluaran")
        }
    }
}

/**
 * Donat pengeluaran per kategori — Compose Canvas native (ponytail: pie Vico masih
 * eksperimental; Canvas arc ~30 baris tanpa risiko API unstable). Flat di atas paper.
 */
@Composable
private fun CategoryDonut(slices: List<CategorySlice>) {
    val total = slices.sumOf { it.total }
    if (total <= 0L) {
        EmptyChartHint("Belum ada pengeluaran bulan ini.")
        return
    }
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        androidx.compose.foundation.Canvas(
            modifier = Modifier.size(140.dp).aspectRatio(1f),
        ) {
            var startAngle = -90f
            val strokeWidth = size.minDimension * 0.22f
            val arcSize = Size(size.minDimension - strokeWidth, size.minDimension - strokeWidth)
            val topLeft = androidx.compose.ui.geometry.Offset(strokeWidth / 2f, strokeWidth / 2f)
            slices.forEachIndexed { i, slice ->
                val sweep = (slice.total.toFloat() / total.toFloat()) * 360f
                drawArc(
                    color = sliceColors[i % sliceColors.size],
                    startAngle = startAngle,
                    sweepAngle = sweep,
                    useCenter = false,
                    topLeft = topLeft,
                    size = arcSize,
                    style = Stroke(width = strokeWidth),
                )
                startAngle += sweep
            }
        }
        Spacer(Modifier.size(16.dp))
        Column {
            Text(
                "Total ${formatIdr(total)}",
                style = MaterialTheme.typography.titleSmall,
                modifier = Modifier.padding(bottom = 6.dp),
            )
            slices.forEachIndexed { i, slice ->
                val pct = (slice.total * 100 / total).toInt()
                Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(vertical = 2.dp)) {
                    Box(Modifier.size(10.dp).clip(CircleShape).background(sliceColors[i % sliceColors.size]))
                    Spacer(Modifier.size(8.dp))
                    Text(
                        "${slice.name} · $pct% · ${formatIdr(slice.total)}",
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
        }
    }
}

@Composable
private fun LegendDot(color: Color, label: String) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Box(Modifier.size(10.dp).clip(CircleShape).background(color))
        Spacer(Modifier.size(6.dp))
        Text(label, style = MaterialTheme.typography.bodySmall)
    }
}

@Composable
private fun EmptyChartHint(text: String) {
    Column(Modifier.fillMaxWidth()) {
        Hairline()
        Text(
            text,
            modifier = Modifier.padding(vertical = 20.dp),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Hairline()
    }
}
