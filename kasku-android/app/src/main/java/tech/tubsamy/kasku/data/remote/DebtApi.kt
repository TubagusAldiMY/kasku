package tech.tubsamy.kasku.data.remote

import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.CreateDebtRequest
import tech.tubsamy.kasku.data.remote.dto.CreatedIdDto
import tech.tubsamy.kasku.data.remote.dto.DebtDto
import tech.tubsamy.kasku.data.remote.dto.DebtPaymentDto
import tech.tubsamy.kasku.data.remote.dto.RecordPaymentRequest
import tech.tubsamy.kasku.data.remote.dto.UpdateDebtRequest

/** Hutang & Piutang — finance-service /v1/debts (path relatif, base URL sudah /v1). */
interface DebtApi {
    @GET("debts")
    suspend fun listDebts(): ApiEnvelope<List<DebtDto>>

    @POST("debts")
    suspend fun createDebt(@Body body: CreateDebtRequest): ApiEnvelope<DebtDto>

    @PUT("debts/{id}")
    suspend fun updateDebt(@Path("id") id: String, @Body body: UpdateDebtRequest): ApiEnvelope<DebtDto>

    // Backend membalas {success,message} tanpa data → CreatedIdDto akan null, diabaikan.
    @DELETE("debts/{id}")
    suspend fun deleteDebt(@Path("id") id: String): ApiEnvelope<CreatedIdDto>

    @POST("debts/{id}/payments")
    suspend fun recordPayment(@Path("id") id: String, @Body body: RecordPaymentRequest): ApiEnvelope<DebtPaymentDto>

    @GET("debts/{id}/payments")
    suspend fun listPayments(@Path("id") id: String): ApiEnvelope<List<DebtPaymentDto>>
}
