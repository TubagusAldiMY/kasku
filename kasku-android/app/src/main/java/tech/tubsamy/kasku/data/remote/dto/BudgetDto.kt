package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Response GET /v1/budgets — backend sudah menghitung progress (BudgetWithProgress).
 * Field daily-allowance tidak dipetakan — belum dipakai UI (ponytail).
 */
@Serializable
data class BudgetDto(
    val id: String = "",
    val name: String = "",
    @SerialName("limit_idr") val limitIdr: Long = 0,
    @SerialName("category_name") val categoryName: String = "",
    @SerialName("alert_threshold") val alertThreshold: Int = 0,
    @SerialName("spent_idr") val spentIdr: Long = 0,
    @SerialName("remaining_idr") val remainingIdr: Long = 0,
    @SerialName("progress_percent") val progressPercent: Double = 0.0,
    @SerialName("is_over_budget") val isOverBudget: Boolean = false,
)
