use std::sync::Arc;
use deadpool_redis::Pool as RedisPool;
use sqlx_postgres::PgPool;

use crate::modules::auth::usecase::AuthUseCases;
use crate::modules::billing::usecase::BillingUseCases;
use crate::modules::user::usecase::UserUseCases;
use crate::modules::finance::usecase::FinanceUseCases;
use crate::modules::transaction::usecase::TransactionUseCases;
use crate::modules::investment::usecase::InvestmentUseCases;
use crate::modules::price::usecase::PriceUseCases;
use crate::modules::sync::usecase::SyncUseCases;
use crate::modules::notification::usecase::NotificationUseCases;
use crate::modules::admin::usecase::AdminUseCases;

#[derive(Clone)]
pub struct AppState {
    pub db_pool: PgPool,
    pub redis_pool: RedisPool,
    pub jwt_public_key: Vec<u8>,
    pub jwt_private_key: Vec<u8>,
    pub admin_jwt_secret: String,

    pub auth_uc: Arc<AuthUseCases>,
    pub billing_uc: Arc<BillingUseCases>,
    pub user_uc: Arc<UserUseCases>,
    pub finance_uc: Arc<FinanceUseCases>,
    pub transaction_uc: Arc<TransactionUseCases>,
    pub investment_uc: Arc<InvestmentUseCases>,
    pub price_uc: Arc<PriceUseCases>,
    pub sync_uc: Arc<SyncUseCases>,
    pub notification_uc: Arc<NotificationUseCases>,
    pub admin_uc: Arc<AdminUseCases>,
}
