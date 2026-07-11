package tech.tubsamy.kasku.ui.home

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.Logout
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.outlined.Warning
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.ui.components.Hairline
import tech.tubsamy.kasku.ui.components.LedgerRow
import tech.tubsamy.kasku.ui.components.MoneyText
import tech.tubsamy.kasku.ui.components.SectionLabel

@Composable
fun HomeScreen(
    vm: HomeViewModel,
    onLogout: () -> Unit,
    onAddTransaction: () -> Unit,
    onConflicts: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accounts by vm.accounts.collectAsState()
    val conflictCount by vm.conflictCount.collectAsState()

    Box(modifier = modifier.fillMaxSize()) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(24.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = "Akun Saya",
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold,
                )
                Row(verticalAlignment = Alignment.CenterVertically) {
                    // Konflik hanya muncul saat ada — badge coral, tap → layar konflik.
                    if (conflictCount > 0) {
                        TextButton(onClick = onConflicts) {
                            Icon(
                                Icons.Outlined.Warning,
                                contentDescription = "Konflik",
                                tint = MaterialTheme.colorScheme.error,
                                modifier = Modifier.size(18.dp),
                            )
                            Spacer(Modifier.width(4.dp))
                            Text("$conflictCount", color = MaterialTheme.colorScheme.error)
                        }
                    }
                    IconButton(onClick = onLogout) {
                        Icon(
                            Icons.AutoMirrored.Outlined.Logout,
                            contentDescription = "Keluar",
                            tint = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            }

            Spacer(Modifier.height(20.dp))

            // Kartu saldo total — anchor visual halaman.
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
                elevation = CardDefaults.cardElevation(defaultElevation = 0.dp),
            ) {
                Column(Modifier.padding(20.dp)) {
                    SectionLabel("Total Saldo")
                    Spacer(Modifier.height(6.dp))
                    MoneyText(accounts.sumOf { it.balance }, fontSize = 44)
                }
            }

            Spacer(Modifier.height(24.dp))
            SectionLabel("Rekening")
            Spacer(Modifier.height(8.dp))

            if (accounts.isEmpty()) {
                Column(
                    Modifier.fillMaxSize(),
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Text(
                        "Belum ada akun.",
                        color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.6f),
                    )
                    Spacer(Modifier.height(12.dp))
                    OutlinedButton(onClick = { vm.refresh() }) { Text("Segarkan") }
                }
            } else {
                LazyColumn {
                    items(accounts) { account ->
                        LedgerRow(
                            label = account.name,
                            amount = account.balance,
                            sub = account.accountType,
                            // Saldo nol diredam agar akun aktif menonjol.
                            color = if (account.balance == 0L) {
                                MaterialTheme.colorScheme.onSurfaceVariant
                            } else {
                                MaterialTheme.colorScheme.onBackground
                            },
                        )
                        Hairline()
                    }
                }
            }
        }
        FloatingActionButton(
            onClick = onAddTransaction,
            modifier = Modifier
                .align(Alignment.BottomEnd)
                .padding(24.dp),
        ) {
            Icon(Icons.Filled.Add, contentDescription = "Tambah transaksi")
        }
    }
}
