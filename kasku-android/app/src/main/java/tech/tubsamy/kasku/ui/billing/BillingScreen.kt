package tech.tubsamy.kasku.ui.billing

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
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.data.remote.dto.PlanDto
import tech.tubsamy.kasku.ui.components.PrimaryButton
import tech.tubsamy.kasku.ui.formatIdr

@Composable
fun BillingScreen(
    vm: BillingViewModel,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val activePlanId = vm.subscription?.planId

    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(24.dp),
    ) {
        Text("Langganan", style = MaterialTheme.typography.headlineMedium)
        Spacer(Modifier.height(4.dp))
        Text(
            vm.subscription?.let { "Status: ${statusLabel(it.status)}" }
                ?: "Pilih paket untuk membuka semua fitur.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        if (vm.error != null) {
            Spacer(Modifier.height(12.dp))
            Text(vm.error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
            Spacer(Modifier.height(8.dp))
            OutlinedButton(onClick = { vm.load() }) { Text("Coba lagi") }
        }

        Spacer(Modifier.height(16.dp))

        if (vm.loading && vm.plans.isEmpty()) {
            Row(Modifier.fillMaxWidth().padding(top = 16.dp), horizontalArrangement = Arrangement.Center) {
                CircularProgressIndicator()
            }
        } else {
            LazyColumn(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                items(vm.plans, key = { it.id }) { plan ->
                    PlanCard(
                        plan = plan,
                        isActive = plan.id == activePlanId,
                        subscribing = vm.subscribingPlanId == plan.id,
                        onSubscribe = { vm.subscribe(plan.id) },
                    )
                }
            }
        }
    }

    vm.payment?.let { p ->
        AlertDialog(
            onDismissRequest = { vm.dismissPayment() },
            title = { Text("Selesaikan pembayaran") },
            text = {
                Column {
                    Text("Tagihan: ${formatIdr(p.amountIdr.toLong())}")
                    Spacer(Modifier.height(8.dp))
                    Text("Order ID: ${p.orderId}", style = MaterialTheme.typography.bodySmall)
                    if (!p.paymentUrl.isNullOrBlank()) {
                        Spacer(Modifier.height(8.dp))
                        Text("Buka tautan pembayaran:", style = MaterialTheme.typography.bodySmall)
                        Text(p.paymentUrl, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = { vm.dismissPayment() }) { Text("Tutup") }
            },
        )
    }
}

@Composable
private fun PlanCard(
    plan: PlanDto,
    isActive: Boolean,
    subscribing: Boolean,
    onSubscribe: () -> Unit,
) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(plan.name, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
                Text(
                    if (plan.priceIdr == 0) "Gratis" else "${formatIdr(plan.priceIdr.toLong())}/bln",
                    style = MaterialTheme.typography.bodyLarge,
                )
            }
            Spacer(Modifier.height(8.dp))
            LimitLine("Transaksi/bulan", plan.limits.maxTransactionsPerMonth)
            LimitLine("Akun keuangan", plan.limits.maxFinancialAccounts)
            LimitLine("Instrumen investasi", plan.limits.maxInvestmentInstruments)
            if (plan.limits.exportCsvEnabled) {
                Text("• Ekspor CSV", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }

            Spacer(Modifier.height(12.dp))
            if (isActive) {
                OutlinedButton(onClick = {}, enabled = false, modifier = Modifier.fillMaxWidth()) {
                    Text("Paket aktif")
                }
            } else {
                PrimaryButton(
                    text = "Pilih paket",
                    onClick = onSubscribe,
                    enabled = !subscribing,
                    loading = subscribing,
                )
            }
        }
    }
}

@Composable
private fun LimitLine(label: String, value: Int) {
    // -1 (atau <0) = tak terbatas di banyak backend tier premium.
    val display = if (value < 0) "tak terbatas" else value.toString()
    Text("• $label: $display", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
}

private fun statusLabel(status: String): String = when (status.lowercase()) {
    "active" -> "Aktif"
    "pending" -> "Menunggu pembayaran"
    "expired" -> "Kedaluwarsa"
    "canceled", "cancelled" -> "Dibatalkan"
    else -> status
}
