package tech.tubsamy.kasku.ui.transaction

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
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
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
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
import tech.tubsamy.kasku.data.TransactionItem
import tech.tubsamy.kasku.ui.components.Hairline
import tech.tubsamy.kasku.ui.components.MoneyText
import tech.tubsamy.kasku.ui.formatIdr

@Composable
fun TransactionHistoryScreen(
    vm: TransactionHistoryViewModel,
    onBack: () -> Unit,
    onEdit: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val items by vm.items.collectAsState()
    // Item yang menunggu konfirmasi hapus (null = tak ada dialog).
    var pendingDelete by remember { mutableStateOf<TransactionItem?>(null) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(24.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text("Riwayat", style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Bold)
            TextButton(onClick = onBack) { Text("Kembali") }
        }

        Spacer(Modifier.height(16.dp))

        if (items.isEmpty()) {
            Column(
                Modifier.fillMaxSize(),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(
                    "Belum ada transaksi.",
                    color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.6f),
                )
            }
        } else {
            LazyColumn {
                items(items, key = { it.id }) { tx ->
                    TransactionRow(
                        tx = tx,
                        onEdit = { onEdit(tx.id) },
                        onDelete = { pendingDelete = tx },
                    )
                    Hairline()
                }
            }
        }
    }

    pendingDelete?.let { tx ->
        AlertDialog(
            onDismissRequest = { pendingDelete = null },
            title = { Text("Hapus transaksi?") },
            text = {
                val label = (if (tx.type == "INCOME") "" else "−") + formatIdr(tx.amountIdr)
                Text("$label · ${tx.categoryName} (${tx.date}) akan dihapus.")
            },
            confirmButton = {
                TextButton(onClick = {
                    vm.delete(tx.id)
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
private fun TransactionRow(
    tx: TransactionItem,
    onEdit: () -> Unit,
    onDelete: () -> Unit,
) {
    val isIncome = tx.type == "INCOME"
    // income = primary (teal), expense = error (clay). Tanpa hardcode hex.
    val amountColor = if (isIncome) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.error

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onEdit) // tap baris = edit
            .padding(vertical = 14.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(tx.categoryName, style = MaterialTheme.typography.bodyLarge, fontWeight = FontWeight.Medium)
            Text(
                text = tx.notes?.takeIf { it.isNotBlank() }?.let { "${tx.date} · $it" } ?: tx.date,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.5f),
            )
        }
        MoneyText(if (isIncome) tx.amountIdr else -tx.amountIdr, fontSize = 20, color = amountColor, signed = true)
        IconButton(onClick = onDelete) {
            Icon(
                Icons.Outlined.Delete,
                contentDescription = "Hapus transaksi",
                tint = MaterialTheme.colorScheme.onSurfaceVariant, // muted; dialog konfirmasi jaga keamanan
            )
        }
    }
}
