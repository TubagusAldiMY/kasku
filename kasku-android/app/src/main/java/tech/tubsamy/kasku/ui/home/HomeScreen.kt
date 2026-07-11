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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.HorizontalDivider
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
import tech.tubsamy.kasku.data.AccountItem
import tech.tubsamy.kasku.ui.formatIdr

@Composable
fun HomeScreen(
    vm: HomeViewModel,
    onLogout: () -> Unit,
    onAddTransaction: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accounts by vm.accounts.collectAsState()

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
                    AccountRow(account)
                    HorizontalDivider()
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

@Composable
private fun AccountRow(account: AccountItem) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 14.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column {
            Text(account.name, style = MaterialTheme.typography.bodyLarge, fontWeight = FontWeight.Medium)
            Text(
                account.accountType,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.5f),
            )
        }
        Text(formatIdr(account.balance), style = MaterialTheme.typography.bodyLarge, fontWeight = FontWeight.SemiBold)
    }
}
