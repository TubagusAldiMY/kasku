use std::sync::Arc;
use axum::{
    extract::{FromRequestParts, Request, State},
    http::{request::Parts, StatusCode},
    middleware::Next,
    response::Response,
};
use chrono::{DateTime, Utc};
use deadpool_redis::Pool as RedisPool;
use jsonwebtoken::{decode, Algorithm, DecodingKey, Validation};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::app_error::AppError;
use crate::app_state::AppState;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct KasKuClaims {
    pub sub: String,
    pub jti: String,
    pub email: String,
    pub tenant_schema: String,
    pub subscription_tier: String,
    pub exp: i64,
    pub iat: i64,
}

#[derive(Debug, Clone)]
pub struct AuthClaims {
    pub user_id: Uuid,
    pub jti: String,
    pub email: String,
    pub tenant_schema: String,
    pub subscription_tier: String,
    pub expires_at: DateTime<Utc>,
}

pub async fn auth_middleware(
    State(state): State<Arc<AppState>>,
    mut req: Request,
    next: Next,
) -> Result<Response, AppError> {
    let token = extract_bearer_token(req.headers())?;
    let claims = verify_token(&token, &state.jwt_public_key, &state.redis_pool).await?;
    req.extensions_mut().insert(claims);
    Ok(next.run(req).await)
}

pub async fn verify_token(
    token_str: &str,
    public_key_pem: &[u8],
    redis_pool: &RedisPool,
) -> Result<AuthClaims, AppError> {
    let decoding_key = DecodingKey::from_rsa_pem(public_key_pem)
        .map_err(|_| AppError::Unauthorized("invalid public key configuration".into()))?;

    let mut validation = Validation::new(Algorithm::RS256);
    validation.validate_exp = true;

    let token_data = decode::<KasKuClaims>(token_str, &decoding_key, &validation)
        .map_err(|e| AppError::Unauthorized(format!("token tidak valid: {}", e)))?;

    let claims = token_data.claims;
    let jti = claims.jti.clone();

    if jti.is_empty() {
        return Err(AppError::Unauthorized("token tidak memiliki JTI".into()));
    }

    // Check blacklist — fail-secure: if Redis errors, reject the token
    let blacklisted = is_blacklisted(redis_pool, &jti).await.map_err(|e| {
        tracing::error!(error = %e, "redis blacklist check failed");
        AppError::Unauthorized("tidak dapat memverifikasi status token".into())
    })?;

    if blacklisted {
        return Err(AppError::Unauthorized("token sudah direvoke".into()));
    }

    let user_id = Uuid::parse_str(&claims.sub)
        .map_err(|_| AppError::Unauthorized("subject token bukan UUID valid".into()))?;

    Ok(AuthClaims {
        user_id,
        jti,
        email: claims.email,
        tenant_schema: claims.tenant_schema,
        subscription_tier: claims.subscription_tier,
        expires_at: DateTime::from_timestamp(claims.exp, 0).unwrap_or_default(),
    })
}

async fn is_blacklisted(pool: &RedisPool, jti: &str) -> anyhow::Result<bool> {
    use deadpool_redis::redis::AsyncCommands;
    let mut conn = pool.get().await?;
    let key = format!("blacklist:jti:{}", jti);
    let exists: bool = conn.exists(&key).await?;
    Ok(exists)
}

pub async fn blacklist_token(pool: &RedisPool, jti: &str, ttl_secs: i64) -> anyhow::Result<()> {
    use deadpool_redis::redis::AsyncCommands;
    let mut conn = pool.get().await?;
    let key = format!("blacklist:jti:{}", jti);
    conn.set_ex::<_, _, ()>(&key, "1", ttl_secs.max(1) as u64).await?;
    Ok(())
}

fn extract_bearer_token(headers: &axum::http::HeaderMap) -> Result<String, AppError> {
    let auth_header = headers
        .get(axum::http::header::AUTHORIZATION)
        .and_then(|v| v.to_str().ok())
        .ok_or_else(|| AppError::Unauthorized("authorization header tidak ditemukan".into()))?;

    let parts: Vec<&str> = auth_header.splitn(2, ' ').collect();
    if parts.len() != 2 || !parts[0].eq_ignore_ascii_case("Bearer") {
        return Err(AppError::Unauthorized("format Authorization header tidak valid".into()));
    }

    let token = parts[1].trim();
    if token.is_empty() {
        return Err(AppError::Unauthorized("bearer token kosong".into()));
    }

    Ok(token.to_string())
}
