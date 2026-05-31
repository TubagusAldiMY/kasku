use std::sync::Arc;
use uuid::Uuid;
use tracing::warn;

use crate::modules::notification::domain::{
    entity::NotificationPreference,
    error::NotificationError,
    repository::PreferenceRepository,
};
use crate::modules::notification::infrastructure::SmtpEmailSender;
use crate::modules::notification::infrastructure::templates;

pub struct NotificationHandler {
    repo: Arc<dyn PreferenceRepository>,
    smtp: Arc<SmtpEmailSender>,
    frontend_url: String,
}

impl NotificationHandler {
    pub fn new(
        repo: Arc<dyn PreferenceRepository>,
        smtp: Arc<SmtpEmailSender>,
        frontend_url: String,
    ) -> Self {
        Self { repo, smtp, frontend_url }
    }

    async fn is_email_enabled(&self, user_id_str: &str) -> bool {
        let Ok(uid) = user_id_str.parse::<Uuid>() else { return true; };
        match self.repo.find_by_user_id(uid).await {
            Ok(Some(pref)) => pref.email_enabled,
            _ => true,
        }
    }

    async fn send(&self, email: &str, subject: &str, template: &str, ctx: &tera::Context) {
        match templates::render(template, ctx) {
            Ok(html) => {
                if let Err(e) = self.smtp.send_html(email, subject, &html).await {
                    warn!(email=%email, template=%template, error=%e, "gagal kirim email notifikasi");
                }
            }
            Err(e) => warn!(template=%template, error=%e, "gagal render template email"),
        }
    }

    pub async fn handle_user_registered(&self, user_id: &str, email: &str, username: &str, token: &str) {
        if !self.is_email_enabled(user_id).await { return; }
        let url = format!("{}/verify-email?token={}", self.frontend_url, token);
        let mut ctx = tera::Context::new();
        ctx.insert("username", username);
        ctx.insert("verification_url", &url);
        self.send(email, "Verifikasi Email KasKu Anda", "email_verification.html", &ctx).await;
    }

    pub async fn handle_email_verification_resent(&self, user_id: &str, email: &str, token: &str) {
        if !self.is_email_enabled(user_id).await { return; }
        let url = format!("{}/verify-email?token={}", self.frontend_url, token);
        let mut ctx = tera::Context::new();
        ctx.insert("verification_url", &url);
        self.send(email, "Kirim Ulang Verifikasi Email KasKu", "email_verification_resent.html", &ctx).await;
    }

    pub async fn handle_password_reset_requested(&self, user_id: &str, email: &str, token: &str) {
        if !self.is_email_enabled(user_id).await { return; }
        let url = format!("{}/reset-password?token={}", self.frontend_url, token);
        let mut ctx = tera::Context::new();
        ctx.insert("email", email);
        ctx.insert("reset_url", &url);
        self.send(email, "Reset Password KasKu", "password_reset.html", &ctx).await;
    }

    pub async fn handle_payment_succeeded(&self, user_id: &str, email: &str, order_id: &str, amount_idr: i64, plan_name: &str) {
        if !self.is_email_enabled(user_id).await { return; }
        let mut ctx = tera::Context::new();
        ctx.insert("email", email);
        ctx.insert("order_id", order_id);
        ctx.insert("amount_idr", &format_idr(amount_idr));
        ctx.insert("plan_name", plan_name);
        self.send(email, "Pembayaran KasKu Berhasil", "payment_success.html", &ctx).await;
    }

    pub async fn handle_payment_failed(&self, user_id: &str, email: &str, order_id: &str, reason: &str) {
        if !self.is_email_enabled(user_id).await { return; }
        let mut ctx = tera::Context::new();
        ctx.insert("email", email);
        ctx.insert("order_id", order_id);
        ctx.insert("reason", reason);
        self.send(email, "Pembayaran KasKu Gagal", "payment_failed.html", &ctx).await;
    }

    pub async fn handle_subscription_expiring(&self, user_id: &str, email: &str, plan_name: &str, expires_at: &str) {
        if !self.is_email_enabled(user_id).await { return; }
        let mut ctx = tera::Context::new();
        ctx.insert("email", email);
        ctx.insert("plan_name", plan_name);
        ctx.insert("expires_at", expires_at);
        ctx.insert("frontend_url", &self.frontend_url);
        self.send(email, "Subscription KasKu Akan Berakhir", "subscription_expiring.html", &ctx).await;
    }

    pub async fn handle_subscription_expired(&self, user_id: &str, email: &str, plan_name: &str) {
        if !self.is_email_enabled(user_id).await { return; }
        let mut ctx = tera::Context::new();
        ctx.insert("email", email);
        ctx.insert("plan_name", plan_name);
        ctx.insert("frontend_url", &self.frontend_url);
        self.send(email, "Subscription KasKu Anda Telah Berakhir", "subscription_expired.html", &ctx).await;
    }

    pub async fn handle_subscription_cancelled(&self, user_id: &str, email: &str, plan_name: &str, cancelled_at: &str) {
        if !self.is_email_enabled(user_id).await { return; }
        let mut ctx = tera::Context::new();
        ctx.insert("email", email);
        ctx.insert("plan_name", plan_name);
        ctx.insert("cancelled_at", cancelled_at);
        self.send(email, "Subscription KasKu Dibatalkan", "subscription_cancelled.html", &ctx).await;
    }
}

fn format_idr(amount: i64) -> String {
    format!("{:.*}", 0, amount as f64)
}

/// Wrapper exposed on AppState for preference management API.
pub struct NotificationUseCases {
    pub repo: Arc<dyn PreferenceRepository>,
}

impl NotificationUseCases {
    pub fn new(repo: Arc<dyn PreferenceRepository>) -> Self { Self { repo } }

    pub async fn get_preference(&self, user_id: Uuid) -> Result<NotificationPreference, NotificationError> {
        Ok(self.repo.find_by_user_id(user_id).await?.unwrap_or_else(|| NotificationPreference {
            user_id,
            email_enabled: true,
            payment_alerts: true,
            subscription_alerts: true,
            security_alerts: true,
            created_at: chrono::Utc::now(),
            updated_at: chrono::Utc::now(),
        }))
    }

    pub async fn update_preference(
        &self, user_id: Uuid,
        email_enabled: Option<bool>,
        payment_alerts: Option<bool>,
        subscription_alerts: Option<bool>,
        security_alerts: Option<bool>,
    ) -> Result<NotificationPreference, NotificationError> {
        let mut pref = self.get_preference(user_id).await?;
        if let Some(v) = email_enabled { pref.email_enabled = v; }
        if let Some(v) = payment_alerts { pref.payment_alerts = v; }
        if let Some(v) = subscription_alerts { pref.subscription_alerts = v; }
        if let Some(v) = security_alerts { pref.security_alerts = v; }
        self.repo.upsert(&pref).await?;
        Ok(pref)
    }
}
