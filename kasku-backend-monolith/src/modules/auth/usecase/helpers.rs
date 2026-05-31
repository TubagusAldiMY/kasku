use argon2::{
    password_hash::{rand_core::OsRng, PasswordHash, PasswordHasher, PasswordVerifier, SaltString},
    Argon2, Params,
};
use sha2::{Digest, Sha256};

#[derive(Clone)]
pub struct Argon2Config {
    pub time: u32,
    pub memory_kb: u32,
    pub threads: u32,
    pub key_length: u32,
}

/// Hash password using Argon2id.
/// Produces PHC string format: $argon2id$v=19$m=...,t=...,p=...$<salt>$<hash>
/// Compatible with Go's golang.org/x/crypto/argon2.IDKey() + manual PHC formatting.
pub fn hash_password(password: &str, cfg: &Argon2Config) -> anyhow::Result<String> {
    let params = Params::new(cfg.memory_kb, cfg.time, cfg.threads, Some(cfg.key_length as usize))
        .map_err(|e| anyhow::anyhow!("argon2 params error: {}", e))?;

    let argon2 = Argon2::new(argon2::Algorithm::Argon2id, argon2::Version::V0x13, params);
    let salt = SaltString::generate(&mut OsRng);

    let hash = argon2
        .hash_password(password.as_bytes(), &salt)
        .map_err(|e| anyhow::anyhow!("argon2 hash error: {}", e))?;

    Ok(hash.to_string())
}

/// Verify password against stored PHC hash.
pub fn verify_password(password: &str, hash_str: &str) -> anyhow::Result<bool> {
    let parsed = PasswordHash::new(hash_str)
        .map_err(|e| anyhow::anyhow!("invalid password hash: {}", e))?;

    let argon2 = Argon2::default();
    match argon2.verify_password(password.as_bytes(), &parsed) {
        Ok(()) => Ok(true),
        Err(argon2::password_hash::Error::Password) => Ok(false),
        Err(e) => Err(anyhow::anyhow!("password verify error: {}", e)),
    }
}

/// Generate 32 random bytes as hex string (64 chars), plus its SHA-256 hash.
/// rawToken is sent to user; tokenHash is stored in DB.
pub fn generate_secure_token() -> anyhow::Result<(String, String)> {
    use rand::RngCore;
    let mut raw = [0u8; 32];
    rand::thread_rng().fill_bytes(&mut raw);
    let raw_token = hex::encode(raw);
    let token_hash = sha256_hex(&raw_token);
    Ok((raw_token, token_hash))
}

pub fn sha256_hex(input: &str) -> String {
    let mut hasher = Sha256::new();
    hasher.update(input.as_bytes());
    hex::encode(hasher.finalize())
}

pub fn mask_email(email: &str) -> String {
    if let Some(at) = email.find('@') {
        let local = &email[..at];
        let domain = &email[at..];
        if local.len() <= 1 {
            format!("*{}", domain)
        } else {
            format!("{}***{}", &local[..1], domain)
        }
    } else {
        "***".to_string()
    }
}
