use std::time::Duration;
use serde::Deserialize;

#[derive(Debug, Deserialize, Clone)]
pub struct Config {
    // Server
    #[serde(default = "default_http_port")]
    pub http_port: u16,
    #[serde(default = "default_app_env")]
    pub app_env: String,
    #[serde(default = "default_log_level")]
    pub log_level: String,
    #[serde(default = "default_service_version")]
    pub service_version: String,

    // Database
    pub database_url: String,
    #[serde(default = "default_db_max_connections")]
    pub database_max_connections: u32,

    // Redis
    pub redis_url: String,

    // RabbitMQ
    pub rabbitmq_url: String,

    // JWT RS256 (user) — base64-encoded PEM
    pub jwt_private_key: String,
    pub jwt_public_key: String,
    #[serde(default = "default_jwt_access_ttl")]
    pub jwt_access_token_ttl: String,
    #[serde(default = "default_jwt_refresh_ttl")]
    pub jwt_refresh_token_ttl: String,

    // JWT HS256 (admin)
    pub admin_jwt_secret: String,
    #[serde(default = "default_admin_jwt_ttl")]
    pub admin_jwt_ttl_seconds: u64,

    // Argon2id — same names as auth-service
    #[serde(default = "default_argon2_time")]
    pub argon2_time: u32,
    #[serde(default = "default_argon2_memory")]
    pub argon2_memory_kb: u32,
    #[serde(default = "default_argon2_threads")]
    pub argon2_threads: u32,
    #[serde(default = "default_argon2_key_length")]
    pub argon2_key_length: u32,

    // Brute force
    #[serde(default = "default_brute_force_max")]
    pub brute_force_max_attempts: i16,
    #[serde(default = "default_lockout_duration")]
    pub brute_force_lockout_duration: String,

    // SMTP
    pub smtp_host: Option<String>,
    #[serde(default = "default_smtp_port")]
    pub smtp_port: u16,
    pub smtp_username: Option<String>,
    pub smtp_password: Option<String>,
    #[serde(default = "default_smtp_from")]
    pub smtp_from_address: String,
    #[serde(default = "default_smtp_name")]
    pub smtp_from_name: String,

    // Frontend
    #[serde(default = "default_frontend_url")]
    pub frontend_base_url: String,

    // Payment
    pub payment_orchestrator_url: Option<String>,
    pub payment_webhook_secret: Option<String>,

    // Price (MetalsLive only — CoinGecko removed)
    #[serde(default = "default_price_cache_ttl")]
    pub price_cache_ttl_seconds: u64,
    #[serde(default = "default_external_timeout")]
    pub external_request_timeout_seconds: u64,
    #[serde(default = "default_gold_rate")]
    pub gold_usd_idr_rate: f64,
    pub metals_live_url: Option<String>,
    #[serde(default = "default_price_symbols")]
    pub price_scheduler_symbols: String,
    #[serde(default = "default_price_interval")]
    pub price_scheduler_interval_seconds: u64,

    // Admin bootstrap
    pub admin_bootstrap_username: Option<String>,
    pub admin_bootstrap_password: Option<String>,

    // Observability (empty = disabled)
    #[serde(default)]
    pub otel_exporter_otlp_endpoint: String,

    // CORS
    #[serde(default = "default_cors_origins")]
    pub cors_allowed_origins: String,
}

impl Config {
    pub fn from_env() -> anyhow::Result<Self> {
        envy::from_env::<Config>().map_err(|e| anyhow::anyhow!("config error: {}", e))
    }

    pub fn is_development(&self) -> bool {
        self.app_env == "development"
    }

    pub fn jwt_access_ttl(&self) -> anyhow::Result<Duration> {
        parse_duration_str(&self.jwt_access_token_ttl)
    }

    pub fn jwt_refresh_ttl(&self) -> anyhow::Result<Duration> {
        parse_duration_str(&self.jwt_refresh_token_ttl)
    }

    pub fn lockout_duration(&self) -> anyhow::Result<Duration> {
        parse_duration_str(&self.brute_force_lockout_duration)
    }

    pub fn cors_origins(&self) -> Vec<String> {
        self.cors_allowed_origins
            .split(',')
            .map(str::trim)
            .filter(|s| !s.is_empty())
            .map(ToOwned::to_owned)
            .collect()
    }

    pub fn price_symbols(&self) -> Vec<String> {
        self.price_scheduler_symbols
            .split(',')
            .map(str::trim)
            .filter(|s| !s.is_empty())
            .map(ToOwned::to_owned)
            .collect()
    }
}

fn parse_duration_str(s: &str) -> anyhow::Result<Duration> {
    let s = s.trim();
    if s.ends_with('s') {
        let n: u64 = s[..s.len() - 1].parse()?;
        return Ok(Duration::from_secs(n));
    }
    if s.ends_with('m') {
        let n: u64 = s[..s.len() - 1].parse()?;
        return Ok(Duration::from_secs(n * 60));
    }
    if s.ends_with('h') {
        let n: u64 = s[..s.len() - 1].parse()?;
        return Ok(Duration::from_secs(n * 3600));
    }
    Err(anyhow::anyhow!("unknown duration format: {}", s))
}

fn default_http_port() -> u16 { 8080 }
fn default_app_env() -> String { "development".into() }
fn default_log_level() -> String { "info".into() }
fn default_service_version() -> String { "1.0.0".into() }
fn default_db_max_connections() -> u32 { 30 }
fn default_jwt_access_ttl() -> String { "15m".into() }
fn default_jwt_refresh_ttl() -> String { "720h".into() }
fn default_admin_jwt_ttl() -> u64 { 28800 }
fn default_argon2_time() -> u32 { 3 }
fn default_argon2_memory() -> u32 { 65536 }
fn default_argon2_threads() -> u32 { 4 }
fn default_argon2_key_length() -> u32 { 32 }
fn default_brute_force_max() -> i16 { 5 }
fn default_lockout_duration() -> String { "15m".into() }
fn default_smtp_port() -> u16 { 587 }
fn default_smtp_from() -> String { "noreply@kasku.app".into() }
fn default_smtp_name() -> String { "KasKu".into() }
fn default_frontend_url() -> String { "https://kasku.app".into() }
fn default_price_cache_ttl() -> u64 { 900 }
fn default_external_timeout() -> u64 { 10 }
fn default_gold_rate() -> f64 { 16000.0 }
fn default_price_symbols() -> String { "XAU".into() }
fn default_price_interval() -> u64 { 900 }
fn default_cors_origins() -> String { "http://localhost:5173".into() }
