package tech.tubsamy.kasku.data.remote

import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.RecordUnitRequest
import tech.tubsamy.kasku.data.remote.dto.UnitHistoryDto

/**
 * Riwayat & pencatatan transaksi investasi — investment-service /v1/investments
 * (base URL sudah /v1). Daftar instrumen sendiri tetap offline-synced (Room), bukan di sini.
 */
interface InvestmentApi {
    @GET("investments/{id}/history")
    suspend fun getHistory(@Path("id") id: String): ApiEnvelope<List<UnitHistoryDto>>

    @POST("investments/{id}/units")
    suspend fun recordUnit(@Path("id") id: String, @Body body: RecordUnitRequest): ApiEnvelope<UnitHistoryDto>
}
