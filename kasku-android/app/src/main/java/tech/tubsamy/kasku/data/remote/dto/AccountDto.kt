package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonNames

/**
 * Response GET/POST/PUT accounts. Backend bisa memarshal snake_case (monolith Rust:
 * account_type, balance, ...) ATAU PascalCase (microservices Go entity tanpa json tag:
 * AccountType, Balance, ...). @JsonNames menerima kedua ejaan saat decode — sama seperti
 * web yang menormalkan keduanya. Field lain (color/icon/is_default/created_at) diabaikan
 * (ignoreUnknownKeys) — UI belum memakainya.
 */
@OptIn(ExperimentalSerializationApi::class)
@Serializable
data class AccountDto(
    @JsonNames("id", "ID") val id: String = "",
    @JsonNames("name", "Name") val name: String = "",
    @JsonNames("account_type", "AccountType") val accountType: String = "",
    @JsonNames("balance", "Balance") val balance: Long = 0, // rupiah bulat (integer)
    @JsonNames("currency", "Currency") val currency: String = "IDR",
)

/**
 * Request POST accounts. balance/initial_balance dikirim keduanya karena microservices
 * memakai key `balance`, monolith memakai `initial_balance`; backend yang tak kenal salah
 * satunya mengabaikannya (Go & serde tanpa deny_unknown_fields). color/icon/is_default
 * memakai default backend (ponytail: UI belum mengaturnya).
 */
@Serializable
data class CreateAccountRequest(
    val name: String,
    @SerialName("account_type") val accountType: String,
    val balance: Long = 0,
    @SerialName("initial_balance") val initialBalance: Long = 0,
    val currency: String = "IDR",
)

/**
 * Request PUT accounts/{id}. Balance TIDAK bisa diubah lewat endpoint ini (backend hanya
 * menerima name/type/tampilan) — saldo bergerak lewat transaksi. Microservices mewajibkan
 * account_type; monolith mengabaikannya (unknown field).
 */
@Serializable
data class UpdateAccountRequest(
    val name: String,
    @SerialName("account_type") val accountType: String,
)
