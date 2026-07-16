package tech.tubsamy.kasku.ui.debt

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.ui.components.PrimaryButton
import tech.tubsamy.kasku.ui.components.SectionLabel

@Composable
fun AddDebtScreen(
    vm: AddDebtViewModel,
    onSaved: () -> Unit,
    onCancel: () -> Unit,
) {
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
                if (vm.isEdit) "Edit catatan" else "Catat hutang/piutang",
                style = MaterialTheme.typography.headlineSmall,
            )
            TextButton(onClick = onCancel) { Text("Batal") }
        }

        Spacer(Modifier.height(20.dp))

        // Arah — hutang (kita berhutang) vs piutang (orang berhutang ke kita).
        // Tidak bisa diubah saat edit (backend hanya update person/tanggal/catatan).
        SectionLabel("Jenis")
        Spacer(Modifier.height(4.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            FilterChip(
                selected = vm.direction == "PAYABLE",
                onClick = { if (!vm.isEdit) vm.direction = "PAYABLE" },
                enabled = !vm.isEdit,
                label = { Text("Hutang saya") },
            )
            FilterChip(
                selected = vm.direction == "RECEIVABLE",
                onClick = { if (!vm.isEdit) vm.direction = "RECEIVABLE" },
                enabled = !vm.isEdit,
                label = { Text("Piutang") },
            )
        }

        Spacer(Modifier.height(16.dp))

        OutlinedTextField(
            value = vm.personName,
            onValueChange = { vm.personName = it },
            label = { Text("Nama orang") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(16.dp))

        OutlinedTextField(
            value = vm.amount,
            onValueChange = { vm.amount = it.filter { c -> c.isDigit() } },
            label = { Text("Jumlah (Rp)") },
            singleLine = true,
            enabled = !vm.isEdit, // nominal awal tidak bisa diubah setelah dibuat
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(16.dp))

        // ponytail: input tanggal manual YYYY-MM-DD; ganti ke DatePicker bila UX menuntut.
        OutlinedTextField(
            value = vm.dueDate,
            onValueChange = { vm.dueDate = it.filter { c -> c.isDigit() || c == '-' } },
            label = { Text("Jatuh tempo (YYYY-MM-DD, opsional)") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(16.dp))

        OutlinedTextField(
            value = vm.notes,
            onValueChange = { vm.notes = it },
            label = { Text("Catatan (opsional)") },
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
