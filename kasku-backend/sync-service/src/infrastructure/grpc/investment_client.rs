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
}

impl InvestmentInternalClient {
    pub fn new(channel: Channel) -> Self {
        Self { channel }
    }

    pub async fn upsert_investment_assets(
        &self,
        request: UpsertInvestmentAssetsRequest,
    ) -> Result<UpsertInvestmentAssetsResponse, tonic::Status> {
        let mut grpc = tonic::client::Grpc::new(self.channel.clone());
        grpc.ready().await.map_err(|e| {
            tonic::Status::unavailable(format!("investment-service gRPC not ready: {e}"))
        })?;
        let codec =
            ProstCodec::<UpsertInvestmentAssetsRequest, UpsertInvestmentAssetsResponse>::default();
        let path = UPSERT_PATH
            .parse::<http::uri::PathAndQuery>()
            .map_err(|_| tonic::Status::internal("invalid gRPC path constant"))?;
        grpc.unary(tonic::Request::new(request), path, codec)
            .await
            .map(|r| r.into_inner())
    }

    pub async fn list_investment_assets(
        &self,
        request: ListInvestmentAssetsRequest,
    ) -> Result<ListInvestmentAssetsResponse, tonic::Status> {
        let mut grpc = tonic::client::Grpc::new(self.channel.clone());
        grpc.ready().await.map_err(|e| {
            tonic::Status::unavailable(format!("investment-service gRPC not ready: {e}"))
        })?;
        let codec =
            ProstCodec::<ListInvestmentAssetsRequest, ListInvestmentAssetsResponse>::default();
        let path = LIST_PATH
            .parse::<http::uri::PathAndQuery>()
            .map_err(|_| tonic::Status::internal("invalid gRPC path constant"))?;
        grpc.unary(tonic::Request::new(request), path, codec)
            .await
            .map(|r| r.into_inner())
    }
}
