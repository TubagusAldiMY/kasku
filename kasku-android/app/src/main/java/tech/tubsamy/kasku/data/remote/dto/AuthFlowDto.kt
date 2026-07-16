package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * DTO untuk alur email verification + password reset (auth-service).
 * Endpoint-endpoint ini hanya membalas { "message": ... } di dalam envelope data.
 */

/** Response data untuk endpoint auth yang hanya mengembalikan { "message": ... }. */
@Serializable
data class MessageData(val message: String = "")

/** Request POST /v1/auth/forgot-password. */
@Serializable
data class ForgotPasswordRequest(val email: String)

/** Request POST /v1/auth/reset-password. Token berasal dari email reset. */
@Serializable
data class ResetPasswordRequest(
    val token: String,
    @SerialName("new_password") val newPassword: String,
)

/** Request POST /v1/auth/resend-verification. */
@Serializable
data class ResendVerificationRequest(val email: String)
