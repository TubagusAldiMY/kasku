use std::sync::Arc;
use chrono::Utc;
use hmac::{Hmac, Mac};
use sha2::Sha256;
use serde::Deserialize;
use serde_json::json;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::billing::domain::{error::BillingError, repository::PaymentRepository};

/// Payload webhook dari Payment Orchestrator — format snake_case, sejajar dengan
/// `billing-service` microservice. `ref_id` == `order_id` internal kita.
#[derive(Debug, Deserialize)]
pub struct WebhookPayload {
    #[serde(rename = "type", default)]
    pub trx_type: String,
    #[serde(default)]
    pub status: String,
    #[serde(default)]
    pub ref_id: String,
    #[serde(default)]
    pub provider_trx_id: Option<String>,
    #[serde(default)]
    pub amount: i64,
}

pub struct HandleWebhookUseCase {
    payment_repo: Arc<dyn PaymentRepository>,
    pool: PgPool,
    webhook_secret: Option<String>,
}

impl HandleWebhookUseCase {
    pub fn new(
        payment_repo: Arc<dyn PaymentRepository>,
        pool: PgPool,
        webhook_secret: Option<String>,
    ) -> Self {
        Self { payment_repo, pool, webhook_secret }
    }

    /// Verifikasi `HMAC-SHA256(secret, raw_body)` terhadap `signature` (hex, dari header
    /// `X-Signature`) SEBELUM memproses apa pun, lalu update payment + subscription.
    ///
    /// Mengembalikan `Ok(())` untuk semua event yang aman diabaikan (ref_id tak dikenal,
    /// payment sudah final/idempotent, guard nilai gagal, status tak dikenal) supaya
    /// orchestrator tidak retry. `Err(Internal)` hanya untuk kegagalan server (agar retry).
    pub async fn execute(&self, raw_body: &[u8], signature: &str) -> Result<(), BillingError> {
        // 1. Signature verification — fail-closed jika secret tidak dikonfigurasi.
        let secret = self
            .webhook_secret
            .as_deref()
            .ok_or(BillingError::InvalidWebhookSignature)?;
        if signature.is_empty() || !verify_signature(secret, raw_body, signature) {
            return Err(BillingError::InvalidWebhookSignature);
        }

        // 2. Parse payload dari body yang SUDAH terverifikasi.
        let payload: WebhookPayload =
            serde_json::from_slice(raw_body).map_err(|_| BillingError::InvalidWebhookPayload)?;
        if payload.ref_id.is_empty() {
            return Err(BillingError::InvalidWebhookPayload);
        }

        // 3. Lookup payment by ref_id (== order_id). Tak dikenal → abaikan aman.
        let payment = match self.payment_repo.find_by_order_id(&payload.ref_id).await? {
            Some(p) => p,
            None => return Ok(()),
        };

        // 4. Idempotency — hanya proses payment yang masih PENDING.
        if payment.status != "PENDING" {
            return Ok(());
        }

        let now = Utc::now();
        match payload.status.as_str() {
            "success" => {
                // Value-integrity guards SEBELUM menyentuh state. Signature hanya menjamin
                // autentisitas pengirim, bukan bahwa nilai payload cocok dengan payment
                // tersimpan. Tolak diam-diam (tanpa mengubah status, tetap 200) jika tipe
                // transaksi bukan deposit atau amount tidak cocok.
                if payload.trx_type != "deposit" || payload.amount != payment.amount_idr {
                    return Ok(());
                }

                let mut tx = self.pool.begin().await?;

                sqlx::query(
                    "UPDATE billing.payments SET status = 'PAID', orchestrator_ref = $2, paid_at = $3, updated_at = $3 WHERE order_id = $1",
                )
                .bind(&payment.order_id)
                .bind(payload.provider_trx_id.as_deref())
                .bind(now)
                .execute(&mut *tx)
                .await?;

                let period_end = now + chrono::Duration::days(payment.duration_days.max(1) as i64);
                sqlx::query(
                    "UPDATE billing.subscriptions SET plan_id = $2, status = 'ACTIVE', current_period_start = $3, current_period_end = $4, updated_at = $3 WHERE user_id = $1",
                )
                .bind(payment.user_id)
                .bind(payment.plan_id)
                .bind(now)
                .bind(period_end)
                .execute(&mut *tx)
                .await?;

                // Outbox untuk notifikasi — email di-resolve dari auth.users (single DB).
                let email: Option<String> =
                    sqlx::query_scalar("SELECT email FROM auth.users WHERE id = $1")
                        .bind(payment.user_id)
                        .fetch_optional(&mut *tx)
                        .await?;
                if let Some(email) = email {
                    let ev_payload = json!({
                        "user_id": payment.user_id.to_string(),
                        "email": email,
                        "order_id": payment.order_id,
                        "amount_idr": payment.amount_idr,
                        "plan_name": "",
                    });
                    sqlx::query(
                        "INSERT INTO billing.outbox_events (id, event_type, routing_key, payload, created_at) VALUES ($1, $2, $3, $4::jsonb, $5)",
                    )
                    .bind(Uuid::new_v4())
                    .bind("payment.succeeded")
                    .bind("payment.succeeded")
                    .bind(
                        serde_json::to_string(&ev_payload)
                            .map_err(|e| BillingError::Internal(anyhow::anyhow!(e)))?,
                    )
                    .bind(now)
                    .execute(&mut *tx)
                    .await?;
                }

                tx.commit().await?;
            }
            "failed" | "expired" => {
                sqlx::query(
                    "UPDATE billing.payments SET status = 'FAILED', updated_at = $2 WHERE order_id = $1",
                )
                .bind(&payment.order_id)
                .bind(now)
                .execute(&self.pool)
                .await?;
            }
            _ => {
                // Event tidak dikenal — jangan ubah apa pun.
            }
        }

        Ok(())
    }
}

/// `HMAC-SHA256(secret, raw_body)` hex-encoded, dibandingkan constant-time dengan signature.
fn verify_signature(secret: &str, raw_body: &[u8], signature: &str) -> bool {
    let mut mac = match Hmac::<Sha256>::new_from_slice(secret.as_bytes()) {
        Ok(m) => m,
        Err(_) => return false,
    };
    mac.update(raw_body);
    let expected = hex::encode(mac.finalize().into_bytes());
    constant_time_eq(expected.as_bytes(), signature.as_bytes())
}

fn constant_time_eq(a: &[u8], b: &[u8]) -> bool {
    if a.len() != b.len() {
        return false;
    }
    a.iter().zip(b.iter()).fold(0u8, |acc, (x, y)| acc | (x ^ y)) == 0
}
