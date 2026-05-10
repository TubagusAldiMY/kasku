use thiserror::Error;

#[derive(Debug, Error)]
pub enum DomainError {
    #[error("tenant schema tidak valid: {0}")]
    InvalidTenantSchema(String),

    #[error("tenant schema tidak cocok dengan user: expected {expected}, got {actual}")]
    TenantMismatch { expected: String, actual: String },

    #[error("tipe entitas tidak didukung: {0}")]
    UnsupportedEntityType(String),

    #[error("operasi tidak didukung: {0}")]
    UnsupportedOperation(String),

    #[error("header autentikasi tidak ditemukan")]
    Unauthorized,

    #[error("kesalahan database: {0}")]
    DatabaseError(String),

    #[error("kesalahan internal: {0}")]
    Internal(String),
}
