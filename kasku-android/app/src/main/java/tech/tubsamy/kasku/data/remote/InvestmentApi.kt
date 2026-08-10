package tech.tubsamy.kasku.data.remote

import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.AssetDto
import tech.tubsamy.kasku.data.remote.dto.CreateAssetRequest
import tech.tubsamy.kasku.data.remote.dto.CreatedIdDto
import tech.tubsamy.kasku.data.remote.dto.RecordUnitRequest
import tech.tubsamy.kasku.data.remote.dto.UnitHistoryDto
import tech.tubsamy.kasku.data.remote.dto.UpdateAssetRequest

/**
 * Investasi ONLINE — investment-service /v1/investments (base URL sudah /v1).
 * CRUD instrumen + riwayat/pencatatan unit.
 */
interface InvestmentApi {
    @GET("investments")
    suspend fun listInvestments(): ApiEnvelope<List<AssetDto>>

    @POST("investments")
    suspend fun createInvestment(@Body body: CreateAssetRequest): ApiEnvelope<AssetDto>

    @PUT("investments/{id}")
    suspend fun updateInvestment(@Path("id") id: String, @Body body: UpdateAssetRequest): ApiEnvelope<AssetDto>

    @DELETE("investments/{id}")
    suspend fun deleteInvestment(@Path("id") id: String): ApiEnvelope<CreatedIdDto>

    @GET("investments/{id}/history")
    suspend fun getHistory(@Path("id") id: String): ApiEnvelope<List<UnitHistoryDto>>

    @POST("investments/{id}/units")
    suspend fun recordUnit(@Path("id") id: String, @Body body: RecordUnitRequest): ApiEnvelope<UnitHistoryDto>
}
