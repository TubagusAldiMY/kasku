package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

const (
	exchangeName         = "kasku.events"
	exchangeType         = "topic"
	queueName            = "kasku.user-service"
	routingKeyRegistered = "user.registered"
	dlxExchange          = "kasku.events.dlx"
	msgTTL               = 86400000 // 24 jam dalam ms
	maxRetries           = 3
)

// UserRegisteredEvent adalah payload event dari auth-service.
type UserRegisteredEvent struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// EventHandler mendefinisikan kontrak handler untuk event yang diterima.
type EventHandler interface {
	HandleUserRegistered(ctx context.Context, event UserRegisteredEvent) error
}

// RabbitMQConsumer mengkonsumsi events dari RabbitMQ.
type RabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	log     zerolog.Logger
}

// NewRabbitMQConsumer membuat koneksi ke RabbitMQ, declare queue, dan bind ke exchange.
func NewRabbitMQConsumer(amqpURL string, log zerolog.Logger) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("gagal koneksi ke RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("gagal buat channel RabbitMQ: %w", err),
			wrapCloseError("gagal tutup koneksi RabbitMQ", conn.Close()),
		)
	}

	// Declare exchange (idempotent)
	if err := ch.ExchangeDeclare(exchangeName, exchangeType, true, false, false, false, nil); err != nil {
		return nil, errors.Join(
			fmt.Errorf("gagal declare exchange: %w", err),
			wrapCloseError("gagal tutup channel RabbitMQ", ch.Close()),
			wrapCloseError("gagal tutup koneksi RabbitMQ", conn.Close()),
		)
	}

	// Declare queue dengan DLX
	_, err = ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange": dlxExchange,
			"x-message-ttl":          int32(msgTTL),
		},
	)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("gagal declare queue %s: %w", queueName, err),
			wrapCloseError("gagal tutup channel RabbitMQ", ch.Close()),
			wrapCloseError("gagal tutup koneksi RabbitMQ", conn.Close()),
		)
	}

	// Bind queue ke exchange
	if err := ch.QueueBind(queueName, routingKeyRegistered, exchangeName, false, nil); err != nil {
		return nil, errors.Join(
			fmt.Errorf("gagal bind queue %s: %w", queueName, err),
			wrapCloseError("gagal tutup channel RabbitMQ", ch.Close()),
			wrapCloseError("gagal tutup koneksi RabbitMQ", conn.Close()),
		)
	}

	return &RabbitMQConsumer{conn: conn, channel: ch, log: log}, nil
}

// StartConsuming memulai goroutine untuk mengkonsumsi events.
// Berhenti saat ctx dibatalkan.
func (c *RabbitMQConsumer) StartConsuming(ctx context.Context, handler EventHandler) error {
	msgs, err := c.channel.Consume(
		queueName,
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("gagal mulai consume dari queue %s: %w", queueName, err)
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

func (c *RabbitMQConsumer) processMessage(ctx context.Context, msg amqp.Delivery, handler EventHandler) {
	retryCount := int32(0)
	if xDeath, ok := msg.Headers["x-death"]; ok {
		if deaths, ok := xDeath.([]interface{}); ok && len(deaths) > 0 {
			if death, ok := deaths[0].(amqp.Table); ok {
				if count, ok := death["count"].(int64); ok {
					retryCount = int32(count)
				}
			}
		}
	}

	var processErr error
	switch msg.RoutingKey {
	case routingKeyRegistered:
		var event UserRegisteredEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			c.log.Error().Err(err).Msg("gagal unmarshal UserRegisteredEvent")
			_ = msg.Nack(false, false) // buang ke DLQ
			return
		}
		processErr = handler.HandleUserRegistered(ctx, event)
	default:
		c.log.Warn().Str("routing_key", msg.RoutingKey).Msg("routing key tidak dikenal, dibuang")
		_ = msg.Ack(false)
		return
	}

	if processErr != nil {
		c.log.Error().
			Err(processErr).
			Str("routing_key", msg.RoutingKey).
			Int32("retry_count", retryCount).
			Msg("gagal proses message")

		if retryCount >= maxRetries {
			c.log.Error().Str("routing_key", msg.RoutingKey).Msg("max retries tercapai, kirim ke DLQ")
			_ = msg.Nack(false, false)
			return
		}

		// Retry dengan exponential backoff: 10s, 30s, 90s
		delays := []time.Duration{10 * time.Second, 30 * time.Second, 90 * time.Second}
		delayIndex := retryCount
		if int(delayIndex) >= len(delays) {
			delayIndex = int32(len(delays) - 1)
		}
		time.Sleep(delays[delayIndex])
		_ = msg.Nack(false, true) // requeue
		return
	}

	_ = msg.Ack(false)
}

// Ping memeriksa apakah koneksi RabbitMQ masih aktif.
func (c *RabbitMQConsumer) Ping() error {
	if c.conn.IsClosed() {
		return fmt.Errorf("koneksi RabbitMQ tertutup")
	}
	return nil
}

// Close menutup channel dan koneksi RabbitMQ.
func (c *RabbitMQConsumer) Close() error {
	return errors.Join(
		wrapCloseError("gagal tutup channel", c.channel.Close()),
		wrapCloseError("gagal tutup koneksi", c.conn.Close()),
	)
}

func wrapCloseError(message string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
