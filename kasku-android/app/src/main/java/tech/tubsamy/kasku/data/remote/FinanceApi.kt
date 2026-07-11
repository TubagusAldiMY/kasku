package tech.tubsamy.kasku.data.remote

import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import tech.tubsamy.kasku.data.remote.dto.AccountDto
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.CategoryDto
import tech.tubsamy.kasku.data.remote.dto.CreateCategoryRequest

interface FinanceApi {
    @GET("accounts")
    suspend fun listAccounts(): ApiEnvelope<List<AccountDto>>

    @GET("categories")
    suspend fun listCategories(): ApiEnvelope<List<CategoryDto>>

    @POST("categories")
    suspend fun createCategory(@Body body: CreateCategoryRequest): ApiEnvelope<CategoryDto>
}
