package tech.tubsamy.kasku.ui.home

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.DateRange
import androidx.compose.material.icons.outlined.Home
import androidx.compose.material.icons.outlined.ShoppingCart
import androidx.compose.material.icons.outlined.Star
import androidx.compose.material.icons.outlined.Warning
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
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
    onDashboard: () -> Unit,
    onHistory: () -> Unit,
    onConflicts: () -> Unit,
    onCategories: () -> Unit,
    onInvestments: () -> Unit,
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
            TextButton(onClick = onLogout) { Text("Keluar") }
        }

        Spacer(Modifier.height(16.dp))

        // Navigasi: grid kartu ikon (Dashboard · Riwayat · Kategori · Investasi · Konflik).
        NavGrid(
            conflictCount = conflictCount,
            onDashboard = onDashboard,
            onHistory = onHistory,
            onCategories = onCategories,
            onInvestments = onInvestments,
            onConflicts = onConflicts,
        )

        Spacer(Modifier.height(28.dp))

        // Hero: total saldo semua akun — anchor visual halaman (Instrument Serif).
        SectionLabel("Total Saldo")
        Spacer(Modifier.height(4.dp))
        MoneyText(accounts.sumOf { it.balance }, fontSize = 44)

        Spacer(Modifier.height(24.dp))
        Hairline()
        Spacer(Modifier.height(4.dp))

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
            Text("+", style = MaterialTheme.typography.headlineMedium)
        }
    }
}

/** Grid 3-kolom kartu ikon; Konflik memakai aksen clay bila ada konflik tertunda. */
@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun NavGrid(
    conflictCount: Int,
    onDashboard: () -> Unit,
    onHistory: () -> Unit,
    onCategories: () -> Unit,
    onInvestments: () -> Unit,
    onConflicts: () -> Unit,
) {
    FlowRow(
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
        maxItemsInEachRow = 3,
    ) {
        NavCard("Dashboard", Icons.Outlined.Home, onDashboard, Modifier.weight(1f))
        NavCard("Riwayat", Icons.Outlined.DateRange, onHistory, Modifier.weight(1f))
        NavCard("Kategori", Icons.Outlined.ShoppingCart, onCategories, Modifier.weight(1f))
        NavCard("Investasi", Icons.Outlined.Star, onInvestments, Modifier.weight(1f))
        NavCard(
            label = if (conflictCount > 0) "Konflik ($conflictCount)" else "Konflik",
            icon = Icons.Outlined.Warning,
            onClick = onConflicts,
            modifier = Modifier.weight(1f),
            accent = conflictCount > 0,
        )
        // Slot ke-6 kosong agar dua kartu baris kedua selebar baris pertama (1/3).
        Spacer(Modifier.weight(1f))
    }
}

/** Kartu navigasi: ikon + label, border hairline, latar kartu — flat & editorial. */
@Composable
private fun NavCard(
    label: String,
    icon: ImageVector,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    accent: Boolean = false,
) {
    val tint = if (accent) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onBackground
    Surface(
        onClick = onClick,
        modifier = modifier,
        shape = RoundedCornerShape(6.dp),
        color = MaterialTheme.colorScheme.surface,
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant),
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = 18.dp, horizontal = 8.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Icon(icon, contentDescription = null, tint = tint)
            Spacer(Modifier.height(8.dp))
            Text(
                label,
                style = MaterialTheme.typography.labelLarge,
                color = tint,
                textAlign = TextAlign.Center,
                maxLines = 1,
            )
        }
    }
}

