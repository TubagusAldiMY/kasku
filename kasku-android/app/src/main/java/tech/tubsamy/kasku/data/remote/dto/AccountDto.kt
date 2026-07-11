package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Response GET /accounts. Backend memarshal entity Go TANPA json tag → key PascalCase
 * (ID/Name/AccountType/Balance/Currency). @SerialName memetakannya ke nama idiomatik Kotlin.
 * ponytail: pakai DTO langsung di UI untuk sekarang; domain model + mapping menyusul di M2.
 */
@Serializable
data class AccountDto(
    @SerialName("ID") val id: String = "",
    @SerialName("Name") val name: String = "",
    @SerialName("AccountType") val accountType: String = "",
    @SerialName("Balance") val balance: Long = 0, // rupiah bulat (integer)
    @SerialName("Currency") val currency: String = "IDR",
)
