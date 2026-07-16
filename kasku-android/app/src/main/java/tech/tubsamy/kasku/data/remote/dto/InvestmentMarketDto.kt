package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Response GET /v1/prices/{symbol} → data { price_idr, is_fresh }. Sama di price-service
 * (Rust) maupun monolith — keduanya snake_case. price_idr bisa besar (mis. harga BTC),
 * jadi Double.
 */
@Serializable
data class PriceDto(
    @SerialName("price_idr") val priceIdr: Double = 0.0,
    @SerialName("is_fresh") val isFresh: Boolean = false,
)

/**
 * Item GET /v1/investments/{id}/history. Go entity.UnitHistory & monolith
 * UnitHistoryResponse sama-sama snake_case. transaction_type: BUY | SELL | ADJUST.
 * transaction_date = RFC3339 → di-trim ke tanggal saja di repository.
 */
@Serializable
data class UnitHistoryDto(
    val id: String = "",
    @SerialName("transaction_type") val transactionType: String = "",
    @SerialName("quantity_change") val quantityChange: Double = 0.0,
    @SerialName("price_per_unit") val pricePerUnit: Double = 0.0,
    @SerialName("total_value") val totalValue: Double = 0.0,
    @SerialName("transaction_date") val transactionDate: String = "",
    val notes: String = "",
)

/**
 * Request POST /v1/investments/{id}/units. Server hitung ulang units & avg dari entri ini.
 * transaction_type WAJIB BUY|SELL, quantity_change>0, price_per_unit>0, transaction_date=YYYY-MM-DD.
 */
@Serializable
data class RecordUnitRequest(
    @SerialName("transaction_type") val transactionType: String,
    @SerialName("quantity_change") val quantityChange: Double,
    @SerialName("price_per_unit") val pricePerUnit: Double,
    @SerialName("transaction_date") val transactionDate: String,
    val notes: String = "",
)
