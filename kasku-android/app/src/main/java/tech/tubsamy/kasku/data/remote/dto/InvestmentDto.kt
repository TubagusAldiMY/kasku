package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Response GET/POST/PUT investments (daftar instrumen). Monolith & microservices sama-sama
 * snake_case (microservices entity punya json tag). Web memetakan quantity→units dan
 * avg_buy_price→avgBuyPriceIdr; pemetaan itu dilakukan di InvestmentsRepository. current_price/
 * is_price_fresh (enriched) diabaikan di sini — harga live diambil InvestmentMarketRepository.
 */
@Serializable
data class AssetDto(
    val id: String = "",
    val name: String = "",
    @SerialName("asset_type") val assetType: String = "",
    val symbol: String = "",
    val quantity: Double = 0.0,
    @SerialName("avg_buy_price") val avgBuyPrice: Double = 0.0,
    val currency: String = "IDR",
)

/**
 * Request POST investments. Microservices mewajibkan name, asset_type, symbol; quantity &
 * avg_buy_price mengisi posisi awal. Monolith menerima field yang sama.
 */
@Serializable
data class CreateAssetRequest(
    val name: String,
    @SerialName("asset_type") val assetType: String,
    val symbol: String,
    val quantity: Double,
    @SerialName("avg_buy_price") val avgBuyPrice: Double,
    val currency: String = "IDR",
)

/**
 * Request PUT investments/{id}. Hanya name/asset_type/symbol yang dapat diubah — units & avg
 * dikelola lewat pencatatan unit (BUY/SELL), sama seperti web.
 */
@Serializable
data class UpdateAssetRequest(
    val name: String,
    @SerialName("asset_type") val assetType: String,
    val symbol: String,
)
