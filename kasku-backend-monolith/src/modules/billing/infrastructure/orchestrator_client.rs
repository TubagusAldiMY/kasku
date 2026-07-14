use std::time::Duration;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

const DEPOSIT_ENDPOINT_PATH: &str = "/v1/payment/deposit";

/// Body request inisiasi pembayaran ke Payment Orchestrator (field camelCase,
/// sejajar dengan `billing-service` microservice).
#[derive(Debug, Serialize)]
pub struct DepositRequest {
    #[serde(rename = "refId")]
    pub ref_id: String,
    pub amount: i64,
    pub currency: String,
    #[serde(rename = "paymentMethod")]
    pub payment_method: String,
    pub remarks: String,
}

/// Field `data` dari respons sukses (orchestrator mengembalikan snake_case).
#[derive(Debug, Deserialize, Default)]
pub struct DepositData {
    #[serde(default)]
    pub provider_trx_id: String,
    #[serde(default)]
    pub payment_url: String,
    #[serde(default)]
    pub qr_string: String,
    #[serde(default)]
    pub expires_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Deserialize)]
struct DepositEnvelope {
    #[serde(default)]
    success: bool,
    #[serde(default)]
    data: DepositData,
}

/// Client HTTP ke Payment Orchestrator (Bearer token + Idempotency-Key).
pub struct OrchestratorClient {
    base_url: String,
    api_key: String,
    http: reqwest::Client,
}

impl OrchestratorClient {
    pub fn new(base_url: String, api_key: String, timeout: Duration) -> anyhow::Result<Self> {
        let http = reqwest::Client::builder().timeout(timeout).build()?;
        Ok(Self {
            base_url: base_url.trim_end_matches('/').to_string(),
            api_key,
            http,
        })
    }

    /// Menginisiasi pembayaran baru. `idempotency_key` (biasanya order_id) dikirim sebagai
    /// header `Idempotency-Key` untuk mencegah double-charge saat retry jaringan.
    pub async fn initiate_deposit(
        &self,
        req: &DepositRequest,
        idempotency_key: &str,
    ) -> anyhow::Result<DepositData> {
        let url = format!("{}{}", self.base_url, DEPOSIT_ENDPOINT_PATH);
        let resp = self
            .http
            .post(&url)
            .bearer_auth(&self.api_key)
            .header("Idempotency-Key", idempotency_key)
            .header("Accept", "application/json")
            .json(req)
            .send()
            .await?;

        let status = resp.status();
        let body = resp.text().await?;
        if !status.is_success() {
            anyhow::bail!(
                "payment orchestrator mengembalikan HTTP {}: {}",
                status,
                truncate_for_log(&body, 200)
            );
        }

        let env: DepositEnvelope = serde_json::from_str(&body)
            .map_err(|e| anyhow::anyhow!("gagal parse deposit response orchestrator: {}", e))?;
        if !env.success {
            anyhow::bail!("payment orchestrator melaporkan kegagalan inisiasi deposit");
        }

        Ok(env.data)
    }
}

/// Potong string panjang untuk log agar tidak mencatat body sensitif berlebihan.
fn truncate_for_log(s: &str, max: usize) -> String {
    if s.chars().count() <= max {
        s.to_string()
    } else {
        let head: String = s.chars().take(max).collect();
        format!("{head}...[truncated]")
    }
}
