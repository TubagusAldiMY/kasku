use thiserror::Error;

#[derive(Debug, Error)]
pub enum PriceError {
    #[error("harga tidak ditemukan untuk symbol: {0}")]
    PriceNotFound(String),

    #[error("sumber harga tidak didukung: {0}")]
    UnsupportedSource(String),

    #[error("gagal mengambil harga dari API eksternal: {0}")]
    ExternalApiFailed(String),

    #[error("domain SSRF tidak diizinkan: {0}")]
    SsrfBlocked(String),

    #[error("kesalahan database: {0}")]
    DatabaseError(String),

    #[error("kesalahan internal: {0}")]
    Internal(String),
}
