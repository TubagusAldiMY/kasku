pub mod get_profile;
pub mod provision_tenant;

pub use provision_tenant::ProvisionTenantUseCase;
pub use get_profile::GetProfileUseCase;

use std::sync::Arc;
use crate::modules::user::domain::repository::UserProfileRepository;
use crate::modules::billing::domain::repository::{SubscriptionPlanRepository, SubscriptionRepository};

pub struct UserUseCases {
    pub get_profile: GetProfileUseCase,
    pub provision_tenant: Arc<ProvisionTenantUseCase>,
}

impl UserUseCases {
    pub fn new(
        profile_repo: Arc<dyn UserProfileRepository>,
        sub_repo: Arc<dyn SubscriptionRepository>,
        plan_repo: Arc<dyn SubscriptionPlanRepository>,
        pool: sqlx_postgres::PgPool,
    ) -> Self {
        let provision_uc = Arc::new(ProvisionTenantUseCase::new(
            pool.clone(),
            profile_repo.clone(),
            sub_repo,
            plan_repo,
        ));
        Self {
            get_profile: GetProfileUseCase::new(profile_repo),
            provision_tenant: provision_uc,
        }
    }
}
