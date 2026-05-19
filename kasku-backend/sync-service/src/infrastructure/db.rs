use sqlx::postgres::PgPoolOptions;
use sqlx::PgPool;
use tracing::info;

/// Create a new PostgreSQL connection pool.
pub async fn new_postgres_pool(database_url: &str) -> Result<PgPool, sqlx::Error> {
    let pool = PgPoolOptions::new()
        .max_connections(10)
        .min_connections(2)
        .acquire_timeout(std::time::Duration::from_secs(5))
        .idle_timeout(std::time::Duration::from_secs(300))
        .connect(database_url)
        .await?;

    info!("PostgreSQL terhubung");
    Ok(pool)
}

/// Run embedded migrations (public schema metadata only).
/// Per-tenant sync_log dikelola oleh finance-service provision_tenant().
pub async fn run_migrations(pool: &PgPool) -> Result<(), sqlx::migrate::MigrateError> {
    sqlx::migrate!("./migrations").run(pool).await?;
    info!("sync-service migrations applied");
    Ok(())
}

/// Ping the database to check connectivity.
pub async fn ping(pool: &PgPool) -> Result<(), sqlx::Error> {
    sqlx::query("SELECT 1").execute(pool).await?;
    Ok(())
}
