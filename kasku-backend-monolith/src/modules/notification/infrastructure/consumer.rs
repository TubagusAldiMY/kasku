use std::sync::Arc;
use lapin::{
    Connection, ConnectionProperties,
    options::*,
    types::{AMQPValue, FieldTable, LongInt, LongString},
    BasicProperties, ExchangeKind,
};
use tokio_util::sync::CancellationToken;
use tracing::{error, info, warn};
use futures_lite::stream::StreamExt;

use crate::modules::notification::usecase::NotificationHandler;

const EXCHANGE: &str = "kasku.events";
const QUEUE: &str = "kasku.notification-service";
const DLX: &str = "kasku.events.dlx";
const MAX_RETRY: i64 = 3;

pub struct NotificationConsumer {
    amqp_url: String,
    pub handler: Arc<NotificationHandler>,
}

impl NotificationConsumer {
    pub fn new(amqp_url: String, handler: Arc<NotificationHandler>) -> Self {
        Self { amqp_url, handler }
    }

    pub async fn run(self, cancel: CancellationToken) {
        loop {
            tokio::select! {
                _ = cancel.cancelled() => {
                    info!("notification consumer stopping");
                    return;
                }
                result = self.connect_and_consume(cancel.clone()) => {
                    if cancel.is_cancelled() { return; }
                    match result {
                        Ok(()) => return,
                        Err(e) => {
                            error!(error=%e, "notification consumer disconnected, reconnecting in 5s");
                            tokio::time::sleep(std::time::Duration::from_secs(5)).await;
                        }
                    }
                }
            }
        }
    }

    async fn connect_and_consume(&self, cancel: CancellationToken) -> anyhow::Result<()> {
        let conn = Connection::connect(&self.amqp_url, ConnectionProperties::default()).await?;
        let ch = conn.create_channel().await?;

        ch.exchange_declare(
            EXCHANGE, ExchangeKind::Topic,
            ExchangeDeclareOptions { durable: true, ..Default::default() },
            FieldTable::default(),
        ).await?;

        let mut queue_args = FieldTable::default();
        queue_args.insert("x-dead-letter-exchange".into(), AMQPValue::LongString(LongString::from(DLX)));
        queue_args.insert("x-message-ttl".into(), AMQPValue::LongInt(86400000));
        ch.queue_declare(
            QUEUE,
            QueueDeclareOptions { durable: true, ..Default::default() },
            queue_args,
        ).await?;

        let routing_keys = [
            "user.registered", "user.email_verification_resent", "user.password_reset_requested",
            "payment.succeeded", "payment.failed",
            "subscription.expiring", "subscription.expired", "subscription.cancelled",
        ];
        for key in &routing_keys {
            ch.queue_bind(QUEUE, EXCHANGE, key, QueueBindOptions::default(), FieldTable::default()).await?;
        }

        let mut consumer = ch.basic_consume(
            QUEUE, "kasku-monolith",
            BasicConsumeOptions::default(), FieldTable::default(),
        ).await?;

        info!("notification consumer started");

        loop {
            tokio::select! {
                _ = cancel.cancelled() => return Ok(()),
                delivery = consumer.next() => {
                    let Some(Ok(d)) = delivery else { return Ok(()); };
                    let routing_key = d.routing_key.as_str().to_string();

                    // Retry count from x-death header
                    let retry_count: i64 = d.properties.headers()
                        .as_ref()
                        .and_then(|h| h.inner().get("x-death"))
                        .and_then(|v| if let AMQPValue::FieldArray(arr) = v {
                            arr.as_slice().first().and_then(|entry| {
                                if let AMQPValue::FieldTable(table) = entry {
                                    table.inner().get("count").and_then(|c| {
                                        if let AMQPValue::LongInt(n) = c { Some(*n as i64) } else { None }
                                    })
                                } else { None }
                            })
                        } else { None })
                        .unwrap_or(0);

                    if retry_count >= MAX_RETRY {
                        warn!(routing_key=%routing_key, "message exceeded max retries, nacking to DLX");
                        let _ = d.nack(BasicNackOptions { requeue: false, ..Default::default() }).await;
                        continue;
                    }

                    let payload = &d.data;
                    let handler = &self.handler;

                    let ok = match routing_key.as_str() {
                        "user.registered" => {
                            #[derive(serde::Deserialize)] struct E { user_id: String, email: String, username: String, verification_token: String }
                            if let Ok(e) = serde_json::from_slice::<E>(payload) {
                                handler.handle_user_registered(&e.user_id, &e.email, &e.username, &e.verification_token).await; true
                            } else { false }
                        }
                        "user.email_verification_resent" => {
                            #[derive(serde::Deserialize)] struct E { user_id: String, email: String, verification_token: String }
                            if let Ok(e) = serde_json::from_slice::<E>(payload) {
                                handler.handle_email_verification_resent(&e.user_id, &e.email, &e.verification_token).await; true
                            } else { false }
                        }
                        "user.password_reset_requested" => {
                            #[derive(serde::Deserialize)] struct E { user_id: String, email: String, reset_token: String }
                            if let Ok(e) = serde_json::from_slice::<E>(payload) {
                                handler.handle_password_reset_requested(&e.user_id, &e.email, &e.reset_token).await; true
                            } else { false }
                        }
                        "payment.succeeded" => {
                            #[derive(serde::Deserialize)] struct E { user_id: String, email: String, order_id: String, amount_idr: i64, plan_name: String }
                            if let Ok(e) = serde_json::from_slice::<E>(payload) {
                                handler.handle_payment_succeeded(&e.user_id, &e.email, &e.order_id, e.amount_idr, &e.plan_name).await; true
                            } else { false }
                        }
                        "payment.failed" => {
                            #[derive(serde::Deserialize)] struct E { user_id: String, email: String, order_id: String, reason: String }
                            if let Ok(e) = serde_json::from_slice::<E>(payload) {
                                handler.handle_payment_failed(&e.user_id, &e.email, &e.order_id, &e.reason).await; true
                            } else { false }
                        }
                        "subscription.expiring" => {
                            #[derive(serde::Deserialize)] struct E { user_id: String, email: String, plan_name: String, expires_at: String }
                            if let Ok(e) = serde_json::from_slice::<E>(payload) {
                                handler.handle_subscription_expiring(&e.user_id, &e.email, &e.plan_name, &e.expires_at).await; true
                            } else { false }
                        }
                        "subscription.expired" => {
                            #[derive(serde::Deserialize)] struct E { user_id: String, email: String, plan_name: String }
                            if let Ok(e) = serde_json::from_slice::<E>(payload) {
                                handler.handle_subscription_expired(&e.user_id, &e.email, &e.plan_name).await; true
                            } else { false }
                        }
                        "subscription.cancelled" => {
                            #[derive(serde::Deserialize)] struct E { user_id: String, email: String, plan_name: String, cancelled_at: String }
                            if let Ok(e) = serde_json::from_slice::<E>(payload) {
                                handler.handle_subscription_cancelled(&e.user_id, &e.email, &e.plan_name, &e.cancelled_at).await; true
                            } else { false }
                        }
                        _ => {
                            warn!(routing_key=%routing_key, "unknown routing key, acking");
                            true
                        }
                    };

                    if ok {
                        let _ = d.ack(BasicAckOptions::default()).await;
                    } else {
                        warn!(routing_key=%routing_key, "failed to process message, nacking for retry");
                        let _ = d.nack(BasicNackOptions { requeue: true, ..Default::default() }).await;
                    }
                }
            }
        }
    }
}
