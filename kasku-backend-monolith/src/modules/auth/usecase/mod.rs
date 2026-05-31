pub mod change_password;
pub mod cleanup;
pub mod forgot_password;
pub mod helpers;
pub mod login;
pub mod logout;
pub mod refresh_token;
pub mod register;
pub mod resend_verification;
pub mod reset_password;
pub mod verify_email;

use std::sync::Arc;

use crate::modules::auth::domain::repository::{
    EmailVerificationRepository, PasswordResetRepository, RefreshTokenRepository, UserRepository,
};
use crate::modules::billing::usecase::GetTierLimitsUseCase;
use crate::modules::user::usecase::ProvisionTenantUseCase;
use deadpool_redis::Pool as RedisPool;
use sqlx_postgres::PgPool;

use self::{
    change_password::ChangePasswordUseCase,
    cleanup::AuthCleanupTask,
    forgot_password::ForgotPasswordUseCase,
    helpers::Argon2Config,
    login::LoginUseCase,
    logout::LogoutUseCase,
    refresh_token::RefreshTokenUseCase,
    register::RegisterUseCase,
    resend_verification::ResendVerificationUseCase,
    reset_password::ResetPasswordUseCase,
    verify_email::VerifyEmailUseCase,
};

pub struct AuthUseCases {
    pub register: RegisterUseCase,
    pub login: LoginUseCase,
    pub verify_email: VerifyEmailUseCase,
    pub resend_verification: ResendVerificationUseCase,
    pub refresh_token: RefreshTokenUseCase,
    pub logout: LogoutUseCase,
    pub forgot_password: ForgotPasswordUseCase,
    pub reset_password: ResetPasswordUseCase,
    pub change_password: ChangePasswordUseCase,
}

impl AuthUseCases {
    pub fn new(
        pool: PgPool,
        user_repo: Arc<dyn UserRepository>,
        ev_repo: Arc<dyn EmailVerificationRepository>,
        rt_repo: Arc<dyn RefreshTokenRepository>,
        reset_repo: Arc<dyn PasswordResetRepository>,
        provision_uc: Arc<ProvisionTenantUseCase>,
        tier_uc: Arc<dyn GetTierLimitsUseCase>,
        redis_pool: RedisPool,
        jwt_private_key: Vec<u8>,
        access_ttl_secs: i64,
        refresh_ttl_secs: i64,
        argon2_cfg: Argon2Config,
        max_attempts: i16,
        lockout_secs: i64,
    ) -> Self {
        let argon2_cfg2 = Argon2Config {
            time: argon2_cfg.time,
            memory_kb: argon2_cfg.memory_kb,
            threads: argon2_cfg.threads,
            key_length: argon2_cfg.key_length,
        };
        let argon2_cfg3 = Argon2Config {
            time: argon2_cfg.time,
            memory_kb: argon2_cfg.memory_kb,
            threads: argon2_cfg.threads,
            key_length: argon2_cfg.key_length,
        };

        Self {
            register: RegisterUseCase::new(
                pool.clone(),
                user_repo.clone(),
                ev_repo.clone(),
                provision_uc,
                argon2_cfg,
            ),
            login: LoginUseCase::new(
                user_repo.clone(),
                rt_repo.clone(),
                tier_uc.clone(),
                jwt_private_key.clone(),
                access_ttl_secs,
                refresh_ttl_secs,
                max_attempts,
                lockout_secs,
            ),
            verify_email: VerifyEmailUseCase::new(user_repo.clone(), ev_repo.clone()),
            resend_verification: ResendVerificationUseCase::new(pool.clone(), user_repo.clone(), ev_repo.clone()),
            refresh_token: RefreshTokenUseCase::new(
                user_repo.clone(),
                rt_repo.clone(),
                tier_uc,
                jwt_private_key,
                access_ttl_secs,
                refresh_ttl_secs,
            ),
            logout: LogoutUseCase::new(rt_repo.clone(), redis_pool),
            forgot_password: ForgotPasswordUseCase::new(pool.clone(), user_repo.clone(), reset_repo.clone()),
            reset_password: ResetPasswordUseCase::new(user_repo.clone(), reset_repo.clone(), argon2_cfg2),
            change_password: ChangePasswordUseCase::new(user_repo.clone(), argon2_cfg3),
        }
    }
}
