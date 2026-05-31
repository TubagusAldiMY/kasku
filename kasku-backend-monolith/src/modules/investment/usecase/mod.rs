use std::sync::Arc;
use chrono::Utc;
use uuid::Uuid;

use crate::app_error::AppError;
use crate::modules::investment::domain::entity::{AssetType, InvestmentAsset, UnitHistory, UnitTransactionType};
use crate::modules::investment::domain::error::InvestmentError;
use crate::modules::investment::domain::repository::InvestmentRepository;

pub struct InvestmentUseCases {
    pub repo: Arc<dyn InvestmentRepository>,
}

impl InvestmentUseCases {
    pub fn new(repo: Arc<dyn InvestmentRepository>) -> Self {
        Self { repo }
    }

    pub async fn list_assets(&self, schema: &str) -> Result<Vec<InvestmentAsset>, AppError> {
        self.repo.list(schema).await.map_err(inv_err_to_app)
    }

    pub async fn get_asset(&self, schema: &str, id: Uuid) -> Result<InvestmentAsset, AppError> {
        self.repo.find_by_id(schema, id).await
            .map_err(inv_err_to_app)?
            .ok_or(AppError::NotFound)
    }

    pub async fn create_asset(
        &self, schema: &str,
        name: String, asset_type: String, symbol: String,
        quantity: f64, avg_buy_price: f64, currency: String, sort_order: i32,
    ) -> Result<InvestmentAsset, AppError> {
        let now = Utc::now();
        let asset = InvestmentAsset {
            id: Uuid::new_v4(),
            name,
            asset_type: asset_type.parse::<AssetType>()
                .map_err(|_| AppError::Validation(format!("asset_type tidak valid")))?,
            symbol,
            quantity,
            avg_buy_price,
            currency,
            is_deleted: false,
            deleted_at: None,
            sort_order,
            created_at: now,
            updated_at: now,
        };
        self.repo.create(schema, &asset).await.map_err(inv_err_to_app)?;
        Ok(asset)
    }

    pub async fn update_asset(
        &self, schema: &str, id: Uuid,
        name: Option<String>, asset_type: Option<String>, symbol: Option<String>,
        quantity: Option<f64>, avg_buy_price: Option<f64>, currency: Option<String>, sort_order: Option<i32>,
    ) -> Result<InvestmentAsset, AppError> {
        let mut asset = self.repo.find_by_id(schema, id).await
            .map_err(inv_err_to_app)?
            .ok_or(AppError::NotFound)?;
        if let Some(v) = name { asset.name = v; }
        if let Some(v) = asset_type {
            asset.asset_type = v.parse::<AssetType>()
                .map_err(|_| AppError::Validation("asset_type tidak valid".into()))?;
        }
        if let Some(v) = symbol { asset.symbol = v; }
        if let Some(v) = quantity { asset.quantity = v; }
        if let Some(v) = avg_buy_price { asset.avg_buy_price = v; }
        if let Some(v) = currency { asset.currency = v; }
        if let Some(v) = sort_order { asset.sort_order = v; }
        self.repo.update(schema, &asset).await.map_err(inv_err_to_app)?;
        Ok(asset)
    }

    pub async fn delete_asset(&self, schema: &str, id: Uuid) -> Result<(), AppError> {
        let _ = self.repo.find_by_id(schema, id).await
            .map_err(inv_err_to_app)?
            .ok_or(AppError::NotFound)?;
        self.repo.soft_delete(schema, id).await.map_err(inv_err_to_app)
    }

    pub async fn record_unit_transaction(
        &self, schema: &str, asset_id: Uuid,
        transaction_type: String, quantity_change: f64, price_per_unit: f64,
        notes: String, transaction_date: chrono::DateTime<Utc>,
    ) -> Result<UnitHistory, AppError> {
        let tx_type = transaction_type.parse::<UnitTransactionType>()
            .map_err(|_| AppError::Validation("transaction_type tidak valid (BUY/SELL/ADJUST)".into()))?;
        let total_value = quantity_change.abs() * price_per_unit;
        let h = UnitHistory {
            id: Uuid::new_v4(),
            asset_id,
            transaction_type: tx_type,
            quantity_change,
            price_per_unit,
            total_value,
            notes,
            transaction_date,
            recorded_at: Utc::now(),
        };
        self.repo.create_unit_history(schema, &h).await.map_err(inv_err_to_app)?;

        // Recalculate asset quantity from all unit history
        let history = self.repo.list_unit_history(schema, asset_id).await.map_err(inv_err_to_app)?;
        let new_qty: f64 = history.iter().map(|x| x.quantity_change).sum();
        let buy_cost: f64 = history.iter()
            .filter(|x| x.transaction_type == UnitTransactionType::Buy && x.quantity_change > 0.0)
            .map(|x| x.total_value)
            .sum();
        let buy_units: f64 = history.iter()
            .filter(|x| x.transaction_type == UnitTransactionType::Buy && x.quantity_change > 0.0)
            .map(|x| x.quantity_change)
            .sum();
        let new_avg = if buy_units > 0.0 { buy_cost / buy_units } else { 0.0 };

        if let Some(mut asset) = self.repo.find_by_id(schema, asset_id).await.map_err(inv_err_to_app)? {
            asset.quantity = new_qty;
            asset.avg_buy_price = new_avg;
            self.repo.update(schema, &asset).await.map_err(inv_err_to_app)?;
        }

        Ok(h)
    }

    pub async fn get_unit_history(&self, schema: &str, asset_id: Uuid) -> Result<Vec<UnitHistory>, AppError> {
        self.repo.list_unit_history(schema, asset_id).await.map_err(inv_err_to_app)
    }
}

fn inv_err_to_app(e: InvestmentError) -> AppError {
    match e {
        InvestmentError::NotFound => AppError::NotFound,
        InvestmentError::TenantNotProvisioned => AppError::TenantNotProvisioned,
        InvestmentError::Internal(s) => AppError::Internal(anyhow::anyhow!(s)),
    }
}
