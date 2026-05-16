use chrono::{DateTime, Utc};
use tracing::error;
use uuid::Uuid;

use crate::domain::entity::{EntityChange, PullResponse};
use crate::domain::error::DomainError;
use crate::infrastructure::grpc::SyncGrpcClients;
use crate::infrastructure::tenant::{validate_tenant_schema, verify_tenant_ownership};
use crate::proto::{
    common::EntityChange as ProtoEntityChange,
    finance::ListFinancialAccountsRequest,
    investment::ListInvestmentAssetsRequest,
    transaction::ListTransactionsRequest,
};

/// Pull sync use case.
///
/// Fan-out ke 3 owning service secara paralel via gRPC, aggregasi hasilnya,
/// dan kembalikan semua perubahan sejak `since` diurutkan berdasarkan updated_at.
pub struct PullSyncUseCase {
    grpc: SyncGrpcClients,
}

impl PullSyncUseCase {
    pub fn new(grpc: SyncGrpcClients) -> Self {
        Self { grpc }
    }

    pub async fn execute(
        &self,
        user_id: &str,
        tenant_schema: &str,
        since: DateTime<Utc>,
    ) -> Result<PullResponse, DomainError> {
        validate_tenant_schema(tenant_schema)?;
        verify_tenant_ownership(user_id, tenant_schema)?;

        let since_ms = since.timestamp_millis();
        let schema = tenant_schema.to_string();

        // Fan-out paralel ke semua owning service
        let (finance_result, tx_result, invest_result) = tokio::join!(
            self.grpc.finance.list_financial_accounts(ListFinancialAccountsRequest {
                tenant_schema: schema.clone(),
                since_ms,
            }),
            self.grpc.transaction.list_transactions(ListTransactionsRequest {
                tenant_schema: schema.clone(),
                since_ms,
            }),
            self.grpc.investment.list_investment_assets(ListInvestmentAssetsRequest {
                tenant_schema: schema,
                since_ms,
            }),
        );

        let finance_resp = finance_result.map_err(|s| map_grpc_status(s, "finance-service"))?;
        let tx_resp = tx_result.map_err(|s| map_grpc_status(s, "transaction-service"))?;
        let invest_resp = invest_result.map_err(|s| map_grpc_status(s, "investment-service"))?;

        let mut changes: Vec<EntityChange> = Vec::new();

        for c in finance_resp.changes {
            changes.push(proto_to_domain_change(c, "financial_account")?);
        }
        for c in tx_resp.changes {
            changes.push(proto_to_domain_change(c, "transaction")?);
        }
        for c in invest_resp.changes {
            changes.push(proto_to_domain_change(c, "investment_asset")?);
        }

        changes.sort_by_key(|c| c.updated_at);

        Ok(PullResponse {
            changes,
            server_timestamp: Utc::now(),
        })
    }
}

fn map_grpc_status(status: tonic::Status, service: &'static str) -> DomainError {
    error!(
        service = service,
        code = ?status.code(),
        message = status.message(),
        "gRPC pull call gagal"
    );
    match status.code() {
        tonic::Code::DeadlineExceeded => DomainError::UpstreamTimeout,
        tonic::Code::Unavailable => DomainError::UpstreamUnavailable,
        _ => DomainError::UpstreamUnavailable,
    }
}

fn proto_to_domain_change(
    c: ProtoEntityChange,
    entity_type: &str,
) -> Result<EntityChange, DomainError> {
    let entity_id = Uuid::parse_str(&c.entity_id).map_err(|_| {
        error!(entity_id = %c.entity_id, "upstream mengembalikan UUID invalid");
        DomainError::UpstreamInvalidResponse
    })?;

    let data = if c.data.is_empty() {
        serde_json::Value::Null
    } else {
        serde_json::from_slice(&c.data).map_err(|e| {
            error!(error = %e, "upstream mengembalikan JSON data invalid");
            DomainError::UpstreamInvalidResponse
        })?
    };

    let updated_at = DateTime::from_timestamp_millis(c.updated_at_ms).unwrap_or_else(Utc::now);

    Ok(EntityChange {
        entity_type: entity_type.to_string(),
        entity_id,
        operation: c.operation,
        data,
        updated_at,
    })
}
