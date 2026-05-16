mod finance_client;
mod investment_client;
mod transaction_client;

pub use finance_client::FinanceInternalClient;
pub use investment_client::InvestmentInternalClient;
pub use transaction_client::TransactionInternalClient;

use std::time::Duration;

use anyhow::{Context, Result};
use tonic::transport::Endpoint;

/// Kumpulan gRPC clients untuk semua owning service yang perlu dipanggil sync-service.
/// Channel bersifat lazy-connect — koneksi dibuat saat pertama kali dipakai.
#[derive(Clone)]
pub struct SyncGrpcClients {
    pub finance: FinanceInternalClient,
    pub transaction: TransactionInternalClient,
    pub investment: InvestmentInternalClient,
}

impl SyncGrpcClients {
    pub fn connect(
        finance_addr: &str,
        transaction_addr: &str,
        investment_addr: &str,
        request_timeout_ms: u64,
    ) -> Result<Self> {
        let timeout = Duration::from_millis(request_timeout_ms);

        let finance_channel = Endpoint::new(finance_addr.to_string())
            .context("finance_service_grpc_addr tidak valid")?
            .connect_lazy();

        let transaction_channel = Endpoint::new(transaction_addr.to_string())
            .context("transaction_service_grpc_addr tidak valid")?
            .connect_lazy();

        let investment_channel = Endpoint::new(investment_addr.to_string())
            .context("investment_service_grpc_addr tidak valid")?
            .connect_lazy();

        Ok(Self {
            finance: FinanceInternalClient::new(finance_channel, timeout),
            transaction: TransactionInternalClient::new(transaction_channel, timeout),
            investment: InvestmentInternalClient::new(investment_channel, timeout),
        })
    }
}
