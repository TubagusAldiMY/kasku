use std::sync::Arc;
use deadpool_redis::Pool as RedisPool;
use chrono::Utc;

use crate::modules::auth::domain::{error::AuthError, repository::RefreshTokenRepository};
use crate::shared::middleware::auth::{blacklist_token, AuthClaims};

pub struct LogoutUseCase {
    refresh_token_repo: Arc<dyn RefreshTokenRepository>,
    redis_pool: RedisPool,
}

impl LogoutUseCase {
    pub fn new(refresh_token_repo: Arc<dyn RefreshTokenRepository>, redis_pool: RedisPool) -> Self {
        Self { refresh_token_repo, redis_pool }
    }

    pub async fn execute(&self, claims: &AuthClaims, raw_refresh_token: Option<String>) -> Result<(), AuthError> {
        // Blacklist current access token JTI until it expires
        let ttl = (claims.expires_at - Utc::now()).num_seconds();
        if ttl > 0 {
            blacklist_token(&self.redis_pool, &claims.jti, ttl)
                .await
                .map_err(|e| AuthError::Internal(e))?;
        }

        // Revoke refresh token if provided
        if let Some(raw) = raw_refresh_token {
            let hash = crate::modules::auth::usecase::helpers::sha256_hex(&raw);
            if let Ok(Some(rt)) = self.refresh_token_repo.find_by_hash(&hash).await {
                if rt.user_id == claims.user_id {
                    self.refresh_token_repo.revoke(rt.id).await?;
                }
            }
        }

        Ok(())
    }
}
