package tech.tubsamy.kasku.ui.conflicts

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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.horizontalScroll
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.data.ConflictItem

@Composable
fun ConflictsScreen(
    vm: ConflictsViewModel,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val items by vm.items.collectAsState()

    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(24.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text("Konflik sync", style = MaterialTheme.typography.headlineMedium)
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
                    "Tidak ada konflik.",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        } else {
            LazyColumn(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                items(items, key = { it.id }) { conflict ->
                    ConflictCard(
                        conflict = conflict,
                        onKeepLocal = { vm.keepLocal(conflict) },
                        onAcceptServer = { vm.acceptServer(conflict) },
                    )
                }
            }
        }
    }
}

/** Label resource ID → Bahasa Indonesia. */
private fun resourceLabel(resource: String): String = when (resource) {
    "accounts" -> "Akun"
    "transactions" -> "Transaksi"
    "investments" -> "Investasi"
    else -> resource
}

@Composable
private fun ConflictCard(
    conflict: ConflictItem,
    onKeepLocal: () -> Unit,
    onAcceptServer: () -> Unit,
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp), // flat: warm card, bukan tonal lavender
    ) {
        Column(Modifier.padding(16.dp)) {
            Text(
                "${resourceLabel(conflict.resource)} · ${conflict.entityId}",
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                conflict.detectedAt,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            Spacer(Modifier.height(12.dp))

            ValueBlock("Versi saya", conflict.localValue)
            Spacer(Modifier.height(8.dp))
            ValueBlock("Versi server", conflict.serverValue)

            Spacer(Modifier.height(16.dp))

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = onKeepLocal, modifier = Modifier.weight(1f)) { Text("Pakai versi saya") }
                OutlinedButton(onClick = onAcceptServer, modifier = Modifier.weight(1f)) { Text("Terima server") }
            }
        }
    }
}

@Composable
private fun ValueBlock(label: String, rawJson: String) {
    Text(
        label,
        style = MaterialTheme.typography.labelMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
    // ponytail: tampilkan raw JSON apa adanya (scroll horizontal) — cukup untuk review, tak parse field.
    Text(
        rawJson,
        style = MaterialTheme.typography.bodySmall,
        modifier = Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState()),
        maxLines = 3,
    )
}
