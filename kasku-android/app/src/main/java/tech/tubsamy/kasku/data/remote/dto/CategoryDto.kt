package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Response GET /v1/categories. Backend memarshal dengan json tag snake_case eksplisit
 * (beda dari AccountDto yang PascalCase tanpa tag). categoryType: INCOME | EXPENSE | BOTH.
 */
@Serializable
data class CategoryDto(
    val id: String = "",
    val name: String = "",
    @SerialName("category_type") val categoryType: String = "",
    val icon: String? = null,
    val color: String? = null,
    @SerialName("is_deleted") val isDeleted: Boolean = false,
)
