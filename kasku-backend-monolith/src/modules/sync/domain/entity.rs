use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// A single sync operation sent by the client in a batch push.
#[derive(Debug, Clone, Deserialize)]
pub struct SyncOperation {
    /// Client-generated unique ID for idempotency
    pub sync_id: Uuid,
    /// Entity type: "financial_account", "transaction", "investment_asset"
    pub entity_type: String,
    /// Entity UUID being operated on
    pub entity_id: Uuid,
    /// Operation type: "create", "update", "delete"
    pub operation: String,
    /// JSON payload with entity data
    pub payload: serde_json::Value,
    /// Client-side timestamp when the operation was performed
    pub client_timestamp: DateTime<Utc>,
}

/// Result of processing a single sync operation.
#[derive(Debug, Clone, Serialize)]
pub struct SyncResult {
    pub sync_id: Uuid,
    pub entity_type: String,
    pub entity_id: Uuid,
    pub status: SyncStatus,
    /// Server data returned when conflict is resolved (Server Wins)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub server_data: Option<serde_json::Value>,
}

/// Status of a processed sync operation.
#[derive(Debug, Clone, Serialize, PartialEq)]
pub enum SyncStatus {
    /// Operation applied successfully
    #[serde(rename = "applied")]
    Applied,
    /// Duplicate sync_id — already processed, skipped
    #[serde(rename = "skipped")]
    Skipped,
    /// Conflict detected — server data wins
    #[serde(rename = "conflict")]
    Conflict,
    /// Operation failed due to error
    #[serde(rename = "error")]
    Error,
}

/// Response for batch push.
#[derive(Debug, Serialize)]
pub struct PushResponse {
    pub processed: usize,
    pub conflicts: usize,
    pub skipped: usize,
    pub results: Vec<SyncResult>,
    pub server_timestamp: DateTime<Utc>,
}

/// Response for pull sync.
#[derive(Debug, Serialize)]
pub struct PullResponse {
    pub changes: Vec<EntityChange>,
    pub server_timestamp: DateTime<Utc>,
}

/// A single entity change returned during pull.
#[derive(Debug, Clone, Serialize)]
pub struct EntityChange {
    pub entity_type: String,
    pub entity_id: Uuid,
    pub operation: String,
    pub data: serde_json::Value,
    pub updated_at: DateTime<Utc>,
}
