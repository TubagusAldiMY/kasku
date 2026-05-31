use std::sync::Arc;
use chrono::Utc;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::billing::domain::repository::{SubscriptionPlanRepository, SubscriptionRepository};
use crate::modules::billing::domain::entity::Subscription;
use crate::modules::user::domain::{error::UserError, repository::UserProfileRepository};
use crate::shared::tenant::user_id_to_schema;

pub struct ProvisionTenantUseCase {
    pool: PgPool,
    profile_repo: Arc<dyn UserProfileRepository>,
    sub_repo: Arc<dyn SubscriptionRepository>,
    plan_repo: Arc<dyn SubscriptionPlanRepository>,
}

impl ProvisionTenantUseCase {
    pub fn new(
        pool: PgPool,
        profile_repo: Arc<dyn UserProfileRepository>,
        sub_repo: Arc<dyn SubscriptionRepository>,
        plan_repo: Arc<dyn SubscriptionPlanRepository>,
    ) -> Self {
        Self { pool, profile_repo, sub_repo, plan_repo }
    }

    /// Idempotent — safe to call multiple times for the same user_id.
    pub async fn execute(&self, user_id: Uuid, email: &str, username: &str) -> anyhow::Result<()> {
        tracing::info!(user_id = %user_id, "starting tenant provisioning");

        // 1. Call provision_tenant() SQL function — creates tenant schema + all tables
        sqlx::query("SELECT public.provision_tenant($1)")
            .bind(user_id)
            .execute(&self.pool)
            .await
            .map_err(|e| anyhow::anyhow!("provision_tenant failed: {}", e))?;

        let tenant_schema = user_id_to_schema(&user_id.to_string());

        // 2. ensure_tenant_runtime_objects() — sync_log, categories unique index, budget_id
        sqlx::query("SELECT public.ensure_tenant_runtime_objects($1)")
            .bind(&tenant_schema)
            .execute(&self.pool)
            .await
            .map_err(|e| anyhow::anyhow!("ensure_tenant_runtime_objects failed: {}", e))?;

        // 3. Create FREE subscription if not exists
        if self.sub_repo.find_by_user_id(user_id).await
            .map_err(|e| anyhow::anyhow!(e))?.is_none()
        {
            let free_plan = self.plan_repo.find_by_name("FREE").await
                .map_err(|e| anyhow::anyhow!(e))?
                .ok_or_else(|| anyhow::anyhow!("FREE plan not found in billing.subscription_plans"))?;

            let now = Utc::now();
            let sub = Subscription {
                id: Uuid::new_v4(),
                user_id,
                plan_id: free_plan.id,
                status: "ACTIVE".to_string(),
                current_period_start: now,
                current_period_end: None,
                created_at: now,
                updated_at: now,
            };
            self.sub_repo.create(&sub).await.map_err(|e| anyhow::anyhow!(e))?;
        }

        // 4. Upsert user profile
        self.profile_repo.upsert(user_id, email, username).await
            .map_err(|e| anyhow::anyhow!(e))?;

        tracing::info!(user_id = %user_id, "tenant provisioning complete");
        Ok(())
    }
}
