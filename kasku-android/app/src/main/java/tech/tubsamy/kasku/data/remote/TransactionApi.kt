package tech.tubsamy.kasku.data.remote

import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.CreatedIdDto
import tech.tubsamy.kasku.data.remote.dto.TransactionDto
import tech.tubsamy.kasku.data.remote.dto.TransactionRequest

/**
 * Transaksi ONLINE — transaction-service /v1/transactions (base URL sudah /v1).
 * List mengembalikan envelope { data, summary }; summary diabaikan (ignoreUnknownKeys).
 */
interface TransactionApi {
    @GET("transactions")
    suspend fun listTransactions(): ApiEnvelope<List<TransactionDto>>

    @POST("transactions")
    suspend fun createTransaction(@Body body: TransactionRequest): ApiEnvelope<TransactionDto>

    @PUT("transactions/{id}")
    suspend fun updateTransaction(@Path("id") id: String, @Body body: TransactionRequest): ApiEnvelope<TransactionDto>

    @DELETE("transactions/{id}")
    suspend fun deleteTransaction(@Path("id") id: String): ApiEnvelope<CreatedIdDto>
}
