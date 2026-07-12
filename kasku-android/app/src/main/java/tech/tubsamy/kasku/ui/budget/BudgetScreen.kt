package tech.tubsamy.kasku.ui.budget

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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Delete
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
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.data.BudgetItem
import tech.tubsamy.kasku.ui.components.BudgetProgress
import tech.tubsamy.kasku.ui.components.Hairline
import tech.tubsamy.kasku.ui.formatIdr

@Composable
fun BudgetScreen(
    vm: BudgetViewModel,
    onBack: () -> Unit,
    onAddBudget: () -> Unit,
    onEditBudget: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val budgets by vm.budgets.collectAsState()
    // Item yang menunggu konfirmasi hapus (null = tak ada dialog).
    var pendingDelete by remember { mutableStateOf<BudgetItem?>(null) }

    Box(modifier = modifier.fillMaxSize()) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(24.dp),
        ) {
            Text("Anggaran", style = MaterialTheme.typography.headlineMedium)
            Spacer(Modifier.height(4.dp))
            Text(
                if (budgets.isEmpty()) {
                    "Batasi pengeluaran per kategori."
                } else {
                    "Terpakai ${formatIdr(budgets.sumOf { it.spentIdr })} dari ${formatIdr(budgets.sumOf { it.limitIdr })}"
                },
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            if (vm.error != null) {
                Spacer(Modifier.height(8.dp))
                Text(vm.error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
            }

            Spacer(Modifier.height(16.dp))

            if (budgets.isEmpty()) {
                Column(
                    Modifier.fillMaxSize(),
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Text("Belum ada anggaran.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Spacer(Modifier.height(12.dp))
                    OutlinedButton(onClick = { vm.refresh() }) { Text("Segarkan") }
                }
            } else {
                LazyColumn(contentPadding = PaddingValues(bottom = 88.dp)) {
                    items(budgets, key = { it.id }) { b ->
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            BudgetProgress(
                                b,
                                modifier = Modifier
                                    .weight(1f)
                                    .clickable { onEditBudget(b.id) }, // tap = edit
                            )
                            IconButton(onClick = { pendingDelete = b }) {
                                Icon(
                                    Icons.Outlined.Delete,
                                    contentDescription = "Hapus anggaran",
                                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                        }
                        Hairline()
                    }
                }
            }
        }

        ExtendedFloatingActionButton(
            onClick = onAddBudget,
            modifier = Modifier
                .align(Alignment.BottomEnd)
                .padding(24.dp),
        ) {
            Text("Tambah")
        }
    }

    pendingDelete?.let { b ->
        AlertDialog(
            onDismissRequest = { pendingDelete = null },
            title = { Text("Hapus anggaran?") },
            text = { Text("${b.name} (limit ${formatIdr(b.limitIdr)}) akan dihapus.") },
            confirmButton = {
                TextButton(onClick = {
                    vm.delete(b.id)
                    pendingDelete = null
                }) { Text("Hapus") }
            },
            dismissButton = {
                TextButton(onClick = { pendingDelete = null }) { Text("Batal") }
            },
        )
    }
}
