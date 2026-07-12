package tech.tubsamy.kasku.ui.budget

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
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.ui.components.PrimaryButton
import tech.tubsamy.kasku.ui.components.SectionLabel

@Composable
fun AddBudgetScreen(
    vm: AddBudgetViewModel,
    onSaved: () -> Unit,
    onCancel: () -> Unit,
) {
    var categoryMenuOpen by remember { mutableStateOf(false) }
    // Re-read opsi kategori tiap cache terisi (version naik setelah ensureLoaded sukses).
    val catVersion by vm.categories.version.collectAsState()
    val categoryOptions = remember(catVersion) { vm.categories.forType("EXPENSE") }
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
            Text(
                if (vm.isEdit) "Edit anggaran" else "Buat anggaran",
                style = MaterialTheme.typography.headlineSmall,
            )
            TextButton(onClick = onCancel) { Text("Batal") }
        }

        Spacer(Modifier.height(20.dp))

        OutlinedTextField(
            value = vm.name,
            onValueChange = { vm.name = it },
            label = { Text("Nama anggaran") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(16.dp))

        OutlinedTextField(
            value = vm.limit,
            onValueChange = { vm.limit = it.filter { c -> c.isDigit() } },
            label = { Text("Limit per bulan (Rp)") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(16.dp))

        // Kategori (dropdown, opsional — tanpa kategori = anggaran total pengeluaran)
        SectionLabel("Kategori (opsional)")
        Spacer(Modifier.height(4.dp))
        Box {
            OutlinedButton(onClick = { categoryMenuOpen = true }, modifier = Modifier.fillMaxWidth()) {
                Text(selectedCategoryName ?: "Semua pengeluaran")
            }
            DropdownMenu(expanded = categoryMenuOpen, onDismissRequest = { categoryMenuOpen = false }) {
                DropdownMenuItem(
                    text = { Text("Semua pengeluaran") },
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

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(Modifier.weight(1f).padding(end = 12.dp)) {
                Text("Jatah harian", style = MaterialTheme.typography.bodyLarge)
                Text(
                    "Bagi limit jadi jatah per hari; sisa/lebihnya dibawa ke hari berikutnya.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Switch(
                checked = vm.dailyLimitEnabled,
                onCheckedChange = { vm.dailyLimitEnabled = it },
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
