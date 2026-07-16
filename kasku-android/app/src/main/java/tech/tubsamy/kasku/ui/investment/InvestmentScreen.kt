package tech.tubsamy.kasku.ui.investment

import androidx.compose.foundation.clickable
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
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.data.InvestmentItem
import tech.tubsamy.kasku.data.MarketPrice
import tech.tubsamy.kasku.ui.components.Hairline
import tech.tubsamy.kasku.ui.components.MoneyText
import tech.tubsamy.kasku.ui.components.SectionLabel
import tech.tubsamy.kasku.ui.formatIdr
import tech.tubsamy.kasku.ui.theme.KasKuTeal
import kotlin.math.abs
import kotlin.math.roundToLong

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
    onOpenDetail: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val investments by vm.investments.collectAsState()
    val prices by vm.prices.collectAsState()

    // Total portofolio: nilai sekarang (harga live bila ada, else modal) & untung/rugi vs modal.
    val totalCurrentValue = investments.sumOf { inv ->
        val live = inv.symbol?.let { prices[it]?.priceIdr }
        if (live != null) (live * inv.units).roundToLong() else inv.bookValueIdr
    }
    val totalCostBasis = investments.sumOf { it.bookValueIdr }
    val totalGainLoss = totalCurrentValue - totalCostBasis
    val hasLive = investments.any { it.symbol?.let { s -> prices.containsKey(s) } == true }

    Box(modifier = modifier.fillMaxSize()) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(24.dp),
        ) {
            SectionLabel(if (hasLive) "Nilai portofolio · live" else "Nilai portofolio · modal")
            Spacer(Modifier.height(10.dp))
            MoneyText(totalCurrentValue, fontSize = 40)
            Spacer(Modifier.height(8.dp))
            if (totalCostBasis > 0 && hasLive) {
                Text(
                    buildString {
                        append("Modal ${formatIdr(totalCostBasis)} · ")
                        append(if (totalGainLoss >= 0) "untung " else "rugi ")
                        append(gainLabel(totalGainLoss, totalCostBasis))
                    },
                    style = MaterialTheme.typography.bodyMedium,
                    color = if (totalGainLoss >= 0) KasKuTeal else MaterialTheme.colorScheme.error,
                )
            } else {
                Text(
                    "${investments.size} aset",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Spacer(Modifier.height(20.dp))
            Hairline()
            Spacer(Modifier.height(4.dp))

            if (investments.isEmpty()) {
                Column(
                    Modifier.fillMaxSize(),
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Text(
                        "Belum ada investasi.",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Spacer(Modifier.height(12.dp))
                    OutlinedButton(onClick = { vm.refresh() }) { Text("Segarkan") }
                }
            } else {
                LazyColumn {
                    items(investments, key = { it.id }) { inv ->
                        InvestmentRow(
                            inv = inv,
                            price = inv.symbol?.let { prices[it] },
                            onClick = { onOpenDetail(inv.id) },
                        )
                        Hairline()
                    }
                }
                Text(
                    "Ketuk aset untuk riwayat, catat beli/jual, atau edit.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 12.dp),
                )
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
private fun InvestmentRow(inv: InvestmentItem, price: MarketPrice?, onClick: () -> Unit) {
    val unitLabel = if (inv.assetType == "GOLD") "gram" else "unit"
    val currentValue = price?.let { (it.priceIdr * inv.units).roundToLong() }
    val gainLoss = currentValue?.let { it - inv.bookValueIdr }

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = 14.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column {
            Text(inv.name, style = MaterialTheme.typography.titleLarge)
            Text(
                text = inv.symbol?.takeIf { it.isNotBlank() }
                    ?.let { "${assetTypeLabel(inv.assetType)} · $it · ${formatUnits(inv.units)} $unitLabel" }
                    ?: "${assetTypeLabel(inv.assetType)} · ${formatUnits(inv.units)} $unitLabel",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        Column(horizontalAlignment = Alignment.End) {
            MoneyText(currentValue ?: inv.bookValueIdr, fontSize = 20)
            if (gainLoss != null && inv.bookValueIdr > 0) {
                Text(
                    gainLabel(gainLoss, inv.bookValueIdr) + priceFreshnessSuffix(price),
                    style = MaterialTheme.typography.bodySmall,
                    fontWeight = FontWeight.SemiBold,
                    color = if (gainLoss >= 0) KasKuTeal else MaterialTheme.colorScheme.error,
                )
            } else {
                Text(
                    "Modal @ ${formatIdr(inv.avgBuyPriceIdr)}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

/** "+12,34% · +Rp 4.500" / "−8,00% · −Rp 1.200". */
private fun gainLabel(gainLoss: Long, costBasis: Long): String {
    val pct = if (costBasis > 0) gainLoss.toDouble() / costBasis * 100 else 0.0
    val sign = if (gainLoss >= 0) "+" else "−"
    val pctStr = (if (pct >= 0) "+" else "−") + "%.2f".format(abs(pct)) + "%"
    return "$pctStr · $sign${formatIdr(abs(gainLoss))}"
}

private fun priceFreshnessSuffix(price: MarketPrice?): String =
    if (price != null && !price.isFresh) " · cache" else ""

/** Units: buang trailing zero (2.0 → "2", 1.5 → "1.5"). */
private fun formatUnits(units: Double): String =
    if (units == units.toLong().toDouble()) units.toLong().toString() else units.toString()
