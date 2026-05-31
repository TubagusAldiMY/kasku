use thiserror::Error;

#[derive(Debug, Error)]
pub enum AuthError {
    #[error("email sudah terdaftar")]
    EmailAlreadyExists,

    #[error("username sudah digunakan")]
    UsernameAlreadyExists,

    #[error("email atau password salah")]
    InvalidCredentials,

    #[error("akun belum diverifikasi")]
    AccountNotVerified,

    #[error("akun terkunci hingga {0}")]
    AccountLocked(String),

    #[error("token tidak valid atau sudah kadaluarsa")]
    InvalidToken,

    #[error("token sudah digunakan")]
    TokenAlreadyUsed,

    #[error("validasi gagal: {0}")]
    Validation(String),

    #[error("password terlalu pendek (minimal 8 karakter)")]
    PasswordTooShort,

    #[error("password harus mengandung huruf besar, huruf kecil, dan angka")]
    PasswordTooWeak,

    #[error("user tidak ditemukan")]
    UserNotFound,

    #[error("internal error: {0}")]
    Internal(#[from] anyhow::Error),
}

impl From<sqlx::Error> for AuthError {
    fn from(e: sqlx::Error) -> Self {
        AuthError::Internal(anyhow::anyhow!(e))
    }
}
