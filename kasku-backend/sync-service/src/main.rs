mod config;
mod delivery;
mod domain;
mod infrastructure;
mod proto;
mod usecase;

use std::net::SocketAddr;
use std::sync::Arc;

use axum::{
    routing::{get, post},
    Router,
};
use tokio::signal;
use tracing::{error, info};

use config::Config;
use delivery::http_handler::{self, AppState, SyncMetrics};
use infrastructure::db;
use infrastructure::grpc::SyncGrpcClients;
use infrastructure::repository::SyncRepository;
use usecase::pull_sync::PullSyncUseCase;
use usecase::push_sync::PushSyncUseCase;

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
        service = "sync-service",
        version = "1.0.0",
        env = cfg.app_env.as_str(),
        "sync-service starting"
    );

    // ── Database ────────────────────────────────────────────────────────
    let pool = db::new_postgres_pool(&cfg.database_url)
        .await
        .expect("gagal koneksi ke PostgreSQL");

    db::run_migrations(&pool)
        .await
        .expect("gagal menjalankan database migrations");

    // ── gRPC Clients (lazy-connect) ─────────────────────────────────────
    let grpc_clients = SyncGrpcClients::connect(
        &cfg.finance_service_grpc_addr,
        &cfg.transaction_service_grpc_addr,
        &cfg.investment_service_grpc_addr,
        cfg.grpc_request_timeout_ms,
    )
    .expect("gagal inisialisasi gRPC clients");

    info!(
        finance_addr = cfg.finance_service_grpc_addr.as_str(),
        transaction_addr = cfg.transaction_service_grpc_addr.as_str(),
        investment_addr = cfg.investment_service_grpc_addr.as_str(),
        grpc_timeout_ms = cfg.grpc_request_timeout_ms,
        "gRPC clients terkonfigurasi (lazy-connect)"
    );

    // ── Dependency Injection ────────────────────────────────────────────
    let repo = SyncRepository::new(pool.clone());
    let push_uc = PushSyncUseCase::new(repo, grpc_clients.clone());
    let pull_uc = PullSyncUseCase::new(grpc_clients);

    let app_state = Arc::new(AppState {
        push_uc,
        pull_uc,
        service_version: "1.0.0".to_string(),
        db_pool: pool,
        metrics: SyncMetrics::default(),
    });

    // ── Axum HTTP Server ────────────────────────────────────────────────
    let app = Router::new()
        .route("/health", get(http_handler::health))
        .route("/metrics", get(http_handler::metrics))
        .route("/v1/sync/push", post(http_handler::push_sync))
        .route("/v1/sync/pull", get(http_handler::pull_sync))
        .with_state(app_state);

    let addr = SocketAddr::from(([0, 0, 0, 0], cfg.http_port));

    info!(
        http_port = cfg.http_port,
        "sync-service HTTP server listening"
    );

    let server = axum::serve(
        tokio::net::TcpListener::bind(addr)
            .await
            .expect("gagal bind HTTP port"),
        app,
    )
    .with_graceful_shutdown(shutdown_signal());

    if let Err(e) = server.await {
        error!(error = %e, "HTTP server error");
    }

    info!("sync-service berhenti dengan bersih");
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
