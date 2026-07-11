package tech.tubsamy.kasku.data.remote.dto

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

/**
 * DTO sync sesuai kontrak sync-service (entity.rs + http_handler.rs).
 * Payload/data/server_data adalah JSON bebas (JsonElement) karena bentuknya
 * per-entity (financial_account/transaction/investment_asset) dan berasal dari
 * to_jsonb DB → snake_case. Engine yang memetakannya ke Room entity.
 */

@Serializable
data class SyncOperation(
    val sync_id: String,
    val entity_type: String,   // financial_account | transaction | investment_asset
    val entity_id: String,
    val operation: String,     // create | update | delete
    val payload: JsonElement,
    val client_timestamp: String,
)

@Serializable
data class PushRequest(
    val operations: List<SyncOperation>,
)

@Serializable
data class SyncResult(
    val sync_id: String,
    val entity_type: String,
    val entity_id: String,
    val status: String,                 // applied | skipped | conflict | error
    val server_data: JsonElement? = null,
)

@Serializable
data class PushResponse(
    val processed: Int = 0,
    val conflicts: Int = 0,
    val skipped: Int = 0,
    val results: List<SyncResult> = emptyList(),
    val server_timestamp: String,
)

@Serializable
data class EntityChange(
    val entity_type: String,
    val entity_id: String,
    val operation: String,              // create | update | delete
    val data: JsonElement,
    val updated_at: String,
)

@Serializable
data class PullResponse(
    val changes: List<EntityChange> = emptyList(),
    val server_timestamp: String,
)
