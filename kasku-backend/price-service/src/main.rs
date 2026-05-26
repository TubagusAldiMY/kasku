mod config;
mod delivery;
mod domain;
mod infrastructure;
mod proto_gen;
mod usecase;

use std::net::SocketAddr;
use std::sync::Arc;

use axum::{routing::get, Router};
use opentelemetry::trace::TracerProvider as _;
use opentelemetry::KeyValue;
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::Resource;
use tokio::signal;
use tokio::time::{interval, Duration};
use tonic::transport::Server as TonicServer;
use tracing::{error, info, warn};
use tracing_subscriber::prelude::*;

use config::Config;
use delivery::grpc_handler::PriceGrpcHandler;
use delivery::http_handler;
use infrastructure::db;
use infrastructure::repository::PriceCacheRepository;
use proto_gen::price::v1::price_service_server::PriceServiceServer;
use usecase::fetch_external::{CoinGeckoClient, MetalsLiveClient};
use usecase::get_price::GetPriceUseCase;

/// Inisialisasi distributed tracing via OpenTelemetry OTLP.
///
/// Jika `otlp_endpoint` kosong, tracing dinonaktifkan (noop) — service tetap
/// jalan normal. Return `Some(provider)` yang wajib di-shutdown saat service berhenti.
fn init_tracer(
    service_name: &str,
    otlp_endpoint: &str,
) -> Option<opentelemetry_sdk::trace::TracerProvider> {
    if otlp_endpoint.is_empty() {
        return None;
    }

    let resource = Resource::new(vec![KeyValue::new(
        opentelemetry_semantic_conventions::resource::SERVICE_NAME,
        service_name.to_owned(),
    )]);

    opentelemetry_otlp::new_pipeline()
        .tracing()
        .with_exporter(
            opentelemetry_otlp::new_exporter()
                .tonic()
                .with_endpoint(otlp_endpoint),
        )
        .with_trace_config(
            opentelemetry_sdk::trace::Config::default().with_resource(resource),
        )
        .install_batch(opentelemetry_sdk::runtime::Tokio)
        .map_err(|e| eprintln!("[otel] gagal inisialisasi tracer: {e}, tracing dinonaktifkan"))
        .ok()
}

use std::sync::atomic::AtomicU64;

/// Counters exposed via /metrics (Prometheus text format).
/// Pakai AtomicU64 supaya lock-free dan increment cepat dari background scheduler.
#[derive(Default)]
pub struct PriceMetrics {
    pub fetch_success_total: AtomicU64,
    pub fetch_failure_total: AtomicU64,
    pub cache_hit_total: AtomicU64,
    pub cache_miss_total: AtomicU64,
    pub stale_fallback_total: AtomicU64,
}

/// Shared application state passed to all Axum handlers.
pub struct AppState {
    pub get_price_uc: Arc<GetPriceUseCase>,
    pub service_version: String,
    pub db_pool: sqlx_postgres::PgPool,
    pub metrics: Arc<PriceMetrics>,
}

#[tokio::main]
async fn main() {
    // ── Config ──────────────────────────────────────────────────────────
    let cfg = Config::from_env().expect("gagal load konfigurasi dari environment");

    // ── Tracing (logging + OTel distributed tracing) ────────────────────
    let env_filter = tracing_subscriber::EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new(&cfg.log_level));

    let fmt_layer = tracing_subscriber::fmt::layer()
        .json()
        .with_target(false)
        .with_thread_ids(false);

    // OTel layer hanya aktif jika endpoint di-set; noop jika kosong.
    let tracer_provider = init_tracer("price-service", &cfg.otel_exporter_otlp_endpoint);
    let otel_layer = tracer_provider.as_ref().map(|tp| {
        tracing_opentelemetry::layer().with_tracer(tp.tracer("price-service"))
    });

    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt_layer)
        .with(otel_layer)
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
    )
    .expect("gagal inisialisasi CoinGecko client");

    let metals_live = MetalsLiveClient::new(
        cfg.external_request_timeout_seconds,
        cfg.metals_live_url.clone(),
        cfg.gold_usd_idr_rate,
    )
    .expect("gagal inisialisasi metals.live client");

    let get_price_uc = Arc::new(GetPriceUseCase::new(
        repo,
        coingecko,
        metals_live,
        cfg.price_cache_ttl_seconds,
    ));

    start_price_scheduler(
        get_price_uc.clone(),
        cfg.price_scheduler_symbols.clone(),
        cfg.price_scheduler_interval_seconds,
    );

    // ── Axum HTTP Server ────────────────────────────────────────────────
    let metrics = Arc::new(PriceMetrics::default());
    let app_state = Arc::new(AppState {
        get_price_uc: get_price_uc.clone(),
        service_version: "1.0.0".to_string(),
        db_pool: pool.clone(),
        metrics: metrics.clone(),
    });

    let http_app = Router::new()
        .route("/health", get(http_handler::health))
        .route("/metrics", get(http_handler::metrics))
        .route("/v1/prices/:symbol", get(http_handler::get_price))
        .with_state(app_state);

    let http_addr = SocketAddr::from(([0, 0, 0, 0], cfg.http_port));

    // ── gRPC Server ─────────────────────────────────────────────────────
    let grpc_handler = PriceGrpcHandler::new(get_price_uc.clone());
    let grpc_service = PriceServiceServer::new(grpc_handler);
    let grpc_addr = SocketAddr::from(([0, 0, 0, 0], cfg.grpc_port));

    // ── Start both servers ──────────────────────────────────────────────
    info!(
        http_port = cfg.http_port,
        "price-service HTTP server listening"
    );
    info!(
        grpc_port = cfg.grpc_port,
        "price-service gRPC server listening"
    );

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

    // Flush pending OTel spans sebelum proses berakhir.
    if let Some(provider) = tracer_provider {
        if let Err(e) = provider.shutdown() {
            eprintln!("[otel] gagal shutdown tracer provider: {e}");
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

fn start_price_scheduler(
    get_price_uc: Arc<GetPriceUseCase>,
    symbols_raw: String,
    interval_seconds: u64,
) {
    let symbols: Vec<String> = symbols_raw
        .split(',')
        .map(str::trim)
        .filter(|s| !s.is_empty())
        .map(ToOwned::to_owned)
        .collect();

    if symbols.is_empty() || interval_seconds == 0 {
        warn!("price scheduler dinonaktifkan karena symbol kosong atau interval 0");
        return;
    }

    tokio::spawn(async move {
        info!(
            symbols = symbols.join(","),
            interval_seconds = interval_seconds,
            "price scheduler aktif"
        );

        let mut ticker = interval(Duration::from_secs(interval_seconds));
        loop {
            ticker.tick().await;
            for symbol in &symbols {
                match get_price_uc.execute(symbol, "").await {
                    Ok(result) => {
                        info!(
                            symbol = result.symbol.as_str(),
                            is_fresh = result.is_fresh,
                            "scheduler refresh harga selesai"
                        );
                    }
                    Err(err) => {
                        warn!(symbol = symbol.as_str(), error = %err, "scheduler gagal refresh harga");
                    }
                }
            }
        }
    });
}
