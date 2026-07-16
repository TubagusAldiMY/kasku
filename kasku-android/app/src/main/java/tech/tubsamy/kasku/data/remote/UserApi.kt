package tech.tubsamy.kasku.data.remote

import retrofit2.http.GET
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.ProfileDto

interface UserApi {
    @GET("users/profile")
    suspend fun getProfile(): ApiEnvelope<ProfileDto>
}
