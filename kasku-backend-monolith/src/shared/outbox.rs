use std::sync::Arc;
use std::time::Duration;
use sqlx_postgres::PgPool;
use tokio_util::sync::CancellationToken;
use tracing::{error, info, warn};
use uuid::Uuid;

use crate::shared::messaging::EventPublisher;

const POLL_INTERVAL: Duration = Duration::from_secs(2);
const BATCH_SIZE: i64 = 25;
const MAX_ATTEMPTS: i32 = 5;

pub struct OutboxDispatcher {
    pool: PgPool,
    publisher: Arc<dyn EventPublisher>,
}

impl OutboxDispatcher {
    pub fn new(pool: PgPool, publisher: Arc<dyn EventPublisher>) -> Self {
        Self { pool, publisher }
    }

    pub async fn run(self, cancel: CancellationToken) {
        let mut interval = tokio::time::interval(POLL_INTERVAL);
        loop {
            tokio::select! {
                _ = cancel.cancelled() => {
                    info!("outbox dispatcher stopping");
                    return;
                }
                _ = interval.tick() => {
                    if let Err(e) = self.flush_table("auth", "auth.outbox_events").await {
                        warn!(error = %e, "auth outbox flush failed");
                    }
                    if let Err(e) = self.flush_table("billing", "billing.outbox_events").await {
                        warn!(error = %e, "billing outbox flush failed");
                    }
                }
            }
        }
    }

    async fn flush_table(&self, label: &str, table: &str) -> anyhow::Result<()> {
        let mut tx = self.pool.begin().await?;

        let rows: Vec<(Uuid, String, Vec<u8>)> = sqlx::query_as(
            &format!(
                "SELECT id, routing_key, payload::text::bytea FROM {table}
                 WHERE published_at IS NULL AND publish_attempts < $1
                 ORDER BY created_at ASC LIMIT $2
                 FOR UPDATE SKIP LOCKED"
            ),
        )
        .bind(MAX_ATTEMPTS)
        .bind(BATCH_SIZE)
        .fetch_all(&mut *tx)
        .await?;

        for (id, routing_key, payload) in rows {
            match self.publisher.publish_raw(&routing_key, &payload).await {
                Ok(()) => {
                    sqlx::query(&format!(
                        "UPDATE {table} SET published_at = now() WHERE id = $1"
                    ))
                    .bind(id)
                    .execute(&mut *tx)
                    .await?;
                }
                Err(e) => {
                    error!(table = label, routing_key = %routing_key, error = %e, "publish failed");
                    sqlx::query(&format!(
                        "UPDATE {table}
                         SET publish_attempts = publish_attempts + 1, last_error = $2
                         WHERE id = $1"
                    ))
                    .bind(id)
                    .bind(e.to_string())
                    .execute(&mut *tx)
                    .await?;
                }
            }
        }

        tx.commit().await?;
        Ok(())
    }
}
