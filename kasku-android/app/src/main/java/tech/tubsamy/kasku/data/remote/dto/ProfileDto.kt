package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** Response GET /v1/users/profile. display_name bisa null (backend belum di-set user). */
@Serializable
data class ProfileDto(
    @SerialName("user_id") val userId: String = "",
    val email: String = "",
    val username: String = "",
    @SerialName("display_name") val displayName: String? = null,
    @SerialName("tenant_schema") val tenantSchema: String = "",
    @SerialName("subscription_tier") val subscriptionTier: String = "",
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("updated_at") val updatedAt: String? = null,
)

/** Request PUT /v1/auth/change-password. Bearer disisipkan otomatis oleh interceptor. */
@Serializable
data class ChangePasswordRequest(
    @SerialName("current_password") val currentPassword: String,
    @SerialName("new_password") val newPassword: String,
)

/** data respons endpoint yang hanya balas { "message": ... }. */
@Serializable
data class MessageDto(val message: String = "")
