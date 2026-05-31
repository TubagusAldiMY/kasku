use thiserror::Error;

#[derive(Debug, Error)]
pub enum NotificationError {
    #[error("preferensi notifikasi tidak ditemukan")]
    NotFound,
    #[error("internal error: {0}")]
    Internal(String),
}

impl From<sqlx::Error> for NotificationError {
    fn from(e: sqlx::Error) -> Self {
        NotificationError::Internal(e.to_string())
    }
}
