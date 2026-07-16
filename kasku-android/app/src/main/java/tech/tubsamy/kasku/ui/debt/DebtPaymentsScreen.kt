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
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.ui.components.Hairline
import tech.tubsamy.kasku.ui.components.PrimaryButton
import tech.tubsamy.kasku.ui.components.SectionLabel
import tech.tubsamy.kasku.ui.formatIdr

@Composable
fun DebtPaymentsScreen(
    vm: DebtPaymentsViewModel,
    onBack: () -> Unit,
) {
    val debt = vm.debt

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
                debt?.personName ?: "Rincian",
                style = MaterialTheme.typography.headlineSmall,
            )
            TextButton(onClick = onBack) { Text("Kembali") }
        }

        if (debt == null) {
            Spacer(Modifier.height(16.dp))
            Text(
                if (vm.loading) "Memuat…" else "Catatan tidak ditemukan.",
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            return@Column
        }

        Spacer(Modifier.height(4.dp))
        Text(
            (if (debt.isReceivable) "Piutang" else "Hutang") +
                if (debt.isSettled) " · Lunas" else "",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        Spacer(Modifier.height(12.dp))
        Text("Sisa ${formatIdr(debt.remainingAmount)}", style = MaterialTheme.typography.headlineMedium)
        Text(
            "Terbayar ${formatIdr(debt.paidAmount)} dari ${formatIdr(debt.totalAmount)}",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        Spacer(Modifier.height(24.dp))

        // Form catat pembayaran — sembunyikan bila sudah lunas.
        if (!debt.isSettled) {
            SectionLabel("Catat pembayaran")
            Spacer(Modifier.height(8.dp))
            OutlinedTextField(
                value = vm.amount,
                onValueChange = { vm.amount = it.filter { c -> c.isDigit() } },
                label = { Text("Jumlah (Rp)") },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
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
                text = "Simpan pembayaran",
                onClick = { vm.recordPayment() },
                enabled = vm.canRecord,
                loading = vm.saving,
            )
            Spacer(Modifier.height(24.dp))
        }

        SectionLabel("Riwayat pembayaran")
        Spacer(Modifier.height(8.dp))
        if (vm.payments.isEmpty()) {
            Text(
                if (vm.loading) "Memuat…" else "Belum ada pembayaran.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        } else {
            vm.payments.forEach { p ->
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 8.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Column(Modifier.weight(1f)) {
                        Text(p.paymentDate, style = MaterialTheme.typography.bodyMedium)
                        if (p.notes.isNotBlank()) {
                            Text(p.notes, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                    Text(
                        formatIdr(p.amount),
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.SemiBold,
                    )
                }
                Hairline()
            }
        }
    }
}
