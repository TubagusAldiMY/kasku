package tech.tubsamy.kasku.ui.debt

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.Edit
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.data.DebtItem
import tech.tubsamy.kasku.ui.components.Hairline
import tech.tubsamy.kasku.ui.formatIdr

@Composable
fun DebtsScreen(
    vm: DebtViewModel,
    onBack: () -> Unit,
    onAddDebt: () -> Unit,
    onEditDebt: (String) -> Unit,
    onOpenPayments: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val debts by vm.debts.collectAsState()
    var pendingDelete by remember { mutableStateOf<DebtItem?>(null) }

    // Sisa yang aktif per arah (piutang = orang berhutang ke kita, hutang = kita berhutang).
    val receivable = debts.filter { it.isReceivable && !it.isSettled }.sumOf { it.remainingAmount }
    val payable = debts.filter { !it.isReceivable && !it.isSettled }.sumOf { it.remainingAmount }

    Box(modifier = modifier.fillMaxSize()) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(24.dp),
        ) {
            Text("Hutang & Piutang", style = MaterialTheme.typography.headlineMedium)
            Spacer(Modifier.height(4.dp))
            Text(
                if (debts.isEmpty()) {
                    "Catat siapa berhutang ke kamu dan sebaliknya."
                } else {
                    "Piutang ${formatIdr(receivable)} · Hutang ${formatIdr(payable)}"
                },
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            if (vm.error != null) {
                Spacer(Modifier.height(8.dp))
                Text(vm.error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
            }

            Spacer(Modifier.height(16.dp))

            if (debts.isEmpty()) {
                Column(
                    Modifier.fillMaxSize(),
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Text("Belum ada catatan.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Spacer(Modifier.height(12.dp))
                    OutlinedButton(onClick = { vm.refresh() }) { Text("Segarkan") }
                }
            } else {
                LazyColumn(contentPadding = PaddingValues(bottom = 88.dp)) {
                    items(debts, key = { it.id }) { d ->
                        DebtRow(
                            debt = d,
                            onClick = { onOpenPayments(d.id) },
                            onEdit = { onEditDebt(d.id) },
                            onDelete = { pendingDelete = d },
                        )
                        Hairline()
                    }
                }
            }
        }

        ExtendedFloatingActionButton(
            onClick = onAddDebt,
            modifier = Modifier
                .align(Alignment.BottomEnd)
                .padding(24.dp),
        ) {
            Text("Tambah")
        }
    }

    pendingDelete?.let { d ->
        AlertDialog(
            onDismissRequest = { pendingDelete = null },
            title = { Text("Hapus catatan?") },
            text = { Text("Catatan ${if (d.isReceivable) "piutang" else "hutang"} ${d.personName} akan dihapus.") },
            confirmButton = {
                TextButton(onClick = {
                    vm.delete(d.id)
                    pendingDelete = null
                }) { Text("Hapus") }
            },
            dismissButton = {
                TextButton(onClick = { pendingDelete = null }) { Text("Batal") }
            },
        )
    }
}

@Composable
private fun DebtRow(
    debt: DebtItem,
    onClick: () -> Unit,
    onEdit: () -> Unit,
    onDelete: () -> Unit,
) {
    // Piutang (uang masuk) = primary; Hutang (uang keluar) = error. Lunas = redup.
    val accent = when {
        debt.isSettled -> MaterialTheme.colorScheme.onSurfaceVariant
        debt.isReceivable -> MaterialTheme.colorScheme.primary
        else -> MaterialTheme.colorScheme.error
    }
    Row(verticalAlignment = Alignment.CenterVertically) {
        Column(
            modifier = Modifier
                .weight(1f)
                .clickable(onClick = onClick)
                .padding(vertical = 8.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    debt.personName,
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.SemiBold,
                )
                Spacer(Modifier.width(8.dp))
                Text(
                    if (debt.isReceivable) "Piutang" else "Hutang",
                    style = MaterialTheme.typography.labelSmall,
                    color = accent,
                )
                if (debt.isSettled) {
                    Spacer(Modifier.width(8.dp))
                    Text("Lunas", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
            Spacer(Modifier.height(2.dp))
            Text(
                "Sisa ${formatIdr(debt.remainingAmount)} dari ${formatIdr(debt.totalAmount)}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            debt.dueDate?.let {
                Text("Jatuh tempo $it", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
        IconButton(onClick = onEdit) {
            Icon(Icons.Outlined.Edit, contentDescription = "Edit catatan", tint = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        IconButton(onClick = onDelete) {
            Icon(Icons.Outlined.Delete, contentDescription = "Hapus catatan", tint = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}
