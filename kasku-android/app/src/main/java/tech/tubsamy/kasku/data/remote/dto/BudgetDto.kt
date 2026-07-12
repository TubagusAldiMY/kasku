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
    @SerialName("category_id") val categoryId: String? = null,
    @SerialName("category_name") val categoryName: String = "",
    @SerialName("alert_threshold") val alertThreshold: Int = 0,
    @SerialName("spent_idr") val spentIdr: Long = 0,
    @SerialName("remaining_idr") val remainingIdr: Long = 0,
    @SerialName("progress_percent") val progressPercent: Double = 0.0,
    @SerialName("is_over_budget") val isOverBudget: Boolean = false,
    @SerialName("daily_limit_enabled") val dailyLimitEnabled: Boolean = false,
    // Ada hanya saat daily_limit_enabled = true (backend omitempty).
    @SerialName("daily_allowance_today_idr") val dailyAllowanceTodayIdr: Long? = null,
    @SerialName("spent_today_idr") val spentTodayIdr: Long? = null,
    @SerialName("daily_remaining_idr") val dailyRemainingIdr: Long? = null,
)

/**
 * Request POST /v1/budgets. period/threshold/start_date TIDAK dikirim — backend
 * default MONTHLY / 80 / hari ini (ponytail: UI belum butuh periode custom).
 */
@Serializable
data class CreateBudgetRequest(
    val name: String,
    @SerialName("limit_idr") val limitIdr: Long,
    @SerialName("category_id") val categoryId: String? = null,
    @SerialName("daily_limit_enabled") val dailyLimitEnabled: Boolean = false,
)

/** Request PUT /v1/budgets/{id}. alert_threshold diteruskan dari nilai lama (tidak diedit UI). */
@Serializable
data class UpdateBudgetRequest(
    val name: String,
    @SerialName("limit_idr") val limitIdr: Long,
    @SerialName("category_id") val categoryId: String? = null,
    @SerialName("alert_threshold") val alertThreshold: Int = 0,
    @SerialName("daily_limit_enabled") val dailyLimitEnabled: Boolean = false,
)

/** data respons POST /v1/budgets: { "id": ... }. */
@Serializable
data class CreatedIdDto(val id: String = "")
