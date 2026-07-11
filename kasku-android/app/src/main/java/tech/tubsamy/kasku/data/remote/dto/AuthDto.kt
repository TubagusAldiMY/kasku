package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.Serializable

/** Envelope standar respons KasKu: { success, data, error }. */
@Serializable
data class ApiEnvelope<T>(
    val success: Boolean = false,
    val data: T? = null,
    val error: ApiError? = null,
)

/** Hanya bagian error — untuk mem-parse body respons non-2xx. */
@Serializable
data class ErrorEnvelope(
    val error: ApiError? = null,
)

@Serializable
data class ApiError(
    val code: String = "",
    val message: String = "",
)

@Serializable
data class LoginRequest(
    val email: String,
    val password: String,
)

@Serializable
data class LoginData(
    val access_token: String,
    val token_type: String = "Bearer",
    val expires_in: Long = 0,
)
