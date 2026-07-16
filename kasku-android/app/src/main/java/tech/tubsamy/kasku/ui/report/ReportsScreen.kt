package tech.tubsamy.kasku.ui.report

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.data.remote.dto.CategoryBreakdownDto
import tech.tubsamy.kasku.data.remote.dto.MonthlyPointDto
import tech.tubsamy.kasku.ui.formatIdr

@Composable
fun ReportsScreen(
    vm: ReportsViewModel,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val report = vm.report

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(24.dp),
    ) {
        Text("Laporan", style = MaterialTheme.typography.headlineMedium)
        Spacer(Modifier.height(4.dp))
        Text(
            if (report != null && report.periodFrom.isNotBlank()) {
                "${report.periodFrom} — ${report.periodTo}"
            } else {
                "Ringkasan pemasukan & pengeluaran."
            },
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        if (vm.error != null) {
            Spacer(Modifier.height(16.dp))
            Text(vm.error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
            Spacer(Modifier.height(12.dp))
            OutlinedButton(onClick = { vm.load() }) { Text("Coba lagi") }
            return@Column
        }

        if (vm.loading && report == null) {
            Spacer(Modifier.height(32.dp))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.Center) {
                CircularProgressIndicator()
            }
            return@Column
        }

        if (report == null) return@Column

        Spacer(Modifier.height(20.dp))
        StatRow("Pemasukan", report.totalIncome, MaterialTheme.colorScheme.primary)
        StatRow("Pengeluaran", report.totalExpense, MaterialTheme.colorScheme.error)
        StatRow("Bersih", report.netAmount, MaterialTheme.colorScheme.onSurface)

        Spacer(Modifier.height(24.dp))
        Text("Per kategori", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
        Spacer(Modifier.height(8.dp))
        if (report.categoryBreakdown.isEmpty()) {
            Text(
                "Belum ada transaksi pada periode ini.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        } else {
            report.categoryBreakdown.forEach { c ->
                CategoryRow(c)
                HorizontalDivider()
            }
        }

        Spacer(Modifier.height(24.dp))
        Text("Tren bulanan", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
        Spacer(Modifier.height(8.dp))
        if (report.monthlyTrend.isEmpty()) {
            Text(
                "Belum ada data tren.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        } else {
            report.monthlyTrend.forEach { m ->
                MonthlyRow(m)
                HorizontalDivider()
            }
        }
    }
}

@Composable
private fun StatRow(label: String, amount: Long, color: androidx.compose.ui.graphics.Color) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 8.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, style = MaterialTheme.typography.bodyLarge)
        Text(formatIdr(amount), style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold, color = color)
    }
}

@Composable
private fun CategoryRow(c: CategoryBreakdownDto) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 12.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(c.categoryName.ifBlank { "Tanpa kategori" }, style = MaterialTheme.typography.bodyLarge, fontWeight = FontWeight.Medium)
            Text(
                "${c.count} transaksi · ${"%.1f".format(c.percentage)}%",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        Text(formatIdr(c.totalAmount), style = MaterialTheme.typography.bodyLarge)
    }
}

@Composable
private fun MonthlyRow(m: MonthlyPointDto) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 12.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(m.month, style = MaterialTheme.typography.bodyLarge, fontWeight = FontWeight.Medium)
        Column(horizontalAlignment = Alignment.End) {
            Text("+${formatIdr(m.income)}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
            Text("-${formatIdr(m.expense)}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
        }
    }
}
