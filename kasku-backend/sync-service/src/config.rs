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

    /// gRPC address untuk finance-service (e.g. "http://finance-service:9084")
    #[serde(default = "default_finance_grpc_addr")]
    pub finance_service_grpc_addr: String,

    /// gRPC address untuk transaction-service (e.g. "http://transaction-service:9085")
    #[serde(default = "default_transaction_grpc_addr")]
    pub transaction_service_grpc_addr: String,

    /// gRPC address untuk investment-service (e.g. "http://investment-service:9086")
    #[serde(default = "default_investment_grpc_addr")]
    pub investment_service_grpc_addr: String,
}

impl Config {
    pub fn from_env() -> Result<Self, envy::Error> {
        envy::from_env::<Config>()
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
fn default_finance_grpc_addr() -> String {
    "http://localhost:9084".to_string()
}
fn default_transaction_grpc_addr() -> String {
    "http://localhost:9085".to_string()
}
fn default_investment_grpc_addr() -> String {
    "http://localhost:9086".to_string()
}
