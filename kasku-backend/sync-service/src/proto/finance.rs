use crate::proto::common::{EntityChange, SyncUpsertItem, SyncUpsertResult};

/// Request batch upsert financial accounts ke finance-service.
#[derive(Clone, PartialEq, prost::Message)]
pub struct UpsertFinancialAccountsRequest {
    #[prost(string, tag = "1")]
    pub tenant_schema: String,
    #[prost(string, tag = "2")]
    pub user_id: String,
    #[prost(message, repeated, tag = "3")]
    pub items: Vec<SyncUpsertItem>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpsertFinancialAccountsResponse {
    #[prost(message, repeated, tag = "1")]
    pub results: Vec<SyncUpsertResult>,
}

/// Request list perubahan financial accounts sejak timestamp.
#[derive(Clone, PartialEq, prost::Message)]
pub struct ListFinancialAccountsRequest {
    #[prost(string, tag = "1")]
    pub tenant_schema: String,
    /// Unix millis — kembalikan entitas dengan updated_at > since_ms.
    #[prost(int64, tag = "2")]
    pub since_ms: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListFinancialAccountsResponse {
    #[prost(message, repeated, tag = "1")]
    pub changes: Vec<EntityChange>,
}
