package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	exchangeName   = "kasku.events"
	exchangeType   = "topic"
	publishTimeout = 5 * time.Second

	// RoutingKeySubscriptionExpired diterbitkan oleh cron expiry saat status
	// subscription berhasil di-flip dari ACTIVE → EXPIRED.
	RoutingKeySubscriptionExpired = "subscription.expired"

	// RoutingKeySubscriptionExpiring diterbitkan saat current_period_end mendekati
	// (default H-7) supaya notification-service bisa kirim reminder.
	RoutingKeySubscriptionExpiring = "subscription.expiring"
)

// SubscriptionExpiredEvent merupakan payload event subscription.expired.
type SubscriptionExpiredEvent struct {
	SubscriptionID string `json:"subscription_id"`
	UserID         string `json:"user_id"`
	PlanName       string `json:"plan_name"`
	PreviousStatus string `json:"previous_status"`
	ExpiredAt      string `json:"expired_at"`
}

// SubscriptionExpiringEvent merupakan payload event subscription.expiring (H-N).
type SubscriptionExpiringEvent struct {
	SubscriptionID string `json:"subscription_id"`
	UserID         string `json:"user_id"`
	PlanName       string `json:"plan_name"`
	PeriodEnd      string `json:"period_end"`
	DaysRemaining  int    `json:"days_remaining"`
}

// EventPublisher mendefinisikan kontrak untuk menerbitkan event ke message broker.
//
//go:generate mockgen -source=$GOFILE -destination=../../../tests/mocks/mock_event_publisher.go -package=mocks
type EventPublisher interface {
	PublishSubscriptionExpired(ctx context.Context, event SubscriptionExpiredEvent) error
	PublishSubscriptionExpiring(ctx context.Context, event SubscriptionExpiringEvent) error
	PublishRaw(ctx context.Context, routingKey string, body []byte) error
	Close() error
	Ping() error
}

// RabbitMQPublisher mengimplementasikan EventPublisher menggunakan RabbitMQ.
type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQPublisher membuat koneksi ke RabbitMQ dan mendeclare exchange kasku.events.
func NewRabbitMQPublisher(amqpURL string) (*RabbitMQPublisher, error) {
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

	// Declare exchange sebagai durable topic — idempotent.
	if err := ch.ExchangeDeclare(
		exchangeName,
		exchangeType,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		return nil, errors.Join(
			fmt.Errorf("gagal declare exchange RabbitMQ: %w", err),
			wrapCloseError("gagal tutup channel RabbitMQ", ch.Close()),
			wrapCloseError("gagal tutup koneksi RabbitMQ", conn.Close()),
		)
	}

	return &RabbitMQPublisher{conn: conn, channel: ch}, nil
}

// PublishSubscriptionExpired menerbitkan event subscription.expired.
func (p *RabbitMQPublisher) PublishSubscriptionExpired(ctx context.Context, event SubscriptionExpiredEvent) error {
	return p.publish(ctx, RoutingKeySubscriptionExpired, event)
}

// PublishSubscriptionExpiring menerbitkan event subscription.expiring.
func (p *RabbitMQPublisher) PublishSubscriptionExpiring(ctx context.Context, event SubscriptionExpiringEvent) error {
	return p.publish(ctx, RoutingKeySubscriptionExpiring, event)
}

// PublishRaw menerbitkan payload JSON yang sudah tersimpan di outbox.
func (p *RabbitMQPublisher) PublishRaw(ctx context.Context, routingKey string, body []byte) error {
	return p.publishBytes(ctx, routingKey, body)
}

func (p *RabbitMQPublisher) publish(ctx context.Context, routingKey string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gagal marshal event payload: %w", err)
	}
	return p.publishBytes(ctx, routingKey, body)
}

func (p *RabbitMQPublisher) publishBytes(ctx context.Context, routingKey string, body []byte) error {
	publishCtx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	err := p.channel.PublishWithContext(
		publishCtx,
		exchangeName,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("gagal publish event ke RabbitMQ (routingKey=%s): %w", routingKey, err)
	}
	return nil
}

// Ping memeriksa apakah koneksi RabbitMQ masih aktif untuk health check.
func (p *RabbitMQPublisher) Ping() error {
	if p.conn.IsClosed() {
		return fmt.Errorf("koneksi RabbitMQ tertutup")
	}
	return nil
}

// Close menutup channel dan koneksi RabbitMQ dengan urutan yang benar.
func (p *RabbitMQPublisher) Close() error {
	return errors.Join(
		wrapCloseError("gagal tutup channel RabbitMQ", p.channel.Close()),
		wrapCloseError("gagal tutup koneksi RabbitMQ", p.conn.Close()),
	)
}

func wrapCloseError(message string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
