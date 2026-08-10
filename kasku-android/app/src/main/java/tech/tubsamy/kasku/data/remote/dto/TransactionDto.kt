package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonNames

/**
 * Response GET/POST/PUT transactions. Seperti akun, backend bisa snake_case (monolith:
 * account_id, transaction_type, ...) ATAU PascalCase (microservices Go entity tanpa json
 * tag: AccountID, TransactionType, AmountIDR, ...). @JsonNames menerima keduanya.
 * transactionDate bisa "YYYY-MM-DD" (monolith) atau RFC3339 (microservices) → di-trim ke
 * tanggal saja di repository.
 */
@OptIn(ExperimentalSerializationApi::class)
@Serializable
data class TransactionDto(
    @JsonNames("id", "ID") val id: String = "",
    @JsonNames("account_id", "AccountID") val accountId: String = "",
    @JsonNames("category_id", "CategoryID") val categoryId: String? = null,
    @JsonNames("transaction_type", "TransactionType") val transactionType: String = "",
    @JsonNames("amount_idr", "AmountIDR") val amountIdr: Long = 0,
    @JsonNames("transaction_date", "TransactionDate") val transactionDate: String = "",
    @JsonNames("notes", "Notes") val notes: String? = null,
)

/**
 * Request body POST & PUT transactions (dipakai keduanya — field identik). category_id null
 * = tanpa kategori. transaction_date = "YYYY-MM-DD". Microservices mewajibkan account_id,
 * transaction_type, amount_idr; monolith menerima semua field ini juga.
 */
@Serializable
data class TransactionRequest(
    @SerialName("account_id") val accountId: String,
    @SerialName("category_id") val categoryId: String? = null,
    @SerialName("transaction_type") val transactionType: String,
    @SerialName("amount_idr") val amountIdr: Long,
    @SerialName("transaction_date") val transactionDate: String,
    val notes: String = "",
)
