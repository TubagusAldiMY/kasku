package tech.tubsamy.kasku.data.remote

import retrofit2.http.Body
import retrofit2.http.POST
import retrofit2.http.PUT
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.ChangePasswordRequest
import tech.tubsamy.kasku.data.remote.dto.GoogleRequest
import tech.tubsamy.kasku.data.remote.dto.LoginData
import tech.tubsamy.kasku.data.remote.dto.LoginRequest
import tech.tubsamy.kasku.data.remote.dto.MessageDto
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

    /** POST /v1/auth/forgot-password — kirim email instruksi reset (balasan generik). */
    @POST("auth/forgot-password")
    suspend fun forgotPassword(
        @Body body: tech.tubsamy.kasku.data.remote.dto.ForgotPasswordRequest,
    ): ApiEnvelope<tech.tubsamy.kasku.data.remote.dto.MessageData>

    /** POST /v1/auth/reset-password — token + password baru. */
    @POST("auth/reset-password")
    suspend fun resetPassword(
        @Body body: tech.tubsamy.kasku.data.remote.dto.ResetPasswordRequest,
    ): ApiEnvelope<tech.tubsamy.kasku.data.remote.dto.MessageData>

    /** POST /v1/auth/verify-email?token=<token> — token via query param, tanpa body. */
    @POST("auth/verify-email")
    suspend fun verifyEmail(
        @retrofit2.http.Query("token") token: String,
    ): ApiEnvelope<tech.tubsamy.kasku.data.remote.dto.MessageData>

    /** POST /v1/auth/resend-verification — kirim ulang link verifikasi (balasan generik). */
    @POST("auth/resend-verification")
    suspend fun resendVerification(
        @Body body: tech.tubsamy.kasku.data.remote.dto.ResendVerificationRequest,
    ): ApiEnvelope<tech.tubsamy.kasku.data.remote.dto.MessageData>

    /** Ubah password user login. Bearer disisipkan otomatis oleh interceptor jaringan. */
    @PUT("auth/change-password")
    suspend fun changePassword(@Body body: ChangePasswordRequest): ApiEnvelope<MessageDto>
}
