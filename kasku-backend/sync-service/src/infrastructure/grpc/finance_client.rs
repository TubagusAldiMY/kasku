use std::time::Duration;

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
    timeout: Duration,
}

impl FinanceInternalClient {
    pub fn new(channel: Channel, timeout: Duration) -> Self {
        Self { channel, timeout }
    }

    pub async fn upsert_financial_accounts(
        &self,
        request: UpsertFinancialAccountsRequest,
    ) -> Result<UpsertFinancialAccountsResponse, tonic::Status> {
        let mut grpc = tonic::client::Grpc::new(self.channel.clone());
        tokio::time::timeout(self.timeout, grpc.ready())
            .await
            .map_err(|_| tonic::Status::deadline_exceeded("upstream timeout"))?
            .map_err(|_| tonic::Status::unavailable("upstream unavailable"))?;
        let codec =
            ProstCodec::<UpsertFinancialAccountsRequest, UpsertFinancialAccountsResponse>::default();
        let path = UPSERT_PATH
            .parse::<http::uri::PathAndQuery>()
            .map_err(|_| tonic::Status::internal("invalid gRPC path constant"))?;
        let mut tonic_req = tonic::Request::new(request);
        tonic_req.set_timeout(self.timeout);
        tokio::time::timeout(self.timeout, grpc.unary(tonic_req, path, codec))
            .await
            .map_err(|_| tonic::Status::deadline_exceeded("upstream timeout"))?
            .map(|r| r.into_inner())
    }

    pub async fn list_financial_accounts(
        &self,
        request: ListFinancialAccountsRequest,
    ) -> Result<ListFinancialAccountsResponse, tonic::Status> {
        let mut grpc = tonic::client::Grpc::new(self.channel.clone());
        tokio::time::timeout(self.timeout, grpc.ready())
            .await
            .map_err(|_| tonic::Status::deadline_exceeded("upstream timeout"))?
            .map_err(|_| tonic::Status::unavailable("upstream unavailable"))?;
        let codec =
            ProstCodec::<ListFinancialAccountsRequest, ListFinancialAccountsResponse>::default();
        let path = LIST_PATH
            .parse::<http::uri::PathAndQuery>()
            .map_err(|_| tonic::Status::internal("invalid gRPC path constant"))?;
        let mut tonic_req = tonic::Request::new(request);
        tonic_req.set_timeout(self.timeout);
        tokio::time::timeout(self.timeout, grpc.unary(tonic_req, path, codec))
            .await
            .map_err(|_| tonic::Status::deadline_exceeded("upstream timeout"))?
            .map(|r| r.into_inner())
    }
}
