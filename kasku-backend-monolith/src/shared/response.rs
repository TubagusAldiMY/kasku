use axum::{http::StatusCode, response::{IntoResponse, Response}, Json};
use serde::Serialize;
use serde_json::{json, Value};

#[derive(Debug, Serialize)]
pub struct ApiResponse<T: Serialize> {
    pub success: bool,
    pub data: T,
}

impl<T: Serialize> ApiResponse<T> {
    pub fn ok(data: T) -> Self {
        Self { success: true, data }
    }

    pub fn success(data: T) -> Self {
        Self { success: true, data }
    }
}

impl<T: Serialize> IntoResponse for ApiResponse<T> {
    fn into_response(self) -> Response {
        (StatusCode::OK, Json(self)).into_response()
    }
}

pub fn created<T: Serialize>(data: T) -> Response {
    (StatusCode::CREATED, Json(ApiResponse::ok(data))).into_response()
}

pub fn no_content() -> Response {
    StatusCode::NO_CONTENT.into_response()
}

pub fn paginated<T: Serialize>(items: Vec<T>, total: i64, page: i64, page_size: i64) -> Response {
    let body = json!({
        "success": true,
        "data": items,
        "meta": {
            "total": total,
            "page": page,
            "page_size": page_size,
            "total_pages": (total as f64 / page_size as f64).ceil() as i64
        }
    });
    (StatusCode::OK, Json(body)).into_response()
}
