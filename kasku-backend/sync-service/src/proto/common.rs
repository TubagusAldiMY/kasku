/// Satu operasi sync yang dikirim ke owning service.
/// Field numbers HARUS sinkron dengan Go protowire di grpc/server.go masing-masing service.
#[derive(Clone, PartialEq, prost::Message)]
pub struct SyncUpsertItem {
    /// UUID string — dipakai owning service untuk idempotency jika perlu.
    #[prost(string, tag = "1")]
    pub sync_id: String,
    /// UUID string entity yang dioperasikan.
    #[prost(string, tag = "2")]
    pub entity_id: String,
    /// "create" | "update" | "delete"
    #[prost(string, tag = "3")]
    pub operation: String,
    /// JSON payload dari client (entity data).
    #[prost(bytes = "vec", tag = "4")]
    pub payload: Vec<u8>,
    /// Timestamp client saat operasi dilakukan (Unix millis).
    #[prost(int64, tag = "5")]
    pub client_ts_ms: i64,
}

/// Hasil satu operasi sync dari owning service.
#[derive(Clone, PartialEq, prost::Message)]
pub struct SyncUpsertResult {
    #[prost(string, tag = "1")]
    pub sync_id: String,
    /// "applied" | "error"
    #[prost(string, tag = "2")]
    pub status: String,
    /// JSON bytes — diisi jika status = "conflict" (tidak digunakan saat ini
    /// karena conflict detection dilakukan di sync-service sebelum gRPC call).
    #[prost(bytes = "vec", tag = "3")]
    pub server_data: Vec<u8>,
}

/// Satu perubahan entity yang dikembalikan saat pull sync.
#[derive(Clone, PartialEq, prost::Message)]
pub struct EntityChange {
    #[prost(string, tag = "1")]
    pub entity_id: String,
    /// "upsert" | "delete"
    #[prost(string, tag = "2")]
    pub operation: String,
    /// JSON bytes — full entity data (to_jsonb hasil dari DB).
    #[prost(bytes = "vec", tag = "3")]
    pub data: Vec<u8>,
    /// Unix millis dari updated_at entity.
    #[prost(int64, tag = "4")]
    pub updated_at_ms: i64,
}
