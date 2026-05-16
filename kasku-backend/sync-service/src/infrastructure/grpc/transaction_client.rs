use std::time::Duration;

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
    timeout: Duration,
}

impl TransactionInternalClient {
    pub fn new(channel: Channel, timeout: Duration) -> Self {
        Self { channel, timeout }
    }

    pub async fn upsert_transactions(
        &self,
        request: UpsertTransactionsRequest,
    ) -> Result<UpsertTransactionsResponse, tonic::Status> {
        let mut grpc = tonic::client::Grpc::new(self.channel.clone());
        tokio::time::timeout(self.timeout, grpc.ready())
            .await
            .map_err(|_| tonic::Status::deadline_exceeded("upstream timeout"))?
            .map_err(|_| tonic::Status::unavailable("upstream unavailable"))?;
        let codec = ProstCodec::<UpsertTransactionsRequest, UpsertTransactionsResponse>::default();
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

    pub async fn list_transactions(
        &self,
        request: ListTransactionsRequest,
    ) -> Result<ListTransactionsResponse, tonic::Status> {
        let mut grpc = tonic::client::Grpc::new(self.channel.clone());
        tokio::time::timeout(self.timeout, grpc.ready())
            .await
            .map_err(|_| tonic::Status::deadline_exceeded("upstream timeout"))?
            .map_err(|_| tonic::Status::unavailable("upstream unavailable"))?;
        let codec = ProstCodec::<ListTransactionsRequest, ListTransactionsResponse>::default();
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
