use tonic::codec::ProstCodec;
use tonic::transport::Channel;

use crate::proto::finance::{
    ListFinancialAccountsRequest, ListFinancialAccountsResponse,
    UpsertFinancialAccountsRequest, UpsertFinancialAccountsResponse,
};

const UPSERT_PATH: &str = "/finance.v1.FinanceInternal/UpsertFinancialAccounts";
const LIST_PATH: &str = "/finance.v1.FinanceInternal/ListFinancialAccounts";

#[derive(Clone)]
pub struct FinanceInternalClient {
    channel: Channel,
}

impl FinanceInternalClient {
    pub fn new(channel: Channel) -> Self {
        Self { channel }
    }

    pub async fn upsert_financial_accounts(
        &self,
        request: UpsertFinancialAccountsRequest,
    ) -> Result<UpsertFinancialAccountsResponse, tonic::Status> {
        let mut grpc = tonic::client::Grpc::new(self.channel.clone());
        grpc.ready().await.map_err(|e| {
            tonic::Status::unavailable(format!("finance-service gRPC not ready: {e}"))
        })?;
        let codec =
            ProstCodec::<UpsertFinancialAccountsRequest, UpsertFinancialAccountsResponse>::default();
        let path = UPSERT_PATH
            .parse::<http::uri::PathAndQuery>()
            .map_err(|_| tonic::Status::internal("invalid gRPC path constant"))?;
        grpc.unary(tonic::Request::new(request), path, codec)
            .await
            .map(|r| r.into_inner())
    }

    pub async fn list_financial_accounts(
        &self,
        request: ListFinancialAccountsRequest,
    ) -> Result<ListFinancialAccountsResponse, tonic::Status> {
        let mut grpc = tonic::client::Grpc::new(self.channel.clone());
        grpc.ready().await.map_err(|e| {
            tonic::Status::unavailable(format!("finance-service gRPC not ready: {e}"))
        })?;
        let codec =
            ProstCodec::<ListFinancialAccountsRequest, ListFinancialAccountsResponse>::default();
        let path = LIST_PATH
            .parse::<http::uri::PathAndQuery>()
            .map_err(|_| tonic::Status::internal("invalid gRPC path constant"))?;
        grpc.unary(tonic::Request::new(request), path, codec)
            .await
            .map(|r| r.into_inner())
    }
}
