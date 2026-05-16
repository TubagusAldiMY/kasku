package messaging_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/messaging"
	"github.com/TubagusAldiMY/kasku/billing-service/tests/integration"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bindQueue declare ephemeral queue + bind ke exchange dengan routing key
// untuk consumer test.
func bindQueue(t *testing.T, conn *amqp.Connection, routingKey string) <-chan amqp.Delivery {
	t.Helper()
	ch, err := conn.Channel()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ch.Close() })

	require.NoError(t, ch.ExchangeDeclare("kasku.events", "topic", true, false, false, false, nil))

	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	require.NoError(t, err)

	require.NoError(t, ch.QueueBind(q.Name, routingKey, "kasku.events", false, nil))

	deliveries, err := ch.Consume(q.Name, "", true, true, false, false, nil)
	require.NoError(t, err)
	return deliveries
}

func TestRabbitMQPublisher_PublishSubscriptionExpired(t *testing.T) {
	url, conn := integration.SetupRabbitMQ(t)

	deliveries := bindQueue(t, conn, messaging.RoutingKeySubscriptionExpired)

	pub, err := messaging.NewRabbitMQPublisher(url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pub.Close() })

	event := messaging.SubscriptionExpiredEvent{
		SubscriptionID: "sub-1",
		UserID:         "user-1",
		PlanName:       "BASIC",
		PreviousStatus: "ACTIVE",
		ExpiredAt:      "2026-01-01T00:00:00Z",
	}
	require.NoError(t, pub.PublishSubscriptionExpired(context.Background(), event))

	select {
	case d := <-deliveries:
		assert.Equal(t, "application/json", d.ContentType)
		var got messaging.SubscriptionExpiredEvent
		require.NoError(t, json.Unmarshal(d.Body, &got))
		assert.Equal(t, event, got)
	case <-time.After(5 * time.Second):
		t.Fatal("did not receive subscription.expired event within 5s")
	}
}

func TestRabbitMQPublisher_PublishRaw(t *testing.T) {
	url, conn := integration.SetupRabbitMQ(t)
	deliveries := bindQueue(t, conn, "custom.event")

	pub, err := messaging.NewRabbitMQPublisher(url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pub.Close() })

	payload := []byte(`{"hello":"world"}`)
	require.NoError(t, pub.PublishRaw(context.Background(), "custom.event", payload))

	select {
	case d := <-deliveries:
		assert.Equal(t, payload, d.Body)
	case <-time.After(5 * time.Second):
		t.Fatal("did not receive custom.event")
	}
}

func TestRabbitMQPublisher_Ping(t *testing.T) {
	url, _ := integration.SetupRabbitMQ(t)
	pub, err := messaging.NewRabbitMQPublisher(url)
	require.NoError(t, err)

	require.NoError(t, pub.Ping())

	require.NoError(t, pub.Close())
	assert.Error(t, pub.Ping(), "Ping should fail after Close")
}
