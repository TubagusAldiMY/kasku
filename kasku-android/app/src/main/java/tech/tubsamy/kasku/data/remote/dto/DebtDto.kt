package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Response /v1/debts. Backend memarshal entity Go TANPA json tag → key PascalCase
 * (ID/Direction/PersonName/...), sama seperti AccountDto. @SerialName memetakannya ke
 * nama idiomatik Kotlin. UserID/CreatedAt/UpdatedAt diabaikan (ignoreUnknownKeys).
 * Direction: RECEIVABLE (piutang) | PAYABLE (hutang). Status: ACTIVE | SETTLED.
 * DueDate = RFC3339 (mis. "2025-01-02T00:00:00Z") atau null.
 */
@Serializable
data class DebtDto(
    @SerialName("ID") val id: String = "",
    @SerialName("Direction") val direction: String = "",
    @SerialName("PersonName") val personName: String = "",
    @SerialName("TotalAmount") val totalAmount: Long = 0,
    @SerialName("RemainingAmount") val remainingAmount: Long = 0,
    @SerialName("DueDate") val dueDate: String? = null,
    @SerialName("Notes") val notes: String = "",
    @SerialName("Status") val status: String = "",
)

/** Response item /v1/debts/{id}/payments — entity DebtPayment (PascalCase, tanpa tag). */
@Serializable
data class DebtPaymentDto(
    @SerialName("ID") val id: String = "",
    @SerialName("Amount") val amount: Long = 0,
    @SerialName("PaymentDate") val paymentDate: String = "",
    @SerialName("Notes") val notes: String = "",
    @SerialName("CreatedAt") val createdAt: String = "",
)

/**
 * Request POST /v1/debts. Body di-bind backend dengan tag snake_case.
 * due_date opsional (YYYY-MM-DD atau null). direction WAJIB RECEIVABLE|PAYABLE, amount > 0.
 */
@Serializable
data class CreateDebtRequest(
    val direction: String,
    @SerialName("person_name") val personName: String,
    val amount: Long,
    @SerialName("due_date") val dueDate: String? = null,
    val notes: String = "",
)

/** Request PUT /v1/debts/{id}. Backend hanya izinkan edit person_name/due_date/notes. */
@Serializable
data class UpdateDebtRequest(
    @SerialName("person_name") val personName: String,
    @SerialName("due_date") val dueDate: String? = null,
    val notes: String = "",
)

/** Request POST /v1/debts/{id}/payments. payment_date kosong → backend pakai hari ini. */
@Serializable
data class RecordPaymentRequest(
    val amount: Long,
    @SerialName("payment_date") val paymentDate: String = "",
    val notes: String = "",
)
