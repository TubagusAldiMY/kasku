use std::time::Duration;

use tonic::codec::ProstCodec;
use tonic::transport::Channel;

use crate::proto::investment::{
    ListInvestmentAssetsRequest, ListInvestmentAssetsResponse, UpsertInvestmentAssetsRequest,
    UpsertInvestmentAssetsResponse,
};

const UPSERT_PATH: &str = "/investment.v1.InvestmentInternal/UpsertInvestmentAssets";
const LIST_PATH: &str = "/investment.v1.InvestmentInternal/ListInvestmentAssets";

#[derive(Clone)]
pub struct InvestmentInternalClient {
    channel: Channel,
    timeout: Duration,
}

impl InvestmentInternalClient {
    pub fn new(channel: Channel, timeout: Duration) -> Self {
        Self { channel, timeout }
    }

    pub async fn upsert_investment_assets(
        &self,
        request: UpsertInvestmentAssetsRequest,
    ) -> Result<UpsertInvestmentAssetsResponse, tonic::Status> {
        let mut grpc = tonic::client::Grpc::new(self.channel.clone());
        tokio::time::timeout(self.timeout, grpc.ready())
            .await
            .map_err(|_| tonic::Status::deadline_exceeded("upstream timeout"))?
            .map_err(|_| tonic::Status::unavailable("upstream unavailable"))?;
        let codec =
            ProstCodec::<UpsertInvestmentAssetsRequest, UpsertInvestmentAssetsResponse>::default();
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

    pub async fn list_investment_assets(
        &self,
        request: ListInvestmentAssetsRequest,
    ) -> Result<ListInvestmentAssetsResponse, tonic::Status> {
        let mut grpc = tonic::client::Grpc::new(self.channel.clone());
        tokio::time::timeout(self.timeout, grpc.ready())
            .await
            .map_err(|_| tonic::Status::deadline_exceeded("upstream timeout"))?
            .map_err(|_| tonic::Status::unavailable("upstream unavailable"))?;
        let codec =
            ProstCodec::<ListInvestmentAssetsRequest, ListInvestmentAssetsResponse>::default();
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
