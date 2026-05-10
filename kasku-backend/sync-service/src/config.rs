use serde::Deserialize;

/// Configuration loaded from environment variables.
#[derive(Debug, Deserialize)]
pub struct Config {
    /// HTTP server port (default: 8088)
    #[serde(default = "default_http_port")]
    pub http_port: u16,

    /// PostgreSQL connection string (kasku_finance database)
    pub database_url: String,

    /// Application environment
    #[serde(default = "default_app_env")]
    pub app_env: String,

    /// Log level
    #[serde(default = "default_log_level")]
    pub log_level: String,
}

impl Config {
    pub fn from_env() -> Result<Self, envy::Error> {
        envy::from_env::<Config>()
    }

    pub fn is_development(&self) -> bool {
        self.app_env == "development"
    }
}

fn default_http_port() -> u16 {
    8088
}
fn default_app_env() -> String {
    "development".to_string()
}
fn default_log_level() -> String {
    "info".to_string()
}
