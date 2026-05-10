use regex::Regex;
use std::sync::LazyLock;

use crate::domain::error::DomainError;

/// Regex pattern for valid tenant schema names.
/// Format: tenant_{uuid_with_underscores} e.g., tenant_550e8400_e29b_41d4_a716_446655440000
static TENANT_SCHEMA_REGEX: LazyLock<Regex> =
    LazyLock::new(|| Regex::new(r"^tenant_[0-9a-f_]{32,36}$").expect("invalid tenant regex"));

/// Validate that a tenant schema name matches the expected pattern.
/// Prevents SQL injection via schema name.
pub fn validate_tenant_schema(schema: &str) -> Result<(), DomainError> {
    if !TENANT_SCHEMA_REGEX.is_match(schema) {
        return Err(DomainError::InvalidTenantSchema(schema.to_string()));
    }
    Ok(())
}

/// Derive the expected tenant schema from a user ID.
/// user_id UUID → tenant_{uuid_with_hyphens_replaced_by_underscores}
pub fn user_id_to_tenant_schema(user_id: &str) -> String {
    format!("tenant_{}", user_id.replace('-', "_"))
}

/// Verify that the tenant schema matches the user ID.
/// Security check: prevents accessing another user's tenant schema.
pub fn verify_tenant_ownership(user_id: &str, tenant_schema: &str) -> Result<(), DomainError> {
    let expected = user_id_to_tenant_schema(user_id);
    if expected != tenant_schema {
        return Err(DomainError::TenantMismatch {
            expected,
            actual: tenant_schema.to_string(),
        });
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_valid_tenant_schema() {
        assert!(validate_tenant_schema("tenant_550e8400_e29b_41d4_a716_446655440000").is_ok());
    }

    #[test]
    fn test_invalid_tenant_schema() {
        assert!(validate_tenant_schema("public").is_err());
        assert!(validate_tenant_schema("tenant_; DROP TABLE users;--").is_err());
        assert!(validate_tenant_schema("").is_err());
    }

    #[test]
    fn test_user_id_to_tenant_schema() {
        assert_eq!(
            user_id_to_tenant_schema("550e8400-e29b-41d4-a716-446655440000"),
            "tenant_550e8400_e29b_41d4_a716_446655440000"
        );
    }

    #[test]
    fn test_verify_tenant_ownership() {
        assert!(verify_tenant_ownership(
            "550e8400-e29b-41d4-a716-446655440000",
            "tenant_550e8400_e29b_41d4_a716_446655440000"
        ).is_ok());

        assert!(verify_tenant_ownership(
            "550e8400-e29b-41d4-a716-446655440000",
            "tenant_other_user_schema"
        ).is_err());
    }
}
