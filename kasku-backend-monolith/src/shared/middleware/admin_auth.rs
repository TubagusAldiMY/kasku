use std::sync::Arc;
use axum::{extract::{Request, State}, middleware::Next, response::Response};
use jsonwebtoken::{decode, Algorithm, DecodingKey, Validation};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::app_error::AppError;
use crate::app_state::AppState;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct AdminClaims {
    pub sub: String,
    pub username: String,
    pub role: String,
    pub exp: i64,
    pub iat: i64,
}

#[derive(Debug, Clone)]
pub struct AuthAdmin {
    pub admin_id: Uuid,
    pub username: String,
    pub role: String,
}

pub async fn admin_auth_middleware(
    State(state): State<Arc<AppState>>,
    mut req: Request,
    next: Next,
) -> Result<Response, AppError> {
    let token = extract_bearer_token(req.headers())?;
    let claims = verify_admin_token(&token, state.admin_jwt_secret.as_bytes())?;
    req.extensions_mut().insert(claims);
    Ok(next.run(req).await)
}

pub fn verify_admin_token(token_str: &str, secret: &[u8]) -> Result<AuthAdmin, AppError> {
    let decoding_key = DecodingKey::from_secret(secret);
    let mut validation = Validation::new(Algorithm::HS256);
    validation.validate_exp = true;

    let token_data = decode::<AdminClaims>(token_str, &decoding_key, &validation)
        .map_err(|e| AppError::Unauthorized(format!("admin token tidak valid: {}", e)))?;

    let claims = token_data.claims;
    let admin_id = Uuid::parse_str(&claims.sub)
        .map_err(|_| AppError::Unauthorized("admin token sub bukan UUID valid".into()))?;

    Ok(AuthAdmin {
        admin_id,
        username: claims.username,
        role: claims.role,
    })
}

fn extract_bearer_token(headers: &axum::http::HeaderMap) -> Result<String, AppError> {
    let auth_header = headers
        .get(axum::http::header::AUTHORIZATION)
        .and_then(|v| v.to_str().ok())
        .ok_or_else(|| AppError::Unauthorized("authorization header tidak ditemukan".into()))?;

    let parts: Vec<&str> = auth_header.splitn(2, ' ').collect();
    if parts.len() != 2 || !parts[0].eq_ignore_ascii_case("Bearer") {
        return Err(AppError::Unauthorized("format Authorization header tidak valid".into()));
    }

    let token = parts[1].trim().to_string();
    if token.is_empty() {
        return Err(AppError::Unauthorized("bearer token kosong".into()));
    }
    Ok(token)
}
