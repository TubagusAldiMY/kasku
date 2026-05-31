use deadpool_redis::{Config as RedisConfig, Pool as RedisPool, Runtime};

pub fn new_redis_pool(redis_url: &str) -> anyhow::Result<RedisPool> {
    let cfg = RedisConfig::from_url(redis_url);
    let pool = cfg.create_pool(Some(Runtime::Tokio1))?;
    Ok(pool)
}
