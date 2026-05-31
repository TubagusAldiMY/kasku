use std::sync::Arc;
use chrono::Utc;
use jsonwebtoken::{encode, Algorithm, EncodingKey, Header};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::modules::auth::domain::{entity::RefreshToken, error::AuthError, repository::{RefreshTokenRepository, UserRepository}};
use crate::modules::auth::usecase::helpers::{generate_secure_token, mask_email, sha256_hex, verify_password};
use crate::modules::billing::usecase::GetTierLimitsUseCase;

#[derive(Debug, Serialize, Deserialize)]
struct JwtClaims {
    sub: String,
    jti: String,
    email: String,
    tenant_schema: String,
    subscription_tier: String,
    exp: i64,
    iat: i64,
}

pub struct LoginInput {
    pub email: String,
    pub password: String,
    pub user_agent: Option<String>,
    pub ip_address: Option<String>,
}

pub struct LoginOutput {
    pub access_token: String,
    pub refresh_token: String,
    pub user_id: Uuid,
    pub email: String,
    pub username: String,
}

pub struct LoginUseCase {
    user_repo: Arc<dyn UserRepository>,
    refresh_token_repo: Arc<dyn RefreshTokenRepository>,
    tier_uc: Arc<dyn GetTierLimitsUseCase>,
    jwt_private_key: Vec<u8>,
    access_token_ttl_secs: i64,
    refresh_token_ttl_secs: i64,
    max_attempts: i16,
    lockout_secs: i64,
}

impl LoginUseCase {
    pub fn new(
        user_repo: Arc<dyn UserRepository>,
        refresh_token_repo: Arc<dyn RefreshTokenRepository>,
        tier_uc: Arc<dyn GetTierLimitsUseCase>,
        jwt_private_key: Vec<u8>,
        access_token_ttl_secs: i64,
        refresh_token_ttl_secs: i64,
        max_attempts: i16,
        lockout_secs: i64,
    ) -> Self {
        Self {
            user_repo, refresh_token_repo, tier_uc,
            jwt_private_key,
            access_token_ttl_secs, refresh_token_ttl_secs,
            max_attempts, lockout_secs,
        }
    }

    pub async fn execute(&self, input: LoginInput) -> Result<LoginOutput, AuthError> {
        let user = self.user_repo.find_by_email(&input.email).await?
            .ok_or(AuthError::InvalidCredentials)?;

        let now = Utc::now();

        if user.is_account_locked(now) {
            let until = user.locked_until.unwrap().to_rfc3339();
            return Err(AuthError::AccountLocked(until));
        }

        let password_valid = verify_password(&input.password, &user.password_hash)
            .map_err(|e| AuthError::Internal(e))?;

        if !password_valid {
            self.user_repo.increment_failed_login_and_lock(
                user.id,
                self.max_attempts,
                self.lockout_secs,
            ).await?;
            return Err(AuthError::InvalidCredentials);
        }

        if !user.email_verified {
            return Err(AuthError::AccountNotVerified);
        }

        self.user_repo.update_login_success(user.id).await?;

        // Get tier for JWT claim
        let tier = self.tier_uc.get_tier_name(user.id).await.unwrap_or_else(|_| "FREE".to_string());

        let tenant_schema = crate::shared::tenant::user_id_to_schema(&user.id.to_string());
        let jti = Uuid::new_v4().to_string();
        let exp = (now + chrono::Duration::seconds(self.access_token_ttl_secs)).timestamp();

        let claims = JwtClaims {
            sub: user.id.to_string(),
            jti: jti.clone(),
            email: user.email.clone(),
            tenant_schema,
            subscription_tier: tier,
            exp,
            iat: now.timestamp(),
        };

        let encoding_key = EncodingKey::from_rsa_pem(&self.jwt_private_key)
            .map_err(|e| AuthError::Internal(anyhow::anyhow!("JWT key error: {}", e)))?;

        let access_token = encode(&Header::new(Algorithm::RS256), &claims, &encoding_key)
            .map_err(|e| AuthError::Internal(anyhow::anyhow!("JWT encode error: {}", e)))?;

        // Refresh token
        let (raw_refresh, refresh_hash) = generate_secure_token()
            .map_err(|e| AuthError::Internal(e))?;

        let rt = RefreshToken {
            id: Uuid::new_v4(),
            user_id: user.id,
            token_hash: refresh_hash,
            user_agent: input.user_agent,
            ip_address: input.ip_address,
            expires_at: now + chrono::Duration::seconds(self.refresh_token_ttl_secs),
            is_revoked: false,
            revoked_at: None,
            created_at: now,
        };
        self.refresh_token_repo.create(&rt).await?;

        tracing::info!(user_id = %user.id, email = mask_email(&user.email), "login success");

        Ok(LoginOutput {
            access_token,
            refresh_token: raw_refresh,
            user_id: user.id,
            email: user.email,
            username: user.username,
        })
    }
}
