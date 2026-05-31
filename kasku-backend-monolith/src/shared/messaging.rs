use async_trait::async_trait;
use lapin::{
    options::{BasicPublishOptions, ExchangeDeclareOptions},
    types::FieldTable,
    BasicProperties, Channel, Connection, ConnectionProperties, ExchangeKind,
};
use tracing::{error, info};

pub const EXCHANGE_NAME: &str = "kasku.events";

#[async_trait]
pub trait EventPublisher: Send + Sync {
    async fn publish_raw(&self, routing_key: &str, body: &[u8]) -> anyhow::Result<()>;
}

pub struct RabbitMQPublisher {
    channel: Channel,
}

impl RabbitMQPublisher {
    pub async fn new(amqp_url: &str) -> anyhow::Result<Self> {
        let conn = Connection::connect(amqp_url, ConnectionProperties::default()).await?;
        let channel = conn.create_channel().await?;

        channel
            .exchange_declare(
                EXCHANGE_NAME,
                ExchangeKind::Topic,
                ExchangeDeclareOptions {
                    durable: true,
                    ..Default::default()
                },
                FieldTable::default(),
            )
            .await?;

        info!("RabbitMQ publisher connected to exchange '{}'", EXCHANGE_NAME);
        Ok(Self { channel })
    }
}

#[async_trait]
impl EventPublisher for RabbitMQPublisher {
    async fn publish_raw(&self, routing_key: &str, body: &[u8]) -> anyhow::Result<()> {
        self.channel
            .basic_publish(
                EXCHANGE_NAME,
                routing_key,
                BasicPublishOptions::default(),
                body,
                BasicProperties::default().with_delivery_mode(2), // persistent
            )
            .await?
            .await?;
        Ok(())
    }
}
