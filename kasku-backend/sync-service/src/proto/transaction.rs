use crate::proto::common::{EntityChange, SyncUpsertItem, SyncUpsertResult};

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpsertTransactionsRequest {
    #[prost(string, tag = "1")]
    pub tenant_schema: String,
    #[prost(string, tag = "2")]
    pub user_id: String,
    #[prost(message, repeated, tag = "3")]
    pub items: Vec<SyncUpsertItem>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpsertTransactionsResponse {
    #[prost(message, repeated, tag = "1")]
    pub results: Vec<SyncUpsertResult>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListTransactionsRequest {
    #[prost(string, tag = "1")]
    pub tenant_schema: String,
    #[prost(int64, tag = "2")]
    pub since_ms: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListTransactionsResponse {
    #[prost(message, repeated, tag = "1")]
    pub changes: Vec<EntityChange>,
}
