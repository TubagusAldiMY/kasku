package tech.tubsamy.kasku.ui.investment

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
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
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.ui.components.PrimaryButton
import tech.tubsamy.kasku.ui.components.SectionLabel

// (nilai server, label UI). Single source of truth untuk dropdown jenis aset.
private val ASSET_TYPES = listOf(
    "CRYPTO" to "Kripto",
    "GOLD" to "Emas",
    "STOCK" to "Saham",
    "MUTUAL_FUND" to "Reksa Dana",
)

@Composable
fun AddInvestmentScreen(
    vm: AddInvestmentViewModel,
    onSaved: () -> Unit,
    onCancel: () -> Unit,
) {
    var typeMenuOpen by remember { mutableStateOf(false) }
    val selectedTypeLabel = ASSET_TYPES.firstOrNull { it.first == vm.assetType }?.second ?: vm.assetType

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
            Text(
                if (vm.isEdit) "Edit instrumen" else "Tambah investasi",
                style = MaterialTheme.typography.headlineSmall,
            )
            TextButton(onClick = onCancel) { Text("Batal") }
        }

        Spacer(Modifier.height(20.dp))

        OutlinedTextField(
            value = vm.name,
            onValueChange = { vm.name = it },
            label = { Text("Nama aset") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(16.dp))

        // Jenis aset (dropdown)
        SectionLabel("Jenis aset")
        Spacer(Modifier.height(4.dp))
        Box {
            OutlinedButton(onClick = { typeMenuOpen = true }, modifier = Modifier.fillMaxWidth()) {
                Text(selectedTypeLabel)
            }
            DropdownMenu(expanded = typeMenuOpen, onDismissRequest = { typeMenuOpen = false }) {
                ASSET_TYPES.forEach { (value, label) ->
                    DropdownMenuItem(
                        text = { Text(label) },
                        onClick = {
                            vm.assetType = value
                            typeMenuOpen = false
                        },
                    )
                }
            }
        }

        Spacer(Modifier.height(16.dp))

        OutlinedTextField(
            value = vm.symbol,
            onValueChange = { vm.symbol = it },
            label = { Text("Simbol (opsional, mis. BTC)") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )

        // Pembelian awal hanya saat tambah — di mode edit units & avg dikelola lewat
        // pencatatan transaksi BUY/SELL (layar Rincian), sama seperti web.
        if (!vm.isEdit) {
            Spacer(Modifier.height(16.dp))

            OutlinedTextField(
                value = vm.units,
                // Izinkan digit + satu titik desimal.
                onValueChange = { input ->
                    val filtered = input.filter { it.isDigit() || it == '.' }
                    vm.units = if (filtered.count { it == '.' } <= 1) filtered else vm.units
                },
                label = { Text("Jumlah unit") },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                modifier = Modifier.fillMaxWidth(),
            )

            Spacer(Modifier.height(16.dp))

            OutlinedTextField(
                value = vm.avgBuyPrice,
                onValueChange = { vm.avgBuyPrice = it.filter { c -> c.isDigit() } },
                label = { Text("Harga beli rata-rata / unit (Rp)") },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                modifier = Modifier.fillMaxWidth(),
            )
        }

        if (vm.error != null) {
            Spacer(Modifier.height(12.dp))
            Text(vm.error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
        }

        Spacer(Modifier.height(24.dp))

        PrimaryButton(
            text = "Simpan",
            onClick = { vm.save(onSaved) },
            enabled = vm.canSave,
            loading = vm.saving,
        )
    }
}
