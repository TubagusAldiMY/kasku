package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

const (
	defaultPollInterval = 2 * time.Second
	defaultBatchSize    = 25
)

type Publisher interface {
	PublishRaw(ctx context.Context, routingKey string, body []byte) error
}

type Dispatcher struct {
	pool         *pgxpool.Pool
	publisher    Publisher
	log          zerolog.Logger
	pollInterval time.Duration
	batchSize    int
}

func NewDispatcher(pool *pgxpool.Pool, publisher Publisher, log zerolog.Logger) *Dispatcher {
	return &Dispatcher{
		pool:         pool,
		publisher:    publisher,
		log:          log,
		pollInterval: defaultPollInterval,
		batchSize:    defaultBatchSize,
	}
}

func (d *Dispatcher) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(d.pollInterval)
		defer ticker.Stop()

		for {
			if err := d.flush(ctx); err != nil {
				d.log.Warn().Err(err).Msg("outbox flush gagal")
			}

			select {
			case <-ctx.Done():
				d.log.Info().Msg("outbox dispatcher berhenti")
				return
			case <-ticker.C:
			}
		}
	}()
}

func (d *Dispatcher) flush(ctx context.Context) error {
	tx, err := d.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("gagal mulai transaksi outbox: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT id, routing_key, payload::text
		FROM public.outbox_events
		WHERE published_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, d.batchSize)
	if err != nil {
		return fmt.Errorf("gagal query outbox: %w", err)
	}
	defer rows.Close()

	type outboxEvent struct {
		id         uuid.UUID
		routingKey string
		payload    []byte
	}

	events := make([]outboxEvent, 0, d.batchSize)
	for rows.Next() {
		var event outboxEvent
		var payload string
		if err := rows.Scan(&event.id, &event.routingKey, &payload); err != nil {
			return fmt.Errorf("gagal scan outbox event: %w", err)
		}
		event.payload = []byte(payload)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("gagal iterasi outbox event: %w", err)
	}

	for _, event := range events {
		if err := d.publisher.PublishRaw(ctx, event.routingKey, event.payload); err != nil {
			_, updateErr := tx.Exec(ctx, `
				UPDATE public.outbox_events
				SET publish_attempts = publish_attempts + 1, last_error = $2
				WHERE id = $1
			`, event.id, err.Error())
			if updateErr != nil {
				return fmt.Errorf("gagal update outbox error: %w", updateErr)
			}
			d.log.Warn().
				Err(err).
				Str("event_id", event.id.String()).
				Str("routing_key", event.routingKey).
				Msg("publish outbox event gagal")
			continue
		}

		if _, err := tx.Exec(ctx, `
			UPDATE public.outbox_events
			SET published_at = now(), publish_attempts = publish_attempts + 1, last_error = NULL
			WHERE id = $1
		`, event.id); err != nil {
			return fmt.Errorf("gagal mark outbox published: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("gagal commit outbox flush: %w", err)
	}
	return nil
}
