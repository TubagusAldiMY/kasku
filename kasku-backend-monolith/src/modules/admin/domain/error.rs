use thiserror::Error;

#[derive(Debug, Error)]
pub enum AdminError {
    #[error("invalid credentials")]
    InvalidCredentials,
    #[error("admin account inactive")]
    AdminInactive,
    #[error("admin not found")]
    AdminNotFound,
    #[error("user not found")]
    UserNotFound,
    #[error("subscription not found")]
    SubscriptionNotFound,
    #[error("plan not found")]
    PlanNotFound,
    #[error("invalid token")]
    InvalidToken,
    #[error("unauthorized")]
    Unauthorized,
    #[error("forbidden")]
    Forbidden,
    #[error("validation: {0}")]
    Validation(String),
    #[error("internal: {0}")]
    Internal(String),
}

impl From<sqlx::Error> for AdminError {
    fn from(e: sqlx::Error) -> Self {
        AdminError::Internal(e.to_string())
    }
}
