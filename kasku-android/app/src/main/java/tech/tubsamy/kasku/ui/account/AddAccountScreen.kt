package tech.tubsamy.kasku.ui.account

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

// (nilai server, label UI). Single source of truth untuk dropdown jenis akun.
private val ACCOUNT_TYPES = listOf(
    "BANK" to "Bank",
    "EWALLET" to "E-Wallet",
    "CASH" to "Tunai",
)

/** Label ramah-pengguna untuk jenis akun (dipakai juga di daftar akun). */
fun accountTypeLabel(type: String): String =
    ACCOUNT_TYPES.firstOrNull { it.first == type }?.second ?: type

@Composable
fun AddAccountScreen(
    vm: AddAccountViewModel,
    onSaved: () -> Unit,
    onCancel: () -> Unit,
) {
    var typeMenuOpen by remember { mutableStateOf(false) }
    val selectedTypeLabel = accountTypeLabel(vm.accountType)

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
                if (vm.isEdit) "Edit akun" else "Buat akun",
                style = MaterialTheme.typography.headlineSmall,
            )
            TextButton(onClick = onCancel) { Text("Batal") }
        }

        Spacer(Modifier.height(20.dp))

        OutlinedTextField(
            value = vm.name,
            onValueChange = { vm.name = it },
            label = { Text("Nama akun") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(16.dp))

        // Jenis akun (dropdown)
        SectionLabel("Jenis akun")
        Spacer(Modifier.height(4.dp))
        Box {
            OutlinedButton(onClick = { typeMenuOpen = true }, modifier = Modifier.fillMaxWidth()) {
                Text(selectedTypeLabel)
            }
            DropdownMenu(expanded = typeMenuOpen, onDismissRequest = { typeMenuOpen = false }) {
                ACCOUNT_TYPES.forEach { (value, label) ->
                    DropdownMenuItem(
                        text = { Text(label) },
                        onClick = {
                            vm.accountType = value
                            typeMenuOpen = false
                        },
                    )
                }
            }
        }

        Spacer(Modifier.height(16.dp))

        OutlinedTextField(
            value = vm.balance,
            onValueChange = { vm.balance = it.filter { c -> c.isDigit() } },
            label = { Text("Saldo awal (Rp)") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )

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
