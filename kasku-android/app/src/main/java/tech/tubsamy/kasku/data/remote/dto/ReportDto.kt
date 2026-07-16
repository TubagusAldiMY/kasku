package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Response GET /v1/transactions/reports — backend sudah menghitung ringkasan,
 * breakdown per kategori, dan tren bulanan. Dipakai langsung oleh UI tanpa mapping
 * domain terpisah (ponytail: layar read-only, tak ada mutasi).
 */
@Serializable
data class ReportDto(
    @SerialName("period_from") val periodFrom: String = "",
    @SerialName("period_to") val periodTo: String = "",
    @SerialName("total_income") val totalIncome: Long = 0,
    @SerialName("total_expense") val totalExpense: Long = 0,
    @SerialName("net_amount") val netAmount: Long = 0,
    @SerialName("category_breakdown") val categoryBreakdown: List<CategoryBreakdownDto> = emptyList(),
    @SerialName("monthly_trend") val monthlyTrend: List<MonthlyPointDto> = emptyList(),
)

@Serializable
data class CategoryBreakdownDto(
    @SerialName("category_id") val categoryId: String = "",
    @SerialName("category_name") val categoryName: String = "",
    @SerialName("total_amount") val totalAmount: Long = 0,
    val count: Int = 0,
    val percentage: Double = 0.0,
)

@Serializable
data class MonthlyPointDto(
    val month: String = "",
    val income: Long = 0,
    val expense: Long = 0,
)
