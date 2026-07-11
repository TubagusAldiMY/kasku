package tech.tubsamy.kasku.ui.dashboard

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.data.CategorySlice
import tech.tubsamy.kasku.data.MonthlyPoint
import tech.tubsamy.kasku.ui.components.Hairline
import tech.tubsamy.kasku.ui.components.LedgerRow
import tech.tubsamy.kasku.ui.components.MoneyText
import tech.tubsamy.kasku.ui.components.PercentText
import tech.tubsamy.kasku.ui.components.SectionLabel
import tech.tubsamy.kasku.ui.formatIdr

@Composable
fun DashboardScreen(
    vm: DashboardViewModel,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val summary by vm.summary.collectAsState()
    val charts by vm.charts.collectAsState()

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(24.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text("Dashboard", style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Bold)
            TextButton(onClick = onBack) { Text("Kembali") }
        }

        Spacer(Modifier.height(28.dp))

        // Hero: kekayaan bersih — angka serif besar (signature buku kas).
        SectionLabel("Kekayaan Bersih")
        Spacer(Modifier.height(6.dp))
        MoneyText(summary.netWorth, fontSize = 46)

        Spacer(Modifier.height(24.dp))
        Hairline()
        LedgerRow("Pemasukan bulan ini", summary.monthIncome, color = MaterialTheme.colorScheme.primary)
        Hairline()
        LedgerRow("Pengeluaran bulan ini", summary.monthExpense, color = MaterialTheme.colorScheme.error)
        Hairline()
        Row(
            modifier = Modifier.fillMaxWidth().padding(vertical = 14.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text("Tingkat Menabung", style = MaterialTheme.typography.bodyLarge)
            PercentText(summary.savingsRate, color = MaterialTheme.colorScheme.primary)
        }
        Hairline()

        Spacer(Modifier.height(28.dp))
        SectionLabel("Tren Bulanan")
        Spacer(Modifier.height(12.dp))
        MonthlyTrendChart(charts.trend)

        Spacer(Modifier.height(28.dp))
        SectionLabel("Pengeluaran per Kategori")
        Spacer(Modifier.height(12.dp))
        CategoryDonut(charts.expenseByCategory)

        if (summary.insights.isNotEmpty()) {
            Spacer(Modifier.height(28.dp))
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

/** Palet warna deterministik untuk irisan donat — palet brand "Buku Kas" (bukan warna generik). */
private val sliceColors = listOf(
    Color(0xFF1A5F66), Color(0xFFA4502F), Color(0xFF8A6A1F), Color(0xFF7FC7BD),
    Color(0xFF3F6B5E), Color(0xFFB98A3A), Color(0xFF6E4A3A), Color(0xFF5F6E6A),
)

/**
 * Tren income vs expense per bulan — grouped bar Canvas native (ponytail: ganti Vico
 * yang API-nya rewel antar-versi; Canvas ~30 baris, nol dependensi, nol risiko versi).
 * Data lokal dari Room. Kosong → placeholder.
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
    Card(
        Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp), // flat: hindari tonal lavender
    ) {
        Column(Modifier.padding(16.dp)) {
            androidx.compose.foundation.Canvas(
                modifier = Modifier.fillMaxWidth().height(180.dp),
            ) {
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
            Spacer(Modifier.height(4.dp))
            // Label bulan ("YYYY-MM" → "MM") sejajar tiap grup.
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceAround) {
                trend.forEach { p ->
                    Text(p.month.substring(5, 7), style = MaterialTheme.typography.labelSmall)
                }
            }
            Spacer(Modifier.height(8.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                LegendDot(incomeColor, "Pemasukan")
                LegendDot(expenseColor, "Pengeluaran")
            }
        }
    }
}

/**
 * Donat pengeluaran per kategori — Compose Canvas native (ponytail: pie Vico masih
 * eksperimental; Canvas arc ~30 baris tanpa risiko API unstable).
 */
@Composable
private fun CategoryDonut(slices: List<CategorySlice>) {
    val total = slices.sumOf { it.total }
    if (total <= 0L) {
        EmptyChartHint("Belum ada pengeluaran bulan ini.")
        return
    }
    Card(
        Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp), // flat: hindari tonal lavender
    ) {
        Row(Modifier.padding(16.dp), verticalAlignment = Alignment.CenterVertically) {
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
    Card(
        Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp), // flat: hindari tonal lavender
    ) {
        Text(
            text,
            modifier = Modifier.padding(24.dp),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
        )
    }
}

