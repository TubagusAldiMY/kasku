use tonic::codec::ProstCodec;
use tonic::transport::Channel;

use crate::proto::transaction::{
    ListTransactionsRequest, ListTransactionsResponse, UpsertTransactionsRequest,
    UpsertTransactionsResponse,
};

const UPSERT_PATH: &str = "/transaction.v1.TransactionInternal/UpsertTransactions";
const LIST_PATH: &str = "/transaction.v1.TransactionInternal/ListTransactions";

#[derive(Clone)]
pub struct TransactionInternalClient {
    channel: Channel,
}

impl TransactionInternalClient {
    pub fn new(channel: Channel) -> Self {
        Self { channel }
    }

    pub async fn upsert_transactions(
        &self,
        request: UpsertTransactionsRequest,
    ) -> Result<UpsertTransactionsResponse, tonic::Status> {
        let mut grpc = tonic::client::Grpc::new(self.channel.clone());
        grpc.ready().await.map_err(|e| {
            tonic::Status::unavailable(format!("transaction-service gRPC not ready: {e}"))
        })?;
        let codec = ProstCodec::<UpsertTransactionsRequest, UpsertTransactionsResponse>::default();
        let path = UPSERT_PATH
            .parse::<http::uri::PathAndQuery>()
            .map_err(|_| tonic::Status::internal("invalid gRPC path constant"))?;
        grpc.unary(tonic::Request::new(request), path, codec)
            .await
            .map(|r| r.into_inner())
    }

    pub async fn list_transactions(
        &self,
        request: ListTransactionsRequest,
    ) -> Result<ListTransactionsResponse, tonic::Status> {
        let mut grpc = tonic::client::Grpc::new(self.channel.clone());
        grpc.ready().await.map_err(|e| {
            tonic::Status::unavailable(format!("transaction-service gRPC not ready: {e}"))
        })?;
        let codec = ProstCodec::<ListTransactionsRequest, ListTransactionsResponse>::default();
        let path = LIST_PATH
            .parse::<http::uri::PathAndQuery>()
            .map_err(|_| tonic::Status::internal("invalid gRPC path constant"))?;
        grpc.unary(tonic::Request::new(request), path, codec)
            .await
            .map(|r| r.into_inner())
    }
}
