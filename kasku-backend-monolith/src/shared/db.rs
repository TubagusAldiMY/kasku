use sqlx_postgres::PgPool;
use sqlx::migrate::MigrateDatabase;

pub async fn new_postgres_pool(database_url: &str, max_connections: u32) -> anyhow::Result<PgPool> {
    let pool = sqlx_postgres::PgPoolOptions::new()
        .max_connections(max_connections)
        .connect(database_url)
        .await?;
    Ok(pool)
}

pub async fn run_migrations(pool: &PgPool) -> anyhow::Result<()> {
    sqlx::migrate!("./migrations")
        .run(pool)
        .await
        .map_err(|e| anyhow::anyhow!("migration failed: {}", e))
}
