package tech.tubsamy.kasku.ui.investment

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.data.InvestmentTxItem
import tech.tubsamy.kasku.ui.components.Hairline
import tech.tubsamy.kasku.ui.components.MoneyText
import tech.tubsamy.kasku.ui.components.PrimaryButton
import tech.tubsamy.kasku.ui.components.SectionLabel
import tech.tubsamy.kasku.ui.formatIdr
import tech.tubsamy.kasku.ui.theme.KasKuTeal
import kotlin.math.abs
import kotlin.math.roundToLong

@Composable
fun InvestmentDetailScreen(
    vm: InvestmentDetailViewModel,
    onBack: () -> Unit,
    onEdit: (String) -> Unit,
) {
    val asset = vm.asset
    var confirmDelete by remember { mutableStateOf(false) }

    if (confirmDelete && asset != null) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text("Hapus instrumen?") },
            text = { Text("\"${asset.name}\" dan riwayatnya akan dihapus. Tindakan ini tersinkron ke server.") },
            confirmButton = {
                TextButton(onClick = {
                    confirmDelete = false
                    vm.delete(onBack)
                }) { Text("Hapus") }
            },
            dismissButton = { TextButton(onClick = { confirmDelete = false }) { Text("Batal") } },
        )
    }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .verticalScroll(rememberScrollState())
            .padding(24.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(asset?.name ?: "Rincian", style = MaterialTheme.typography.headlineSmall)
            TextButton(onClick = onBack) { Text("Kembali") }
        }

        if (asset == null) {
            Spacer(Modifier.height(16.dp))
            Text(
                if (vm.loading) "Memuat…" else "Instrumen tidak ditemukan.",
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            return@Column
        }

        val unitLabel = if (asset.assetType == "GOLD") "gram" else "unit"
        Spacer(Modifier.height(4.dp))
        Text(
            (asset.symbol?.takeIf { it.isNotBlank() }?.let { "$it · " } ?: "") +
                "${formatUnits(asset.units)} $unitLabel · modal @ ${formatIdr(asset.avgBuyPriceIdr)}",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        // Nilai sekarang (harga live) + untung/rugi.
        Spacer(Modifier.height(12.dp))
        val live = vm.price
        val currentValue = live?.let { (it.priceIdr * asset.units).roundToLong() }
        MoneyText(currentValue ?: asset.bookValueIdr, fontSize = 32)
        if (currentValue != null && asset.bookValueIdr > 0) {
            val gainLoss = currentValue - asset.bookValueIdr
            val pct = gainLoss.toDouble() / asset.bookValueIdr * 100
            Text(
                "${if (gainLoss >= 0) "Untung" else "Rugi"} " +
                    "${if (gainLoss >= 0) "+" else "−"}${formatIdr(abs(gainLoss))} " +
                    "(${if (pct >= 0) "+" else "−"}${"%.2f".format(abs(pct))}%)" +
                    if (!live.isFresh) " · dari cache" else "",
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.SemiBold,
                color = if (gainLoss >= 0) KasKuTeal else MaterialTheme.colorScheme.error,
            )
        }

        Spacer(Modifier.height(16.dp))
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            OutlinedButton(onClick = { onEdit(asset.id) }, modifier = Modifier.weight(1f)) { Text("Edit") }
            OutlinedButton(onClick = { confirmDelete = true }, modifier = Modifier.weight(1f)) { Text("Hapus") }
        }

        Spacer(Modifier.height(24.dp))
        Hairline()
        Spacer(Modifier.height(16.dp))

        // Form catat transaksi BUY/SELL.
        SectionLabel("Catat transaksi")
        Spacer(Modifier.height(8.dp))
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            TypeToggle("Beli", selected = vm.txType == "BUY", modifier = Modifier.weight(1f)) { vm.txType = "BUY" }
            TypeToggle("Jual", selected = vm.txType == "SELL", modifier = Modifier.weight(1f)) { vm.txType = "SELL" }
        }
        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = vm.quantity,
            onValueChange = { input ->
                val filtered = input.filter { it.isDigit() || it == '.' }
                vm.quantity = if (filtered.count { it == '.' } <= 1) filtered else vm.quantity
            },
            label = { Text("Jumlah $unitLabel") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = vm.totalValue,
            onValueChange = { vm.totalValue = it.filter { c -> c.isDigit() } },
            label = { Text("Nilai ${if (vm.txType == "BUY") "pembelian" else "penjualan"} (Rp)") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )
        if (vm.pricePerUnitPreview > 0) {
            Spacer(Modifier.height(6.dp))
            Text(
                "Harga/unit ≈ ${formatIdr(vm.pricePerUnitPreview)}",
                style = MaterialTheme.typography.bodySmall,
                color = KasKuTeal,
            )
        }
        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = vm.txDate,
            onValueChange = { vm.txDate = it },
            label = { Text("Tanggal (YYYY-MM-DD)") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = vm.notes,
            onValueChange = { vm.notes = it },
            label = { Text("Catatan (opsional)") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )

        if (vm.error != null) {
            Spacer(Modifier.height(12.dp))
            Text(vm.error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
        }

        Spacer(Modifier.height(16.dp))
        PrimaryButton(
            text = "Simpan transaksi",
            onClick = { vm.record() },
            enabled = vm.canRecord,
            loading = vm.saving,
        )

        Spacer(Modifier.height(28.dp))
        SectionLabel("Riwayat transaksi")
        Spacer(Modifier.height(8.dp))
        if (vm.history.isEmpty()) {
            Text(
                if (vm.loading) "Memuat…" else "Belum ada riwayat transaksi.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        } else {
            vm.history.forEach { entry ->
                HistoryRow(entry)
                Hairline()
            }
        }
    }
}

@Composable
private fun TypeToggle(label: String, selected: Boolean, modifier: Modifier = Modifier, onClick: () -> Unit) {
    if (selected) {
        Button(onClick = onClick, modifier = modifier) { Text(label) }
    } else {
        OutlinedButton(onClick = onClick, modifier = modifier) { Text(label) }
    }
}

@Composable
private fun HistoryRow(entry: InvestmentTxItem) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 10.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Column(Modifier.weight(1f)) {
            Text(
                if (entry.isBuy) "Beli" else "Jual",
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.SemiBold,
                color = if (entry.isBuy) KasKuTeal else MaterialTheme.colorScheme.error,
            )
            Text(
                "${formatUnits(entry.quantityChange)} @ ${formatIdr(entry.pricePerUnit.roundToLong())}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (entry.notes.isNotBlank()) {
                Text(entry.notes, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
        Spacer(Modifier.width(12.dp))
        Column(horizontalAlignment = androidx.compose.ui.Alignment.End) {
            Text(
                formatIdr(entry.totalValue.roundToLong()),
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.SemiBold,
            )
            Text(entry.transactionDate, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

/** Units: buang trailing zero (2.0 → "2", 1.5 → "1.5"). */
private fun formatUnits(units: Double): String =
    if (units == units.toLong().toDouble()) units.toLong().toString() else units.toString()
