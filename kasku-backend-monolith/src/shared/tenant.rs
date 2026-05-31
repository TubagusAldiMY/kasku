use once_cell::sync::Lazy;
use regex::Regex;
use crate::app_error::AppError;

static TENANT_SCHEMA_REGEX: Lazy<Regex> =
    Lazy::new(|| Regex::new(r"^tenant_[0-9a-f_]{32,36}$").unwrap());

/// Validates tenant schema name format and returns it if valid.
/// Must be called before any schema name interpolation in SQL queries.
pub fn validate_tenant_schema(schema: &str) -> Result<&str, AppError> {
    if TENANT_SCHEMA_REGEX.is_match(schema) {
        Ok(schema)
    } else {
        Err(AppError::Forbidden)
    }
}

/// Derives tenant schema name from user_id UUID string.
pub fn user_id_to_schema(user_id: &str) -> String {
    format!("tenant_{}", user_id.replace('-', "_"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn valid_schema_passes() {
        let schema = "tenant_550e8400_e29b_41d4_a716_446655440000";
        assert!(validate_tenant_schema(schema).is_ok());
    }

    #[test]
    fn invalid_schema_rejected() {
        assert!(validate_tenant_schema("public").is_err());
        assert!(validate_tenant_schema("tenant_'; DROP TABLE").is_err());
        assert!(validate_tenant_schema("tenant_abc").is_err());
    }

    #[test]
    fn user_id_to_schema_correct() {
        let schema = user_id_to_schema("550e8400-e29b-41d4-a716-446655440000");
        assert_eq!(schema, "tenant_550e8400_e29b_41d4_a716_446655440000");
    }
}
