package tech.tubsamy.kasku.data

import retrofit2.HttpException
import tech.tubsamy.kasku.data.remote.BillingApi
import tech.tubsamy.kasku.data.remote.dto.PlanDto
import tech.tubsamy.kasku.data.remote.dto.SubscribeRequest
import tech.tubsamy.kasku.data.remote.dto.SubscribeResponseDto
import tech.tubsamy.kasku.data.remote.dto.SubscriptionDto

/**
 * Langganan — read + inisiasi pembayaran. plans() & subscribe() throw bila gagal
 * (VM tampilkan error). subscription() memetakan 404 (belum berlangganan) ke null.
 */
class BillingRepository(
    private val api: BillingApi,
) {
    suspend fun plans(): List<PlanDto> = api.listPlans().data ?: emptyList()

    /** null = user belum punya langganan (backend balas 404 SUBSCRIPTION_NOT_FOUND). */
    suspend fun subscription(): SubscriptionDto? = try {
        api.getSubscription().data
    } catch (e: HttpException) {
        if (e.code() == 404) null else throw e
    }

    suspend fun subscribe(planId: String): SubscribeResponseDto =
        api.subscribe(SubscribeRequest(planId = planId)).data
            ?: throw IllegalStateException("Respons langganan kosong.")
}
