package tech.tubsamy.kasku.data.remote

import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import tech.tubsamy.kasku.data.remote.dto.AccountDto
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.BudgetDto
import tech.tubsamy.kasku.data.remote.dto.CategoryDto
import tech.tubsamy.kasku.data.remote.dto.CreateAccountRequest
import tech.tubsamy.kasku.data.remote.dto.CreateBudgetRequest
import tech.tubsamy.kasku.data.remote.dto.CreateCategoryRequest
import tech.tubsamy.kasku.data.remote.dto.CreatedIdDto
import tech.tubsamy.kasku.data.remote.dto.UpdateAccountRequest
import tech.tubsamy.kasku.data.remote.dto.UpdateBudgetRequest

interface FinanceApi {
    @GET("accounts")
    suspend fun listAccounts(): ApiEnvelope<List<AccountDto>>

    @POST("accounts")
    suspend fun createAccount(@Body body: CreateAccountRequest): ApiEnvelope<AccountDto>

    @PUT("accounts/{id}")
    suspend fun updateAccount(@Path("id") id: String, @Body body: UpdateAccountRequest): ApiEnvelope<AccountDto>

    @DELETE("accounts/{id}")
    suspend fun deleteAccount(@Path("id") id: String): ApiEnvelope<CreatedIdDto>

    @GET("categories")
    suspend fun listCategories(): ApiEnvelope<List<CategoryDto>>

    @GET("budgets")
    suspend fun listBudgets(): ApiEnvelope<List<BudgetDto>>

    @POST("budgets")
    suspend fun createBudget(@Body body: CreateBudgetRequest): ApiEnvelope<CreatedIdDto>

    @PUT("budgets/{id}")
    suspend fun updateBudget(@Path("id") id: String, @Body body: UpdateBudgetRequest): ApiEnvelope<CreatedIdDto>

    @DELETE("budgets/{id}")
    suspend fun deleteBudget(@Path("id") id: String): ApiEnvelope<CreatedIdDto>

    @POST("categories")
    suspend fun createCategory(@Body body: CreateCategoryRequest): ApiEnvelope<CategoryDto>

    @PUT("categories/{id}")
    suspend fun updateCategory(
        @Path("id") id: String,
        @Body body: tech.tubsamy.kasku.data.remote.dto.UpdateCategoryRequest,
    ): ApiEnvelope<CreatedIdDto>

    @DELETE("categories/{id}")
    suspend fun deleteCategory(@Path("id") id: String): ApiEnvelope<CreatedIdDto>
}
