package tech.tubsamy.kasku.data.remote

import retrofit2.http.Body
import retrofit2.http.POST
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.LoginData
import tech.tubsamy.kasku.data.remote.dto.LoginRequest

interface AuthApi {
    @POST("auth/login")
    suspend fun login(@Body body: LoginRequest): ApiEnvelope<LoginData>
}
