use async_trait::async_trait;
use chrono::{DateTime, Utc};
use uuid::Uuid;

use super::entity::{InvestmentAsset, UnitHistory};
use super::error::InvestmentError;

#[async_trait]
pub trait InvestmentRepository: Send + Sync {
    async fn list(&self, schema: &str) -> Result<Vec<InvestmentAsset>, InvestmentError>;
    async fn find_by_id(&self, schema: &str, id: Uuid) -> Result<Option<InvestmentAsset>, InvestmentError>;
    async fn create(&self, schema: &str, asset: &InvestmentAsset) -> Result<(), InvestmentError>;
    async fn update(&self, schema: &str, asset: &InvestmentAsset) -> Result<(), InvestmentError>;
    async fn soft_delete(&self, schema: &str, id: Uuid) -> Result<(), InvestmentError>;
    async fn list_unit_history(&self, schema: &str, asset_id: Uuid) -> Result<Vec<UnitHistory>, InvestmentError>;
    async fn create_unit_history(&self, schema: &str, h: &UnitHistory) -> Result<(), InvestmentError>;
    async fn list_since(&self, schema: &str, since: DateTime<Utc>) -> Result<Vec<InvestmentAsset>, InvestmentError>;
}
