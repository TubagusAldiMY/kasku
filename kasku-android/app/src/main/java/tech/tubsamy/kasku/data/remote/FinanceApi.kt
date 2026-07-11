package tech.tubsamy.kasku.data.remote

import retrofit2.http.GET
import tech.tubsamy.kasku.data.remote.dto.AccountDto
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope

interface FinanceApi {
    @GET("accounts")
    suspend fun listAccounts(): ApiEnvelope<List<AccountDto>>
}
