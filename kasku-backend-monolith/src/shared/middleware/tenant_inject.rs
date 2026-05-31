use axum::{extract::{Request, State}, middleware::Next, response::Response};
use std::sync::Arc;

use crate::app_error::AppError;
use crate::app_state::AppState;
use crate::shared::middleware::auth::AuthClaims;
use crate::shared::tenant::validate_tenant_schema;

#[derive(Debug, Clone)]
pub struct TenantContext {
    pub schema: String,
}

pub async fn tenant_inject_middleware(
    mut req: Request,
    next: Next,
) -> Result<Response, AppError> {
    let schema = {
        let claims = req.extensions().get::<AuthClaims>().cloned()
            .ok_or_else(|| AppError::Unauthorized("auth claims not found".into()))?;
        claims.tenant_schema.clone()
    };

    // Validate before storing — prevents injection if JWT was somehow tampered
    validate_tenant_schema(&schema)?;
    req.extensions_mut().insert(TenantContext { schema });
    Ok(next.run(req).await)
}
