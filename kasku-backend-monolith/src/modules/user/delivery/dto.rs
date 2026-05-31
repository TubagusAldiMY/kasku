use serde::Serialize;

#[derive(Debug, Serialize)]
pub struct ProfileResponse {
    pub user_id: String,
    pub email: String,
    pub username: String,
    pub display_name: Option<String>,
}
