use serde::Deserialize;

/// Configuration loaded from environment variables.
#[derive(Debug, Deserialize)]
pub struct Config {
    /// HTTP server port (default: 8087)
    #[serde(default = "default_http_port")]
    pub http_port: u16,

    /// gRPC server port (default: 9087)
    #[serde(default = "default_grpc_port")]
    pub grpc_port: u16,

    /// PostgreSQL connection string
    pub database_url: String,

    /// Application environment (development/production)
    #[serde(default = "default_app_env")]
    pub app_env: String,

    /// Log level (debug/info/warn/error)
    #[serde(default = "default_log_level")]
    pub log_level: String,

    /// CoinGecko API key (optional, free tier works without it)
    #[serde(default)]
    pub coingecko_api_key: String,

    /// Price cache TTL in seconds (default: 900 = 15 minutes)
    #[serde(default = "default_cache_ttl")]
    pub price_cache_ttl_seconds: u64,

    /// External HTTP request timeout in seconds (default: 10)
    #[serde(default = "default_external_timeout")]
    pub external_request_timeout_seconds: u64,

    /// Gold USD to IDR exchange rate (static, from env)
    #[serde(default = "default_gold_usd_idr_rate")]
    pub gold_usd_idr_rate: f64,

    /// metals.live API URL
    #[serde(default = "default_metals_live_url")]
    pub metals_live_url: String,

    /// Comma-separated symbols refreshed by the background scheduler.
    #[serde(default = "default_scheduler_symbols")]
    pub price_scheduler_symbols: String,

    /// Background scheduler interval in seconds.
    #[serde(default = "default_scheduler_interval")]
    pub price_scheduler_interval_seconds: u64,
}

impl Config {
    /// Load configuration from environment variables.
    pub fn from_env() -> Result<Self, envy::Error> {
        envy::from_env::<Config>()
    }
}

fn default_http_port() -> u16 {
    8087
}
fn default_grpc_port() -> u16 {
    9087
}
fn default_app_env() -> String {
    "development".to_string()
}
fn default_log_level() -> String {
    "info".to_string()
}
fn default_cache_ttl() -> u64 {
    900
}
fn default_external_timeout() -> u64 {
    10
}
fn default_gold_usd_idr_rate() -> f64 {
    16000.0
}
fn default_metals_live_url() -> String {
    "https://api.metals.live/v1/spot/gold".to_string()
}
fn default_scheduler_symbols() -> String {
    "bitcoin,ethereum,XAU".to_string()
}
fn default_scheduler_interval() -> u64 {
    900
}
