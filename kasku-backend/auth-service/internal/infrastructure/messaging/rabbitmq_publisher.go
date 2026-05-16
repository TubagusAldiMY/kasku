package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	exchangeName   = "kasku.events"
	exchangeType   = "topic"
	publishTimeout = 5 * time.Second

	RoutingKeyUserRegistered          = "user.registered"
	RoutingKeyEmailVerificationResent = "user.email_verification_resent"
	RoutingKeyPasswordResetRequested  = "user.password_reset_requested"
)

// UserRegisteredEvent merupakan payload event yang diterbitkan saat user berhasil registrasi.
// Raw token dikirim ke notification service untuk dikirim via email — tidak pernah disimpan di DB.
type UserRegisteredEvent struct {
	UserID            string `json:"user_id"`
	Email             string `json:"email"`
	Username          string `json:"username"`
	VerificationToken string `json:"verification_token"`
}

// EmailVerificationResentEvent merupakan payload event resend verifikasi email.
type EmailVerificationResentEvent struct {
	UserID            string `json:"user_id"`
	Email             string `json:"email"`
	VerificationToken string `json:"verification_token"`
}

// PasswordResetRequestedEvent merupakan payload event permintaan reset password.
type PasswordResetRequestedEvent struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	ResetToken string `json:"reset_token"`
}

// EventPublisher mendefinisikan kontrak untuk menerbitkan event ke message broker.
//
//go:generate mockgen -source=$GOFILE -destination=../../../tests/mocks/mock_event_publisher.go -package=mocks
type EventPublisher interface {
	PublishUserRegistered(ctx context.Context, event UserRegisteredEvent) error
	PublishEmailVerificationResent(ctx context.Context, event EmailVerificationResentEvent) error
	PublishPasswordResetRequested(ctx context.Context, event PasswordResetRequestedEvent) error
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
		conn.Close()
		return nil, fmt.Errorf("gagal buat channel RabbitMQ: %w", err)
	}

	// Declare exchange sebagai durable topic — idempotent (aman dijalankan ulang)
	if err := ch.ExchangeDeclare(
		exchangeName,
		exchangeType,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("gagal declare exchange RabbitMQ: %w", err)
	}

	return &RabbitMQPublisher{conn: conn, channel: ch}, nil
}

// PublishUserRegistered menerbitkan event user.registered ke kasku.events exchange.
func (p *RabbitMQPublisher) PublishUserRegistered(ctx context.Context, event UserRegisteredEvent) error {
	return p.publish(ctx, RoutingKeyUserRegistered, event)
}

// PublishEmailVerificationResent menerbitkan event resend verifikasi email.
func (p *RabbitMQPublisher) PublishEmailVerificationResent(ctx context.Context, event EmailVerificationResentEvent) error {
	return p.publish(ctx, RoutingKeyEmailVerificationResent, event)
}

// PublishPasswordResetRequested menerbitkan event permintaan reset password.
func (p *RabbitMQPublisher) PublishPasswordResetRequested(ctx context.Context, event PasswordResetRequestedEvent) error {
	return p.publish(ctx, RoutingKeyPasswordResetRequested, event)
}

// PublishRaw menerbitkan payload JSON yang sudah tersimpan di outbox.
func (p *RabbitMQPublisher) PublishRaw(ctx context.Context, routingKey string, body []byte) error {
	return p.publishBytes(ctx, routingKey, body)
}

// publish melakukan serialisasi payload ke JSON dan menerbitkan ke exchange dengan routing key yang diberikan.
func (p *RabbitMQPublisher) publish(ctx context.Context, routingKey string, payload interface{}) error {
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
			DeliveryMode: amqp.Persistent, // pesan tidak hilang saat RabbitMQ restart
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
	if err := p.channel.Close(); err != nil {
		return fmt.Errorf("gagal tutup channel RabbitMQ: %w", err)
	}
	if err := p.conn.Close(); err != nil {
		return fmt.Errorf("gagal tutup koneksi RabbitMQ: %w", err)
	}
	return nil
}
