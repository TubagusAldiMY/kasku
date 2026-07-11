package tech.tubsamy.kasku.ui.transaction

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
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneOffset

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AddTransactionScreen(
    vm: AddTransactionViewModel,
    onSaved: () -> Unit,
    onCancel: () -> Unit,
) {
    val accounts by vm.accounts.collectAsState()
    val categoryOptions by vm.categoryOptions.collectAsState()
    var accountMenuOpen by remember { mutableStateOf(false) }
    var categoryMenuOpen by remember { mutableStateOf(false) }
    var datePickerOpen by remember { mutableStateOf(false) }

    val selectedAccountName = accounts.firstOrNull { it.id == vm.accountId }?.name
    val selectedCategoryName = categoryOptions.firstOrNull { it.id == vm.categoryId }?.name

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
            Text("Tambah Transaksi", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
            TextButton(onClick = onCancel) { Text("Batal") }
        }

        Spacer(Modifier.height(20.dp))

        // Tipe (mengubah type me-reset categoryId & mem-filter dropdown kategori)
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            FilterChip(selected = vm.type == "EXPENSE", onClick = { vm.type = "EXPENSE" }, label = { Text("Pengeluaran") })
            FilterChip(selected = vm.type == "INCOME", onClick = { vm.type = "INCOME" }, label = { Text("Pemasukan") })
        }

        Spacer(Modifier.height(16.dp))

        OutlinedTextField(
            value = vm.amount,
            onValueChange = { vm.amount = it.filter { c -> c.isDigit() } },
            label = { Text("Jumlah (Rp)") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(16.dp))

        // Akun (dropdown)
        Text("Akun", style = MaterialTheme.typography.labelMedium)
        Spacer(Modifier.height(4.dp))
        Box {
            OutlinedButton(onClick = { accountMenuOpen = true }, modifier = Modifier.fillMaxWidth()) {
                Text(selectedAccountName ?: "Pilih akun")
            }
            DropdownMenu(expanded = accountMenuOpen, onDismissRequest = { accountMenuOpen = false }) {
                accounts.forEach { acc ->
                    DropdownMenuItem(
                        text = { Text("${acc.name} (${acc.accountType})") },
                        onClick = {
                            vm.accountId = acc.id
                            accountMenuOpen = false
                        },
                    )
                }
            }
        }

        Spacer(Modifier.height(16.dp))

        // Kategori (dropdown, opsional — filter otomatis by tipe)
        Text("Kategori (opsional)", style = MaterialTheme.typography.labelMedium)
        Spacer(Modifier.height(4.dp))
        Box {
            OutlinedButton(onClick = { categoryMenuOpen = true }, modifier = Modifier.fillMaxWidth()) {
                Text(selectedCategoryName ?: "Tanpa kategori")
            }
            DropdownMenu(expanded = categoryMenuOpen, onDismissRequest = { categoryMenuOpen = false }) {
                DropdownMenuItem(
                    text = { Text("Tanpa kategori") },
                    onClick = {
                        vm.categoryId = null
                        categoryMenuOpen = false
                    },
                )
                categoryOptions.forEach { cat ->
                    DropdownMenuItem(
                        text = { Text(cat.name) },
                        onClick = {
                            vm.categoryId = cat.id
                            categoryMenuOpen = false
                        },
                    )
                }
            }
        }

        Spacer(Modifier.height(16.dp))

        // Tanggal (Material3 DatePicker, default hari ini)
        Text("Tanggal", style = MaterialTheme.typography.labelMedium)
        Spacer(Modifier.height(4.dp))
        OutlinedButton(onClick = { datePickerOpen = true }, modifier = Modifier.fillMaxWidth()) {
            Text(vm.date)
        }

        Spacer(Modifier.height(16.dp))

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

        Spacer(Modifier.height(24.dp))

        Button(
            onClick = { vm.save(onSaved) },
            enabled = vm.canSave,
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (vm.saving) {
                CircularProgressIndicator(
                    modifier = Modifier.height(18.dp),
                    strokeWidth = 2.dp,
                    color = MaterialTheme.colorScheme.onPrimary,
                )
            } else {
                Text("Simpan")
            }
        }
    }

    if (datePickerOpen) {
        // Seed dari vm.date (UTC midnight) supaya picker buka di tanggal terpilih.
        val initialMillis = runCatching {
            LocalDate.parse(vm.date).atStartOfDay(ZoneOffset.UTC).toInstant().toEpochMilli()
        }.getOrNull()
        val state = rememberDatePickerState(initialSelectedDateMillis = initialMillis)
        DatePickerDialog(
            onDismissRequest = { datePickerOpen = false },
            confirmButton = {
                TextButton(onClick = {
                    // millis (UTC) → "YYYY-MM-DD". DatePicker mengembalikan UTC midnight.
                    state.selectedDateMillis?.let { millis ->
                        vm.date = Instant.ofEpochMilli(millis).atZone(ZoneOffset.UTC).toLocalDate().toString()
                    }
                    datePickerOpen = false
                }) { Text("Pilih") }
            },
            dismissButton = {
                TextButton(onClick = { datePickerOpen = false }) { Text("Batal") }
            },
        ) {
            DatePicker(state = state)
        }
    }
}
