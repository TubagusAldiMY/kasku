package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** Response GET /v1/billing/plans. Limits pakai json tag PascalCase (lihat entity.PlanLimits). */
@Serializable
data class PlanDto(
    val id: String = "",
    val name: String = "",
    @SerialName("price_idr") val priceIdr: Int = 0,
    val limits: PlanLimitsDto = PlanLimitsDto(),
)

@Serializable
data class PlanLimitsDto(
    @SerialName("TierName") val tierName: String = "",
    @SerialName("MaxTransactionsPerMonth") val maxTransactionsPerMonth: Int = 0,
    @SerialName("MaxFinancialAccounts") val maxFinancialAccounts: Int = 0,
    @SerialName("MaxInvestmentInstruments") val maxInvestmentInstruments: Int = 0,
    @SerialName("HistoryRetentionMonths") val historyRetentionMonths: Int = 0,
    @SerialName("EmailNotificationsEnabled") val emailNotificationsEnabled: Boolean = false,
    @SerialName("ExportCsvEnabled") val exportCsvEnabled: Boolean = false,
)

/** Response GET /v1/billing/subscription. Bisa 404 (belum berlangganan) → null di repo. */
@Serializable
data class SubscriptionDto(
    val id: String = "",
    @SerialName("user_id") val userId: String = "",
    @SerialName("plan_id") val planId: String = "",
    val status: String = "",
    @SerialName("current_period_start") val currentPeriodStart: String? = null,
    @SerialName("current_period_end") val currentPeriodEnd: String? = null,
)

/** Request POST /v1/billing/subscribe. payment_method default QRIS, billing_cycle default monthly. */
@Serializable
data class SubscribeRequest(
    @SerialName("plan_id") val planId: String,
    @SerialName("payment_method") val paymentMethod: String = "QRIS",
    @SerialName("billing_cycle") val billingCycle: String = "monthly",
)

/** Response POST /v1/billing/subscribe — info pembayaran untuk ditampilkan/di-redirect. */
@Serializable
data class SubscribeResponseDto(
    @SerialName("payment_id") val paymentId: String = "",
    @SerialName("order_id") val orderId: String = "",
    @SerialName("amount_idr") val amountIdr: Int = 0,
    @SerialName("payment_url") val paymentUrl: String = "",
    @SerialName("qr_string") val qrString: String? = null,
    @SerialName("expires_at") val expiresAt: String? = null,
)
