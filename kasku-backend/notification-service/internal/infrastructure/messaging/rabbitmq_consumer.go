package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/notification-service/internal/domain/event"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

const (
	exchangeName       = "kasku.events"
	exchangeType       = "topic"
	queueName          = "kasku.notification-service"
	dlxExchange        = "kasku.events.dlx"
	messageTTLMillisec = 86400000
	maxRetryCount      = 3
)

// NotificationHandler mendefinisikan kontrak handler event notifikasi.
type NotificationHandler interface {
	HandleUserRegistered(ctx context.Context, e event.UserRegisteredEvent) error
	HandleEmailVerificationResent(ctx context.Context, e event.EmailVerificationResentEvent) error
	HandlePasswordResetRequested(ctx context.Context, e event.PasswordResetRequestedEvent) error
	HandlePaymentSucceeded(ctx context.Context, e event.PaymentSucceededEvent) error
	HandlePaymentFailed(ctx context.Context, e event.PaymentFailedEvent) error
	HandleSubscriptionExpiring(ctx context.Context, e event.SubscriptionExpiringEvent) error
	HandleSubscriptionExpired(ctx context.Context, e event.SubscriptionExpiredEvent) error
}

// RabbitMQConsumer mengkonsumsi notification events dari exchange kasku.events.
type RabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	log     zerolog.Logger
}

func NewRabbitMQConsumer(amqpURL string, log zerolog.Logger) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("gagal koneksi ke RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("gagal buat channel: %w", err)
	}

	if err := ch.ExchangeDeclare(exchangeName, exchangeType, true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("gagal declare exchange: %w", err)
	}

	_, err = ch.QueueDeclare(queueName, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange": dlxExchange,
		"x-message-ttl":          int32(messageTTLMillisec),
	})
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("gagal declare queue: %w", err)
	}

	routingKeys := []string{
		"user.registered",
		"user.email_verification_resent",
		"user.password_reset_requested",
		"payment.succeeded",
		"payment.failed",
		"subscription.expiring",
		"subscription.expired",
	}
	for _, key := range routingKeys {
		if err := ch.QueueBind(queueName, key, exchangeName, false, nil); err != nil {
			ch.Close()
			conn.Close()
			return nil, fmt.Errorf("gagal bind routing key %s: %w", key, err)
		}
	}

	return &RabbitMQConsumer{conn: conn, channel: ch, log: log}, nil
}

func (c *RabbitMQConsumer) StartConsuming(ctx context.Context, handler NotificationHandler) error {
	msgs, err := c.channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("gagal mulai consume: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				c.processMessage(ctx, msg, handler)
			}
		}
	}()
	return nil
}

func (c *RabbitMQConsumer) processMessage(ctx context.Context, msg amqp.Delivery, handler NotificationHandler) {
	retryCount := extractRetryCount(msg.Headers)

	var processErr error
	switch msg.RoutingKey {
	case "user.registered":
		var e event.UserRegisteredEvent
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			c.log.Error().Err(err).Msg("gagal unmarshal UserRegisteredEvent")
			_ = msg.Nack(false, false)
			return
		}
		processErr = handler.HandleUserRegistered(ctx, e)

	case "user.email_verification_resent":
		var e event.EmailVerificationResentEvent
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			c.log.Error().Err(err).Msg("gagal unmarshal EmailVerificationResentEvent")
			_ = msg.Nack(false, false)
			return
		}
		processErr = handler.HandleEmailVerificationResent(ctx, e)

	case "user.password_reset_requested":
		var e event.PasswordResetRequestedEvent
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			c.log.Error().Err(err).Msg("gagal unmarshal PasswordResetRequestedEvent")
			_ = msg.Nack(false, false)
			return
		}
		processErr = handler.HandlePasswordResetRequested(ctx, e)

	case "payment.succeeded":
		var e event.PaymentSucceededEvent
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			c.log.Error().Err(err).Msg("gagal unmarshal PaymentSucceededEvent")
			_ = msg.Nack(false, false)
			return
		}
		processErr = handler.HandlePaymentSucceeded(ctx, e)

	case "payment.failed":
		var e event.PaymentFailedEvent
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			c.log.Error().Err(err).Msg("gagal unmarshal PaymentFailedEvent")
			_ = msg.Nack(false, false)
			return
		}
		processErr = handler.HandlePaymentFailed(ctx, e)

	case "subscription.expiring":
		var e event.SubscriptionExpiringEvent
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			c.log.Error().Err(err).Msg("gagal unmarshal SubscriptionExpiringEvent")
			_ = msg.Nack(false, false)
			return
		}
		processErr = handler.HandleSubscriptionExpiring(ctx, e)

	case "subscription.expired":
		var e event.SubscriptionExpiredEvent
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			c.log.Error().Err(err).Msg("gagal unmarshal SubscriptionExpiredEvent")
			_ = msg.Nack(false, false)
			return
		}
		processErr = handler.HandleSubscriptionExpired(ctx, e)

	default:
		c.log.Warn().Str("routing_key", msg.RoutingKey).Msg("routing key tidak dikenal, pesan diabaikan")
		_ = msg.Ack(false)
		return
	}

	if processErr != nil {
		c.log.Error().Err(processErr).
			Str("routing_key", msg.RoutingKey).
			Int32("retry_count", retryCount).
			Msg("gagal proses notification")

		if retryCount >= maxRetryCount {
			c.log.Warn().
				Str("routing_key", msg.RoutingKey).
				Int32("retry_count", retryCount).
				Msg("retry habis, pesan dikirim ke DLQ")
			_ = msg.Nack(false, false)
			return
		}

		retryDelays := []time.Duration{10 * time.Second, 30 * time.Second, 90 * time.Second}
		delayIndex := int(retryCount)
		if delayIndex >= len(retryDelays) {
			delayIndex = len(retryDelays) - 1
		}
		time.Sleep(retryDelays[delayIndex])
		_ = msg.Nack(false, true)
		return
	}
	_ = msg.Ack(false)
}

// extractRetryCount membaca jumlah retry dari x-death header RabbitMQ.
func extractRetryCount(headers amqp.Table) int32 {
	xDeath, ok := headers["x-death"]
	if !ok {
		return 0
	}
	deaths, ok := xDeath.([]interface{})
	if !ok || len(deaths) == 0 {
		return 0
	}
	death, ok := deaths[0].(amqp.Table)
	if !ok {
		return 0
	}
	count, ok := death["count"].(int64)
	if !ok {
		return 0
	}
	return int32(count)
}

// Ping memeriksa apakah koneksi RabbitMQ masih aktif.
func (c *RabbitMQConsumer) Ping() error {
	if c.conn.IsClosed() {
		return fmt.Errorf("koneksi RabbitMQ tertutup")
	}
	return nil
}

// Close menutup channel dan koneksi RabbitMQ secara berurutan.
func (c *RabbitMQConsumer) Close() error {
	if err := c.channel.Close(); err != nil {
		return fmt.Errorf("gagal tutup channel RabbitMQ: %w", err)
	}
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("gagal tutup koneksi RabbitMQ: %w", err)
	}
	return nil
}
