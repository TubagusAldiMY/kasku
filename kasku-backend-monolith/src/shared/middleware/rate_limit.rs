use deadpool_redis::{Pool as RedisPool, redis::AsyncCommands};
use std::time::Duration;

use crate::app_error::AppError;

pub struct RateLimitResult {
    pub allowed: bool,
    pub retry_after_secs: u64,
}

/// Fixed-window rate limiter using Redis INCR + EXPIRE.
/// Increments a counter for the key; sets TTL on first increment.
/// Simple and safe with deadpool-redis.
pub async fn check(
    pool: &RedisPool,
    key: &str,
    limit: i64,
    window: Duration,
) -> anyhow::Result<RateLimitResult> {
    let mut conn = pool.get().await?;
    let window_secs = window.as_secs() as i64;
    let count: i64 = conn.incr(key, 1i64).await?;
    if count == 1 {
        let _: () = conn.expire(key, window_secs).await?;
    }
    let allowed = count <= limit;
    let retry_after = if allowed { 0 } else { window_secs as u64 };
    Ok(RateLimitResult { allowed, retry_after_secs: retry_after })
}

pub async fn check_register(pool: &RedisPool, client_ip: &str) -> anyhow::Result<RateLimitResult> {
    check(pool, &format!("register:ip:{}", client_ip), 5, Duration::from_secs(15 * 60)).await
}

pub async fn check_login_ip(pool: &RedisPool, client_ip: &str) -> anyhow::Result<RateLimitResult> {
    check(pool, &format!("login:ip:{}", client_ip), 10, Duration::from_secs(60)).await
}

pub async fn check_login_email(pool: &RedisPool, email: &str) -> anyhow::Result<RateLimitResult> {
    check(pool, &format!("login:email:{}", email), 5, Duration::from_secs(60)).await
}

pub async fn check_forgot_password(pool: &RedisPool, email: &str) -> anyhow::Result<RateLimitResult> {
    check(pool, &format!("forgotpw:email:{}", email), 3, Duration::from_secs(3600)).await
}

pub async fn check_refresh(pool: &RedisPool, user_id: &str) -> anyhow::Result<RateLimitResult> {
    check(pool, &format!("refresh:user:{}", user_id), 20, Duration::from_secs(60)).await
}

pub async fn check_sync(pool: &RedisPool, user_id: &str) -> anyhow::Result<RateLimitResult> {
    check(pool, &format!("sync:user:{}", user_id), 60, Duration::from_secs(60)).await
}

pub async fn check_default(pool: &RedisPool, user_id: &str) -> anyhow::Result<RateLimitResult> {
    check(pool, &format!("default:user:{}", user_id), 200, Duration::from_secs(60)).await
}

pub fn rate_limited_if_denied(result: RateLimitResult) -> Result<(), AppError> {
    if result.allowed {
        Ok(())
    } else {
        Err(AppError::RateLimited)
    }
}
