mod config;
mod delivery;
mod domain;
mod infrastructure;
mod proto_gen;
mod usecase;

use std::net::SocketAddr;
use std::sync::Arc;

use axum::{routing::get, Router};
use tokio::signal;
use tonic::transport::Server as TonicServer;
use tracing::{error, info};

use config::Config;
use delivery::grpc_handler::PriceGrpcHandler;
use delivery::http_handler;
use infrastructure::db;
use infrastructure::repository::PriceCacheRepository;
use proto_gen::price::v1::price_service_server::PriceServiceServer;
use usecase::fetch_external::{CoinGeckoClient, MetalsLiveClient};
use usecase::get_price::GetPriceUseCase;

/// Shared application state passed to all Axum handlers.
pub struct AppState {
    pub get_price_uc: Arc<GetPriceUseCase>,
    pub service_version: String,
    pub db_pool: sqlx::PgPool,
}

#[tokio::main]
async fn main() {
    // ── Config ──────────────────────────────────────────────────────────
    let cfg = Config::from_env().expect("gagal load konfigurasi dari environment");

    // ── Logger ──────────────────────────────────────────────────────────
    let env_filter = tracing_subscriber::EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new(&cfg.log_level));

    tracing_subscriber::fmt()
        .with_env_filter(env_filter)
        .json()
        .with_target(false)
        .with_thread_ids(false)
        .init();

    info!(
        service = "price-service",
        version = "1.0.0",
        env = cfg.app_env.as_str(),
        "price-service starting"
    );

    // ── Database ────────────────────────────────────────────────────────
    let pool = db::new_postgres_pool(&cfg.database_url)
        .await
        .expect("gagal koneksi ke PostgreSQL");

    info!("menjalankan database migrations");
    db::run_migrations(&pool)
        .await
        .expect("gagal menjalankan migrations");

    // ── Dependency Injection ────────────────────────────────────────────
    let repo = PriceCacheRepository::new(pool.clone());

    let coingecko = CoinGeckoClient::new(
        cfg.external_request_timeout_seconds,
        cfg.coingecko_api_key.clone(),
    );

    let metals_live = MetalsLiveClient::new(
        cfg.external_request_timeout_seconds,
        cfg.metals_live_url.clone(),
        cfg.gold_usd_idr_rate,
    );

    let get_price_uc = Arc::new(GetPriceUseCase::new(
        repo,
        coingecko,
        metals_live,
        cfg.price_cache_ttl_seconds,
    ));

    // ── Axum HTTP Server ────────────────────────────────────────────────
    let app_state = Arc::new(AppState {
        get_price_uc: get_price_uc.clone(),
        service_version: "1.0.0".to_string(),
        db_pool: pool.clone(),
    });

    let http_app = Router::new()
        .route("/health", get(http_handler::health))
        .route("/v1/prices/{symbol}", get(http_handler::get_price))
        .with_state(app_state);

    let http_addr = SocketAddr::from(([0, 0, 0, 0], cfg.http_port));

    // ── gRPC Server ─────────────────────────────────────────────────────
    let grpc_handler = PriceGrpcHandler::new(get_price_uc.clone());
    let grpc_service = PriceServiceServer::new(grpc_handler);
    let grpc_addr = SocketAddr::from(([0, 0, 0, 0], cfg.grpc_port));

    // ── Start both servers ──────────────────────────────────────────────
    info!(http_port = cfg.http_port, "price-service HTTP server listening");
    info!(grpc_port = cfg.grpc_port, "price-service gRPC server listening");

    let http_server = axum::serve(
        tokio::net::TcpListener::bind(http_addr)
            .await
            .expect("gagal bind HTTP port"),
        http_app,
    )
    .with_graceful_shutdown(shutdown_signal());

    let grpc_server = TonicServer::builder()
        .add_service(grpc_service)
        .serve_with_shutdown(grpc_addr, shutdown_signal());

    tokio::select! {
        result = http_server => {
            if let Err(e) = result {
                error!(error = %e, "HTTP server error");
            }
        }
        result = grpc_server => {
            if let Err(e) = result {
                error!(error = %e, "gRPC server error");
            }
        }
    }

    info!("price-service berhenti dengan bersih");
}

async fn shutdown_signal() {
    let ctrl_c = async {
        signal::ctrl_c()
            .await
            .expect("gagal install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("gagal install SIGTERM handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
    }

    info!("sinyal shutdown diterima");
}
