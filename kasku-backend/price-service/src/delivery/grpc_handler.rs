use std::sync::Arc;
use tonic::{Request, Response, Status};

use crate::proto_gen::price::v1::{
    price_service_server::PriceService, GetPriceRequest, GetPriceResponse, GetPricesRequest,
    GetPricesResponse,
};
use crate::usecase::get_price::GetPriceUseCase;

/// gRPC service implementation for PriceService.
pub struct PriceGrpcHandler {
    get_price_uc: Arc<GetPriceUseCase>,
}

impl PriceGrpcHandler {
    pub fn new(get_price_uc: Arc<GetPriceUseCase>) -> Self {
        Self { get_price_uc }
    }
}

#[tonic::async_trait]
impl PriceService for PriceGrpcHandler {
    async fn get_price(
        &self,
        request: Request<GetPriceRequest>,
    ) -> Result<Response<GetPriceResponse>, Status> {
        let req = request.into_inner();

        let result = self
            .get_price_uc
            .execute(&req.symbol, &req.source)
            .await
            .map_err(|e| match e {
                crate::domain::error::DomainError::PriceNotFound(s) => {
                    Status::not_found(format!("harga tidak ditemukan: {}", s))
                }
                _ => Status::internal(e.to_string()),
            })?;

        Ok(Response::new(GetPriceResponse {
            symbol: result.symbol,
            price_idr: result.price_idr,
            price_usd: result.price_usd,
            is_fresh: result.is_fresh,
            updated_at: result.updated_at.to_rfc3339(),
        }))
    }

    async fn get_prices(
        &self,
        request: Request<GetPricesRequest>,
    ) -> Result<Response<GetPricesResponse>, Status> {
        let req = request.into_inner();
        let mut prices = Vec::with_capacity(req.symbols.len());

        for symbol in &req.symbols {
            match self.get_price_uc.execute(symbol, "").await {
                Ok(result) => prices.push(GetPriceResponse {
                    symbol: result.symbol,
                    price_idr: result.price_idr,
                    price_usd: result.price_usd,
                    is_fresh: result.is_fresh,
                    updated_at: result.updated_at.to_rfc3339(),
                }),
                Err(e) => {
                    tracing::warn!(symbol = symbol.as_str(), error = %e, "gagal ambil harga untuk symbol");
                    // Skip symbol yang gagal — partial success is acceptable
                }
            }
        }

        Ok(Response::new(GetPricesResponse { prices }))
    }
}
