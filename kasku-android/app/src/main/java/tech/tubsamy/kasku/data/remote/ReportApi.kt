package tech.tubsamy.kasku.data.remote

import retrofit2.http.GET
import retrofit2.http.Query
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.ReportDto

interface ReportApi {
    /** from/to format YYYY-MM-DD; kosong → backend default (bulan berjalan). months = jumlah titik tren. */
    @GET("transactions/reports")
    suspend fun getReport(
        @Query("from") from: String? = null,
        @Query("to") to: String? = null,
        @Query("months") months: Int? = null,
    ): ApiEnvelope<ReportDto>
}
