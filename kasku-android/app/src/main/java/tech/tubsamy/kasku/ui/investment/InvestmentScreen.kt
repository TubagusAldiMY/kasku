package tech.tubsamy.kasku.ui.investment

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
import androidx.compose.material3.ExtendedFloatingActionButton
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
import tech.tubsamy.kasku.data.InvestmentItem
import tech.tubsamy.kasku.ui.components.Hairline
import tech.tubsamy.kasku.ui.components.MoneyText
import tech.tubsamy.kasku.ui.formatIdr

/** Label ramah untuk asset_type (fallback: nilai mentah). */
private fun assetTypeLabel(type: String): String = when (type) {
    "CRYPTO" -> "Kripto"
    "GOLD" -> "Emas"
    "STOCK" -> "Saham"
    "MUTUAL_FUND" -> "Reksa Dana"
    else -> type
}

@Composable
fun InvestmentScreen(
    vm: InvestmentViewModel,
    onBack: () -> Unit,
    onAddInvestment: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val investments by vm.investments.collectAsState()

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
                Text("Investasi", style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Bold)
                TextButton(onClick = onBack) { Text("Kembali") }
            }

            Spacer(Modifier.height(16.dp))

            if (investments.isEmpty()) {
                Column(
                    Modifier.fillMaxSize(),
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Text(
                        "Belum ada investasi.",
                        color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.6f),
                    )
                    Spacer(Modifier.height(12.dp))
                    OutlinedButton(onClick = { vm.refresh() }) { Text("Segarkan") }
                }
            } else {
                LazyColumn {
                    items(investments, key = { it.id }) { inv ->
                        InvestmentRow(inv)
                        Hairline()
                    }
                }
            }
        }

        ExtendedFloatingActionButton(
            onClick = onAddInvestment,
            modifier = Modifier
                .align(Alignment.BottomEnd)
                .padding(24.dp),
        ) {
            Text("Tambah")
        }
    }
}

@Composable
private fun InvestmentRow(inv: InvestmentItem) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 14.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column {
            Text(inv.name, style = MaterialTheme.typography.bodyLarge, fontWeight = FontWeight.Medium)
            Text(
                text = inv.symbol?.takeIf { it.isNotBlank() }
                    ?.let { "${assetTypeLabel(inv.assetType)} · $it" }
                    ?: assetTypeLabel(inv.assetType),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.5f),
            )
        }
        Column(horizontalAlignment = Alignment.End) {
            MoneyText(inv.bookValueIdr, fontSize = 20)
            Text(
                "${formatUnits(inv.units)} @ ${formatIdr(inv.avgBuyPriceIdr)}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.5f),
            )
        }
    }
}

/** Units: buang trailing zero (2.0 → "2", 1.5 → "1.5"). */
private fun formatUnits(units: Double): String =
    if (units == units.toLong().toDouble()) units.toLong().toString() else units.toString()
