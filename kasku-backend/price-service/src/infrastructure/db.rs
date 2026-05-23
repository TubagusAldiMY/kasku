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

/// Run database migrations from the migrations directory.
pub async fn run_migrations(pool: &PgPool) -> Result<(), sqlx::Error> {
    // Use raw SQL since we have a single migration file
    let migration_sql = include_str!("../../migrations/001_create_price_cache.sql");
    sqlx::raw_sql::raw_sql(migration_sql).execute(pool).await?;
    info!("Database migrations berhasil dijalankan");
    Ok(())
}

/// Ping the database to check connectivity.
pub async fn ping(pool: &PgPool) -> Result<(), sqlx::Error> {
    sqlx::query::query("SELECT 1").execute(pool).await?;
    Ok(())
}
