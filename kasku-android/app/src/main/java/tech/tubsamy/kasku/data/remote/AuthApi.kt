package tech.tubsamy.kasku.data.remote

import retrofit2.http.Body
import retrofit2.http.POST
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.GoogleRequest
import tech.tubsamy.kasku.data.remote.dto.LoginData
import tech.tubsamy.kasku.data.remote.dto.LoginRequest
import tech.tubsamy.kasku.data.remote.dto.RegisterData
import tech.tubsamy.kasku.data.remote.dto.RegisterRequest

interface AuthApi {
    @POST("auth/login")
    suspend fun login(@Body body: LoginRequest): ApiEnvelope<LoginData>

    @POST("auth/register")
    suspend fun register(@Body body: RegisterRequest): ApiEnvelope<RegisterData>

    /** Login via Google ID token (dari Credential Manager). Auto-register jika email baru. */
    @POST("auth/google")
    suspend fun google(@Body body: GoogleRequest): ApiEnvelope<LoginData>

    /** Refresh token dibawa otomatis oleh cookie `refresh_token` (via CookieJar), bukan body. */
    @POST("auth/refresh")
    suspend fun refresh(): ApiEnvelope<LoginData>
}
