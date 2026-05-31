use std::sync::Arc;
use chrono::Utc;
use jsonwebtoken::{encode, Algorithm, EncodingKey, Header};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::modules::auth::domain::{entity::RefreshToken, error::AuthError, repository::{RefreshTokenRepository, UserRepository}};
use crate::modules::auth::usecase::helpers::{generate_secure_token, sha256_hex};
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

pub struct RefreshTokenInput {
    pub raw_refresh_token: String,
    pub user_agent: Option<String>,
    pub ip_address: Option<String>,
}

pub struct RefreshTokenOutput {
    pub access_token: String,
    pub refresh_token: String,
}

pub struct RefreshTokenUseCase {
    user_repo: Arc<dyn UserRepository>,
    refresh_token_repo: Arc<dyn RefreshTokenRepository>,
    tier_uc: Arc<dyn GetTierLimitsUseCase>,
    jwt_private_key: Vec<u8>,
    access_token_ttl_secs: i64,
    refresh_token_ttl_secs: i64,
}

impl RefreshTokenUseCase {
    pub fn new(
        user_repo: Arc<dyn UserRepository>,
        refresh_token_repo: Arc<dyn RefreshTokenRepository>,
        tier_uc: Arc<dyn GetTierLimitsUseCase>,
        jwt_private_key: Vec<u8>,
        access_token_ttl_secs: i64,
        refresh_token_ttl_secs: i64,
    ) -> Self {
        Self { user_repo, refresh_token_repo, tier_uc, jwt_private_key, access_token_ttl_secs, refresh_token_ttl_secs }
    }

    pub async fn execute(&self, input: RefreshTokenInput) -> Result<RefreshTokenOutput, AuthError> {
        let hash = sha256_hex(&input.raw_refresh_token);
        let rt = self.refresh_token_repo.find_by_hash(&hash).await?
            .ok_or(AuthError::InvalidToken)?;

        let now = Utc::now();
        if rt.is_revoked || rt.is_expired(now) {
            return Err(AuthError::InvalidToken);
        }

        let user = self.user_repo.find_by_id(rt.user_id).await?
            .ok_or(AuthError::UserNotFound)?;

        if !user.is_active {
            return Err(AuthError::AccountNotVerified);
        }

        // Revoke old token
        self.refresh_token_repo.revoke(rt.id).await?;

        let tier = self.tier_uc.get_tier_name(user.id).await.unwrap_or_else(|_| "FREE".to_string());
        let tenant_schema = crate::shared::tenant::user_id_to_schema(&user.id.to_string());
        let jti = Uuid::new_v4().to_string();
        let exp = (now + chrono::Duration::seconds(self.access_token_ttl_secs)).timestamp();

        let claims = JwtClaims {
            sub: user.id.to_string(),
            jti,
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

        let (raw_new, new_hash) = generate_secure_token().map_err(|e| AuthError::Internal(e))?;

        let new_rt = RefreshToken {
            id: Uuid::new_v4(),
            user_id: user.id,
            token_hash: new_hash,
            user_agent: input.user_agent,
            ip_address: input.ip_address,
            expires_at: now + chrono::Duration::seconds(self.refresh_token_ttl_secs),
            is_revoked: false,
            revoked_at: None,
            created_at: now,
        };
        self.refresh_token_repo.create(&new_rt).await?;

        Ok(RefreshTokenOutput { access_token, refresh_token: raw_new })
    }
}
