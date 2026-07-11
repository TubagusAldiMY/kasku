package tech.tubsamy.kasku.data.remote

import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Query
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.PullResponse
import tech.tubsamy.kasku.data.remote.dto.PushRequest
import tech.tubsamy.kasku.data.remote.dto.PushResponse

/** Jalur sync offline-first. Lewat mainClient (Bearer + auto-refresh 401). */
interface SyncApi {
    @GET("sync/pull")
    suspend fun pull(@Query("since") since: String): ApiEnvelope<PullResponse>

    @POST("sync/push")
    suspend fun push(@Body body: PushRequest): ApiEnvelope<PushResponse>
}
