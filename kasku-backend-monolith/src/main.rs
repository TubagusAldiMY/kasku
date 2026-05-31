use std::sync::Arc;
use std::time::Duration;
use tokio::signal;
use tokio_util::sync::CancellationToken;
use tracing::info;

mod app_error;
mod app_state;
mod config;
mod modules;
mod router;
mod shared;

use app_state::AppState;
use config::Config;
use modules::auth::infrastructure::{
    email_verification_repo::PostgresEmailVerificationRepository,
    password_reset_repo::PostgresPasswordResetRepository,
    refresh_token_repo::PostgresRefreshTokenRepository,
    user_repo::PostgresUserRepository,
};
use modules::auth::usecase::{helpers::Argon2Config, AuthUseCases};
use modules::billing::infrastructure::{
    payment_repo::PostgresPaymentRepository,
    plan_repo::PostgresSubscriptionPlanRepository,
    subscription_repo::PostgresSubscriptionRepository,
};
use modules::billing::usecase::BillingUseCases;
use modules::finance::infrastructure::account_repo::PostgresFinancialAccountRepository;
use modules::finance::usecase::FinanceUseCases;
use modules::investment::infrastructure::PostgresInvestmentRepository;
use modules::investment::usecase::InvestmentUseCases;
use modules::notification::infrastructure::{
    consumer::NotificationConsumer,
    preference_repo::PostgresPreferenceRepository,
    smtp_sender::SmtpEmailSender,
    templates,
};
use modules::notification::usecase::{NotificationHandler, NotificationUseCases};
use modules::price::infrastructure::repository::PriceCacheRepository;
use modules::price::usecase::{fetch_external::MetalsLiveClient, get_price::GetPriceUseCase, PriceScheduler, PriceUseCases};
use modules::sync::infrastructure::repository::SyncRepository;
use modules::sync::usecase::SyncUseCases;
use modules::transaction::infrastructure::{
    PostgresBudgetRepository, PostgresCategoryRepository, PostgresTransactionRepository,
};
use modules::transaction::usecase::TransactionUseCases;
use modules::user::infrastructure::profile_repo::PostgresUserProfileRepository;
use modules::user::usecase::UserUseCases;
use modules::admin::infrastructure::{
    PostgresAdminUserRepository, PostgresAdminUserReadRepository, PostgresAuditLogRepository,
};
use modules::admin::usecase::AdminUseCases;
use shared::db::{new_postgres_pool, run_migrations};
use shared::messaging::RabbitMQPublisher;
use shared::outbox::OutboxDispatcher;
use shared::redis::new_redis_pool;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let cfg = Config::from_env()?;

    init_tracing(&cfg);

    info!(version = %cfg.service_version, env = %cfg.app_env, "KasKu monolith starting");

    // Database
    let pool = new_postgres_pool(&cfg.database_url, cfg.database_max_connections).await?;
    run_migrations(&pool).await?;
    info!("database migrations complete");

    // Redis
    let redis_pool = new_redis_pool(&cfg.redis_url)?;

    // JWT keys — base64-encoded PEM from env
    use base64::{Engine as _, engine::general_purpose::STANDARD as BASE64};
    let jwt_private_key = BASE64.decode(cfg.jwt_private_key.trim())
        .map_err(|e| anyhow::anyhow!("JWT_PRIVATE_KEY base64 decode failed: {}", e))?;
    let jwt_public_key = BASE64.decode(cfg.jwt_public_key.trim())
        .map_err(|e| anyhow::anyhow!("JWT_PUBLIC_KEY base64 decode failed: {}", e))?;

    let access_ttl_secs = cfg.jwt_access_ttl()?.as_secs() as i64;
    let refresh_ttl_secs = cfg.jwt_refresh_ttl()?.as_secs() as i64;
    let lockout_secs = cfg.lockout_duration()?.as_secs() as i64;

    let argon2_cfg = Argon2Config {
        time: cfg.argon2_time,
        memory_kb: cfg.argon2_memory_kb,
        threads: cfg.argon2_threads,
        key_length: cfg.argon2_key_length,
    };

    // ---- Shared repos ----
    let plan_repo: Arc<dyn modules::billing::domain::repository::SubscriptionPlanRepository> =
        Arc::new(PostgresSubscriptionPlanRepository::new(pool.clone()));
    let sub_repo: Arc<dyn modules::billing::domain::repository::SubscriptionRepository> =
        Arc::new(PostgresSubscriptionRepository::new(pool.clone()));
    let payment_repo: Arc<dyn modules::billing::domain::repository::PaymentRepository> =
        Arc::new(PostgresPaymentRepository::new(pool.clone()));

    // ---- Billing ----
    let billing_uc = Arc::new(BillingUseCases::new(
        plan_repo.clone(),
        sub_repo.clone(),
        payment_repo,
        cfg.payment_orchestrator_url.clone(),
        cfg.payment_webhook_secret.clone(),
        pool.clone(),
    ));

    // ---- User / Provisioning ----
    let user_uc = Arc::new(UserUseCases::new(
        Arc::new(PostgresUserProfileRepository::new(pool.clone())),
        sub_repo.clone(),
        plan_repo.clone(),
        pool.clone(),
    ));

    // ---- Auth ----
    let auth_uc = Arc::new(AuthUseCases::new(
        pool.clone(),
        Arc::new(PostgresUserRepository::new(pool.clone())),
        Arc::new(PostgresEmailVerificationRepository::new(pool.clone())),
        Arc::new(PostgresRefreshTokenRepository::new(pool.clone())),
        Arc::new(PostgresPasswordResetRepository::new(pool.clone())),
        user_uc.provision_tenant.clone(),
        billing_uc.get_tier_limits.clone(),
        redis_pool.clone(),
        jwt_private_key.clone(),
        access_ttl_secs,
        refresh_ttl_secs,
        argon2_cfg.clone(),
        cfg.brute_force_max_attempts,
        lockout_secs,
    ));

    // ---- Finance ----
    let finance_uc = Arc::new(FinanceUseCases::new(
        Arc::new(PostgresFinancialAccountRepository::new(pool.clone())),
    ));

    // ---- Transaction ----
    let transaction_uc = Arc::new(TransactionUseCases::new(
        Arc::new(PostgresTransactionRepository::new(pool.clone())),
        Arc::new(PostgresCategoryRepository::new(pool.clone())),
        Arc::new(PostgresBudgetRepository::new(pool.clone())),
    ));

    // ---- Investment ----
    let investment_repo: Arc<dyn modules::investment::domain::repository::InvestmentRepository> =
        Arc::new(PostgresInvestmentRepository::new(pool.clone()));
    let investment_uc = Arc::new(InvestmentUseCases::new(investment_repo.clone()));

    // ---- Price ----
    let price_repo = PriceCacheRepository::new(pool.clone());
    let metals_client = MetalsLiveClient::new(
        cfg.external_request_timeout_seconds,
        cfg.metals_live_url.clone().unwrap_or_default(),
        cfg.gold_usd_idr_rate,
    ).map_err(|e| anyhow::anyhow!("MetalsLiveClient init failed: {}", e))?;
    let get_price_uc = Arc::new(GetPriceUseCase::new(
        price_repo,
        metals_client,
        cfg.price_cache_ttl_seconds,
    ));
    let price_scheduler = PriceScheduler {
        uc: get_price_uc.clone(),
        symbols: cfg.price_symbols(),
        interval_seconds: cfg.price_scheduler_interval_seconds,
    };
    let price_uc = Arc::new(PriceUseCases::new(get_price_uc.clone()));

    // ---- Sync ----
    let finance_repo_for_sync: Arc<dyn modules::finance::domain::repository::FinancialAccountRepository> =
        Arc::new(PostgresFinancialAccountRepository::new(pool.clone()));
    let transaction_repo_for_sync: Arc<dyn modules::transaction::domain::repository::TransactionRepository> =
        Arc::new(PostgresTransactionRepository::new(pool.clone()));
    let sync_uc = Arc::new(SyncUseCases::new(
        SyncRepository::new(pool.clone()),
        finance_repo_for_sync,
        transaction_repo_for_sync,
        investment_repo.clone(),
    ));

    // ---- Notification ----
    let pref_repo: Arc<dyn modules::notification::domain::repository::PreferenceRepository> =
        Arc::new(PostgresPreferenceRepository::new(pool.clone()));
    templates::init_templates("./templates")
        .map_err(|e| anyhow::anyhow!("template init failed: {}", e))?;
    let smtp = build_smtp(&cfg)?;
    let notification_handler = Arc::new(NotificationHandler::new(
        pref_repo.clone(),
        smtp,
        cfg.frontend_base_url.clone(),
    ));
    let notification_uc = Arc::new(NotificationUseCases::new(pref_repo));

    // ---- Admin ----
    let admin_uc = Arc::new(AdminUseCases::new(
        Arc::new(PostgresAdminUserRepository::new(pool.clone())),
        Arc::new(PostgresAdminUserReadRepository::new(pool.clone())),
        Arc::new(PostgresAuditLogRepository::new(pool.clone())),
        cfg.admin_jwt_secret.clone(),
        cfg.admin_jwt_ttl_seconds,
        Argon2Config {
            time: cfg.argon2_time,
            memory_kb: cfg.argon2_memory_kb,
            threads: cfg.argon2_threads,
            key_length: cfg.argon2_key_length,
        },
    ));

    // Bootstrap admin if configured
    if let (Some(uname), Some(pwd)) = (&cfg.admin_bootstrap_username, &cfg.admin_bootstrap_password) {
        match admin_uc.bootstrap(uname, pwd).await {
            Ok(true) => info!(username = %uname, "bootstrap admin user created"),
            Ok(false) => {},
            Err(e) => tracing::warn!(error = %e, "admin bootstrap failed"),
        }
    }

    // ---- App State ----
    let state = Arc::new(AppState {
        db_pool: pool.clone(),
        redis_pool: redis_pool.clone(),
        jwt_public_key,
        jwt_private_key,
        admin_jwt_secret: cfg.admin_jwt_secret.clone(),
        auth_uc,
        billing_uc,
        user_uc,
        finance_uc,
        transaction_uc,
        investment_uc,
        price_uc,
        sync_uc,
        notification_uc,
        admin_uc,
    });

    // ---- Background tasks ----
    let cancel = CancellationToken::new();

    // Outbox dispatcher (auth + billing)
    let publisher = Arc::new(
        RabbitMQPublisher::new(&cfg.rabbitmq_url).await
            .map_err(|e| anyhow::anyhow!("RabbitMQ connect failed: {}", e))?
    );
    let outbox = OutboxDispatcher::new(pool.clone(), publisher);
    tokio::spawn(outbox.run(cancel.clone()));

    // Notification consumer
    let notif_consumer = NotificationConsumer::new(cfg.rabbitmq_url.clone(), notification_handler);
    tokio::spawn(notif_consumer.run(cancel.clone()));

    // Price scheduler
    tokio::spawn(price_scheduler.run(cancel.clone()));

    // Subscription expiry checker
    let expire_sub_uc = modules::billing::usecase::expire_subscriptions::ExpireSubscriptionsUseCase::new(
        sub_repo.clone(), pool.clone(),
    );
    tokio::spawn(expire_sub_uc.run(cancel.clone()));

    // Auth token cleanup (daily)
    let rt_repo_cleanup: Arc<dyn modules::auth::domain::repository::RefreshTokenRepository> =
        Arc::new(PostgresRefreshTokenRepository::new(pool.clone()));
    let cleanup_task = modules::auth::usecase::cleanup::AuthCleanupTask::new(
        rt_repo_cleanup,
        Duration::from_secs(86400),
    );
    tokio::spawn(cleanup_task.run(cancel.clone()));

    // ---- HTTP server ----
    let app = router::create_router(state, cfg.cors_origins());
    let addr = format!("0.0.0.0:{}", cfg.http_port);
    let listener = tokio::net::TcpListener::bind(&addr).await?;
    info!(addr = %addr, "listening");

    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal(cancel))
        .await?;

    info!("server stopped");
    Ok(())
}

fn build_smtp(cfg: &Config) -> anyhow::Result<Arc<SmtpEmailSender>> {
    let host = cfg.smtp_host.as_deref().unwrap_or("localhost");
    let username = cfg.smtp_username.as_deref().unwrap_or("");
    let password = cfg.smtp_password.as_deref().unwrap_or("");
    let smtp = SmtpEmailSender::new(
        host, cfg.smtp_port, username, password,
        &cfg.smtp_from_address, &cfg.smtp_from_name,
    ).map_err(|e| anyhow::anyhow!("SMTP init failed: {}", e))?;
    Ok(Arc::new(smtp))
}

async fn shutdown_signal(cancel: CancellationToken) {
    let ctrl_c = async {
        signal::ctrl_c().await.expect("failed to install Ctrl+C handler");
    };
    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("failed to install SIGTERM handler")
            .recv()
            .await;
    };
    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
    }
    info!("shutdown signal received, cancelling tasks");
    cancel.cancel();
    tokio::time::sleep(Duration::from_secs(30)).await;
}

fn init_tracing(cfg: &Config) {
    use tracing_subscriber::{fmt, layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

    let filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new(&cfg.log_level));

    tracing_subscriber::registry()
        .with(filter)
        .with(fmt::layer().json())
        .init();
}
