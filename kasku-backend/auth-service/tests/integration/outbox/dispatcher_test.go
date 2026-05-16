package outbox_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/outbox"
	"github.com/TubagusAldiMY/kasku/auth-service/tests/integration"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePublisher capture semua PublishRaw call. Bisa diatur error untuk
// test path failure (retry attempt + last_error).
type fakePublisher struct {
	mu         sync.Mutex
	calls      []publishCall
	failNext   bool
	failedKeys map[string]bool // routing keys yang harus error
}

type publishCall struct {
	routingKey string
	body       []byte
}

func (f *fakePublisher) PublishRaw(_ context.Context, routingKey string, body []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failNext {
		f.failNext = false
		return errors.New("publisher fail (one-shot)")
	}
	if f.failedKeys != nil && f.failedKeys[routingKey] {
		return errors.New("publisher fail (key-targeted)")
	}

	f.calls = append(f.calls, publishCall{
		routingKey: routingKey,
		body:       append([]byte(nil), body...),
	})
	return nil
}

func (f *fakePublisher) publishedKeys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	keys := make([]string, len(f.calls))
	for i, c := range f.calls {
		keys[i] = c.routingKey
	}
	return keys
}

func insertOutboxEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, routingKey, payload string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO public.outbox_events (id, event_type, routing_key, payload, created_at)
		VALUES ($1, $2, $3, $4::jsonb, NOW())
	`, id, routingKey, routingKey, payload)
	require.NoError(t, err)
	return id
}

func getOutboxEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (publishedAt *time.Time, attempts int, lastErr *string) {
	t.Helper()
	err := pool.QueryRow(ctx, `
		SELECT published_at, publish_attempts, last_error
		FROM public.outbox_events WHERE id = $1
	`, id).Scan(&publishedAt, &attempts, &lastErr)
	require.NoError(t, err)
	return
}

// pollUntil retries fn sampai true atau timeout.
func pollUntil(t *testing.T, timeout time.Duration, fn func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func TestDispatcher_Happy_PublishesAndMarks(t *testing.T) {
	pool := integration.SetupPostgres(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pub := &fakePublisher{}
	disp := outbox.NewDispatcher(pool, pub, zerolog.Nop())

	id1 := insertOutboxEvent(t, ctx, pool, "user.registered", `{"user":"a"}`)
	id2 := insertOutboxEvent(t, ctx, pool, "user.password_reset", `{"user":"b"}`)

	disp.Start(ctx)

	require.True(t, pollUntil(t, 10*time.Second, func() bool {
		return len(pub.publishedKeys()) >= 2
	}), "dispatcher should publish 2 events within 10s")

	// Both event harus published_at set
	publishedAt1, attempts1, lastErr1 := getOutboxEvent(t, ctx, pool, id1)
	require.NotNil(t, publishedAt1)
	assert.Equal(t, 1, attempts1)
	assert.Nil(t, lastErr1)

	publishedAt2, attempts2, _ := getOutboxEvent(t, ctx, pool, id2)
	require.NotNil(t, publishedAt2)
	assert.Equal(t, 1, attempts2)
}

func TestDispatcher_PublishFail_IncrementsAttemptsLastError(t *testing.T) {
	pool := integration.SetupPostgres(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pub := &fakePublisher{failedKeys: map[string]bool{"bad.key": true}}
	disp := outbox.NewDispatcher(pool, pub, zerolog.Nop())

	idGood := insertOutboxEvent(t, ctx, pool, "good.key", `{}`)
	idBad := insertOutboxEvent(t, ctx, pool, "bad.key", `{}`)

	disp.Start(ctx)

	require.True(t, pollUntil(t, 10*time.Second, func() bool {
		_, attempts, _ := getOutboxEvent(t, ctx, pool, idBad)
		return attempts > 0
	}), "bad event should have attempt incremented within 10s")

	// Good: published
	pubAt, attemptsGood, errGood := getOutboxEvent(t, ctx, pool, idGood)
	require.NotNil(t, pubAt)
	assert.GreaterOrEqual(t, attemptsGood, 1)
	assert.Nil(t, errGood)

	// Bad: NOT published, but attempt + last_error set
	pubAtBad, attemptsBad, errBad := getOutboxEvent(t, ctx, pool, idBad)
	assert.Nil(t, pubAtBad, "failed event must NOT be marked published")
	assert.GreaterOrEqual(t, attemptsBad, 1)
	require.NotNil(t, errBad)
	assert.Contains(t, *errBad, "publisher fail")
}

func TestDispatcher_StopsOnContextCancel(t *testing.T) {
	pool := integration.SetupPostgres(t)

	pub := &fakePublisher{}
	disp := outbox.NewDispatcher(pool, pub, zerolog.Nop())

	ctx, cancel := context.WithCancel(context.Background())
	disp.Start(ctx)

	// Insert event
	insertOutboxEvent(t, ctx, pool, "test", `{}`)

	// Wait sebentar
	time.Sleep(3 * time.Second)
	cancel()

	// Setelah cancel, no new publishes terjadi
	preCount := len(pub.publishedKeys())
	insertOutboxEvent(t, context.Background(), pool, "after-cancel", `{}`)
	time.Sleep(3 * time.Second)
	postCount := len(pub.publishedKeys())
	assert.Equal(t, preCount, postCount, "no events should be published after cancel")
}
