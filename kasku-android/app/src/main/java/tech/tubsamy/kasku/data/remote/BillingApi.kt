package tech.tubsamy.kasku.data.remote

import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.PlanDto
import tech.tubsamy.kasku.data.remote.dto.SubscribeRequest
import tech.tubsamy.kasku.data.remote.dto.SubscribeResponseDto
import tech.tubsamy.kasku.data.remote.dto.SubscriptionDto

interface BillingApi {
    @GET("billing/plans")
    suspend fun listPlans(): ApiEnvelope<List<PlanDto>>

    /** 404 (SUBSCRIPTION_NOT_FOUND) saat user belum berlangganan — repo memetakan ke null. */
    @GET("billing/subscription")
    suspend fun getSubscription(): ApiEnvelope<SubscriptionDto>

    @POST("billing/subscribe")
    suspend fun subscribe(@Body body: SubscribeRequest): ApiEnvelope<SubscribeResponseDto>
}
