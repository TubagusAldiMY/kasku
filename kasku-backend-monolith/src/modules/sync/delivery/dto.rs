use serde::{Deserialize, Serialize};
use crate::modules::sync::domain::entity::SyncOperation;

#[derive(Debug, Deserialize)]
pub struct PushRequest {
    pub operations: Vec<SyncOperation>,
}

#[derive(Debug, Deserialize)]
pub struct PullParams {
    pub since: Option<String>,
}
