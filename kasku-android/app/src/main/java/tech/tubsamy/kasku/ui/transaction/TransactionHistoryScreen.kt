package tech.tubsamy.kasku.ui.transaction

import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.FilterChip
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
import tech.tubsamy.kasku.ui.formatIdr

@Composable
fun TransactionHistoryScreen(
    vm: TransactionHistoryViewModel,
    onBack: () -> Unit,
    onEdit: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val items by vm.items.collectAsState()
    val typeFilter by vm.typeFilter.collectAsState()
    // Item yang menunggu konfirmasi hapus (null = tak ada dialog).
    var pendingDelete by remember { mutableStateOf<TransactionItem?>(null) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(24.dp),
    ) {
        Text("Riwayat transaksi", style = MaterialTheme.typography.headlineMedium)
        Spacer(Modifier.height(4.dp))
        Text(
            "${items.size} transaksi",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        Spacer(Modifier.height(16.dp))

        // Filter pills desain: Semua / Masuk / Keluar.
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            listOf(null to "Semua", "INCOME" to "Masuk", "EXPENSE" to "Keluar").forEach { (type, label) ->
                FilterChip(
                    selected = typeFilter == type,
                    onClick = { vm.typeFilter.value = type },
                    label = { Text(label) },
                    shape = RoundedCornerShape(999.dp),
                )
            }
        }

        Spacer(Modifier.height(12.dp))

        if (items.isEmpty()) {
            Column(
                Modifier.fillMaxSize(),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(
                    if (typeFilter == null) "Belum ada transaksi." else "Tidak ada transaksi untuk filter ini.",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
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

/** Chip kategori editorial: pill outline tipis, teks kecil muted. */
@Composable
private fun CategoryChip(name: String) {
    Text(
        name,
        style = MaterialTheme.typography.labelSmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier
            .border(1.dp, MaterialTheme.colorScheme.outline, RoundedCornerShape(999.dp))
            .padding(horizontal = 9.dp, vertical = 2.dp),
    )
}

@Composable
private fun TransactionRow(
    tx: TransactionItem,
    onEdit: () -> Unit,
    onDelete: () -> Unit,
) {
    val isIncome = tx.type == "INCOME"
    // Gaya tabel desain: income teal semibold (+), expense ink netral (−).
    val amountColor = if (isIncome) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onBackground
    // Angka rata: tabular figures supaya kolom nominal lurus antar-baris.
    val amountStyle = MaterialTheme.typography.bodyLarge.copy(
        fontFeatureSettings = "tnum",
        fontWeight = if (isIncome) FontWeight.SemiBold else FontWeight.Normal,
    )

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onEdit) // tap baris = edit
            .padding(vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        // Kolom tanggal ala tabel editorial ("2026-07-08" → "08/07").
        Text(
            tx.date.takeLast(5).split("-").reversed().joinToString("/"),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.width(44.dp),
        )
        Column(modifier = Modifier.weight(1f).padding(end = 8.dp)) {
            Text(
                tx.notes?.takeIf { it.isNotBlank() } ?: tx.categoryName,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium,
            )
            Spacer(Modifier.height(4.dp))
            CategoryChip(tx.categoryName)
        }
        Text(
            (if (isIncome) "+" else "−") + formatIdr(tx.amountIdr).removePrefix("Rp "),
            style = amountStyle,
            color = amountColor,
        )
        IconButton(onClick = onDelete) {
            Icon(
                Icons.Outlined.Delete,
                contentDescription = "Hapus transaksi",
                tint = MaterialTheme.colorScheme.onSurfaceVariant, // muted; dialog konfirmasi jaga keamanan
            )
        }
    }
}
