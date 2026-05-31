use std::sync::Arc;
use axum::{
    extract::{Extension, Query, State},
    http::HeaderMap,
    Json,
    response::{IntoResponse, Response},
};

use crate::app_error::AppError;
use crate::app_state::AppState;
use crate::shared::middleware::auth::AuthClaims;
use crate::shared::middleware::rate_limit::*;
use crate::shared::response::{created, no_content, ApiResponse};

use super::dto::*;
use crate::modules::auth::usecase::register::RegisterInput;
use crate::modules::auth::usecase::login::LoginInput;
use crate::modules::auth::usecase::refresh_token::RefreshTokenInput;
use crate::modules::auth::domain::error::AuthError;

fn auth_err_to_app(e: AuthError) -> AppError {
    match e {
        AuthError::EmailAlreadyExists | AuthError::UsernameAlreadyExists => {
            AppError::Conflict(e.to_string())
        }
        AuthError::InvalidCredentials | AuthError::InvalidToken => {
            AppError::Unauthorized(e.to_string())
        }
        AuthError::AccountNotVerified => AppError::Forbidden,
        AuthError::AccountLocked(t) => AppError::Unauthorized(format!("akun terkunci hingga {}", t)),
        AuthError::TokenAlreadyUsed => AppError::Unprocessable(e.to_string()),
        AuthError::Validation(_) | AuthError::PasswordTooShort | AuthError::PasswordTooWeak => {
            AppError::Validation(e.to_string())
        }
        AuthError::UserNotFound => AppError::NotFound,
        AuthError::Internal(inner) => AppError::Internal(inner),
    }
}

fn get_client_ip(headers: &HeaderMap) -> String {
    headers
        .get("X-Forwarded-For")
        .and_then(|v| v.to_str().ok())
        .and_then(|s| s.split(',').next())
        .unwrap_or("unknown")
        .trim()
        .to_string()
}

pub async fn register(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(req): Json<RegisterRequest>,
) -> Result<Response, AppError> {
    let ip = get_client_ip(&headers);
    let rl = check_register(&state.redis_pool, &ip).await
        .map_err(|e| AppError::Internal(e))?;
    rate_limited_if_denied(rl)?;

    let out = state.auth_uc.register.execute(RegisterInput {
        email: req.email,
        username: req.username,
        password: req.password,
    }).await.map_err(auth_err_to_app)?;

    Ok(created(RegisterResponse {
        user_id: out.user_id.to_string(),
        email: out.email,
        username: out.username,
        message: "Registrasi berhasil. Silakan verifikasi email Anda.".into(),
    }))
}

pub async fn login(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(req): Json<LoginRequest>,
) -> Result<Response, AppError> {
    let ip = get_client_ip(&headers);

    let rl_ip = check_login_ip(&state.redis_pool, &ip).await
        .map_err(|e| AppError::Internal(e))?;
    rate_limited_if_denied(rl_ip)?;

    let rl_email = check_login_email(&state.redis_pool, &req.email).await
        .map_err(|e| AppError::Internal(e))?;
    rate_limited_if_denied(rl_email)?;

    let user_agent = headers.get("User-Agent").and_then(|v| v.to_str().ok()).map(|s| s.to_string());

    let out = state.auth_uc.login.execute(LoginInput {
        email: req.email,
        password: req.password,
        user_agent,
        ip_address: Some(ip),
    }).await.map_err(auth_err_to_app)?;

    Ok(ApiResponse::ok(LoginResponse {
        access_token: out.access_token,
        refresh_token: out.refresh_token,
        user_id: out.user_id.to_string(),
        email: out.email,
        username: out.username,
        token_type: "Bearer".into(),
    }).into_response())
}

pub async fn refresh(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(req): Json<RefreshRequest>,
) -> Result<Response, AppError> {
    let user_agent = headers.get("User-Agent").and_then(|v| v.to_str().ok()).map(|s| s.to_string());
    let ip = get_client_ip(&headers);

    let out = state.auth_uc.refresh_token.execute(RefreshTokenInput {
        raw_refresh_token: req.refresh_token,
        user_agent,
        ip_address: Some(ip),
    }).await.map_err(auth_err_to_app)?;

    Ok(ApiResponse::ok(RefreshResponse {
        access_token: out.access_token,
        refresh_token: out.refresh_token,
        token_type: "Bearer".into(),
    }).into_response())
}

pub async fn logout(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Json(req): Json<LogoutRequest>,
) -> Result<Response, AppError> {
    state.auth_uc.logout.execute(&claims, req.refresh_token)
        .await
        .map_err(auth_err_to_app)?;
    Ok(no_content())
}

pub async fn verify_email(
    State(state): State<Arc<AppState>>,
    Query(q): Query<VerifyEmailQuery>,
) -> Result<Response, AppError> {
    state.auth_uc.verify_email.execute(&q.token)
        .await
        .map_err(auth_err_to_app)?;

    Ok(ApiResponse::ok(serde_json::json!({"message": "Email berhasil diverifikasi."})).into_response())
}

pub async fn resend_verification(
    State(state): State<Arc<AppState>>,
    Json(req): Json<ResendVerificationRequest>,
) -> Result<Response, AppError> {
    state.auth_uc.resend_verification.execute(&req.email)
        .await
        .map_err(auth_err_to_app)?;

    Ok(ApiResponse::ok(serde_json::json!({"message": "Email verifikasi telah dikirim ulang."})).into_response())
}

pub async fn forgot_password(
    State(state): State<Arc<AppState>>,
    Json(req): Json<ForgotPasswordRequest>,
) -> Result<Response, AppError> {
    let rl = check_forgot_password(&state.redis_pool, &req.email).await
        .map_err(|e| AppError::Internal(e))?;
    rate_limited_if_denied(rl)?;

    state.auth_uc.forgot_password.execute(&req.email)
        .await
        .map_err(auth_err_to_app)?;

    Ok(ApiResponse::ok(serde_json::json!({"message": "Jika email terdaftar, link reset password telah dikirim."})).into_response())
}

pub async fn reset_password(
    State(state): State<Arc<AppState>>,
    Json(req): Json<ResetPasswordRequest>,
) -> Result<Response, AppError> {
    state.auth_uc.reset_password.execute(&req.token, &req.new_password)
        .await
        .map_err(auth_err_to_app)?;

    Ok(ApiResponse::ok(serde_json::json!({"message": "Password berhasil direset."})).into_response())
}

pub async fn change_password(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Json(req): Json<ChangePasswordRequest>,
) -> Result<Response, AppError> {
    state.auth_uc.change_password.execute(claims.user_id, &req.current_password, &req.new_password)
        .await
        .map_err(auth_err_to_app)?;

    Ok(ApiResponse::ok(serde_json::json!({"message": "Password berhasil diubah."})).into_response())
}
