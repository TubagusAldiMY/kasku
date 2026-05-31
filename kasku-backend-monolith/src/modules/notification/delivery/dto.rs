use serde::{Deserialize, Serialize};
use crate::modules::notification::domain::entity::NotificationPreference;

#[derive(Debug, Serialize)]
pub struct PreferenceResponse {
    pub email_enabled: bool,
    pub payment_alerts: bool,
    pub subscription_alerts: bool,
    pub security_alerts: bool,
}

impl From<NotificationPreference> for PreferenceResponse {
    fn from(p: NotificationPreference) -> Self {
        Self {
            email_enabled: p.email_enabled,
            payment_alerts: p.payment_alerts,
            subscription_alerts: p.subscription_alerts,
            security_alerts: p.security_alerts,
        }
    }
}

#[derive(Debug, Deserialize)]
pub struct UpdatePreferenceRequest {
    pub email_enabled: Option<bool>,
    pub payment_alerts: Option<bool>,
    pub subscription_alerts: Option<bool>,
    pub security_alerts: Option<bool>,
}
