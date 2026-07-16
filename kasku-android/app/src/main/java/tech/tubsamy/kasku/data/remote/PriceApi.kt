package tech.tubsamy.kasku.data.remote

import retrofit2.http.GET
import retrofit2.http.Path
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.PriceDto

/** Harga live per simbol — price-service /v1/prices/{symbol} (base URL sudah /v1). */
interface PriceApi {
    @GET("prices/{symbol}")
    suspend fun getPrice(@Path("symbol") symbol: String): ApiEnvelope<PriceDto>
}
