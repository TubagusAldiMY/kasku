use std::path::Path;

use sqlx_postgres::{PgPool, PgPoolOptions};
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
    let migrator = sqlx::migrate::Migrator::new(Path::new("./migrations")).await?;
    migrator.run(pool).await?;
    info!("sync-service migrations applied");
    Ok(())
}

/// Ping the database to check connectivity.
pub async fn ping(pool: &PgPool) -> Result<(), sqlx::Error> {
    sqlx::query::query("SELECT 1").execute(pool).await?;
    Ok(())
}

/// Verifikasi bahwa finance-service sudah menjalankan migration 000004
/// yang membuat fungsi provision_tenant() dan struktur sync_log per tenant.
/// Dipanggil saat startup — gagal jika finance-service belum migrate.
pub async fn verify_finance_migrations_applied(pool: &PgPool) -> Result<(), sqlx::Error> {
    let exists: bool = sqlx::query_scalar::query_scalar(
        "SELECT EXISTS(
            SELECT 1 FROM pg_proc p
            JOIN pg_namespace n ON n.oid = p.pronamespace
            WHERE n.nspname = 'public' AND p.proname = 'provision_tenant'
        )",
    )
    .fetch_one(pool)
    .await?;

    if !exists {
        return Err(sqlx::Error::Protocol(
            "finance-service migration 000004 belum dijalankan: fungsi provision_tenant() tidak ditemukan di kasku_finance. \
             Pastikan finance-service sudah healthy sebelum sync-service dijalankan."
                .to_string(),
        ));
    }

    Ok(())
}
