// Package cleanup menjalankan retention/garbage-collection job untuk tabel
// outbox_events di kasku_billing. Job berjalan sebagai goroutine background
// yang di-tick periodik (default 1 jam) dan akan berhenti saat context dibatalkan.
//
// Yang dibersihkan:
//
//	outbox_events           — published > 7 hari (audit retention)
//
// Setiap DELETE dibatasi LIMIT 10000 untuk menghindari long-running lock di
// production, lalu di-loop sampai tidak ada lagi baris yang match.
package cleanup

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// CleanupJob menjalankan retention task periodik.
type CleanupJob struct {
	pool     *pgxpool.Pool
	log      zerolog.Logger
	interval time.Duration
	dryRun   bool
}

// NewCleanupJob membuat instance CleanupJob.
//
//	interval — periode tick (mis. 1 jam).
//	dryRun   — jika true, hanya LOG count yang akan dihapus, tanpa DELETE aktual.
func NewCleanupJob(pool *pgxpool.Pool, log zerolog.Logger, interval time.Duration, dryRun bool) *CleanupJob {
	return &CleanupJob{
		pool:     pool,
		log:      log,
		interval: interval,
		dryRun:   dryRun,
	}
}

// Run memulai loop ticker. Blocking — caller biasanya memanggil di goroutine.
// Akan return saat ctx dibatalkan.
func (j *CleanupJob) Run(ctx context.Context) {
	j.log.Info().
		Dur("interval", j.interval).
		Bool("dry_run", j.dryRun).
		Msg("cleanup job started")

	// Jalan sekali di awal supaya garbage tidak menumpuk saat boot lama.
	if err := j.RunOnce(ctx); err != nil {
		j.log.Warn().Err(err).Msg("cleanup runOnce gagal di startup")
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			j.log.Info().Msg("cleanup job stopped")
			return
		case <-ticker.C:
			if err := j.RunOnce(ctx); err != nil {
				j.log.Warn().Err(err).Msg("cleanup runOnce gagal")
			}
		}
	}
}

type cleanupTask struct {
	name       string
	countQuery string
	deleteSQL  string
}

const batchLimit = 10000

var tasks = []cleanupTask{
	{
		name:       "outbox_events",
		countQuery: `SELECT count(*) FROM public.outbox_events WHERE published_at IS NOT NULL AND published_at < NOW() - INTERVAL '7 days'`,
		deleteSQL: `DELETE FROM public.outbox_events
			WHERE id IN (
				SELECT id FROM public.outbox_events
				WHERE published_at IS NOT NULL AND published_at < NOW() - INTERVAL '7 days'
				LIMIT $1
			)`,
	},
}

// RunOnce menjalankan satu pass cleanup untuk semua task.
func (j *CleanupJob) RunOnce(ctx context.Context) error {
	start := time.Now()
	summary := make(map[string]int64, len(tasks))

	for _, t := range tasks {
		count, err := j.runTask(ctx, t)
		if err != nil {
			j.log.Warn().Err(err).Str("task", t.name).Msg("cleanup task gagal")
			continue
		}
		summary[t.name] = count
	}

	event := j.log.Info().
		Dur("duration_ms", time.Since(start)).
		Bool("dry_run", j.dryRun)
	for name, count := range summary {
		event = event.Int64(name, count)
	}
	event.Msg("cleanup pass selesai")

	return nil
}

func (j *CleanupJob) runTask(ctx context.Context, t cleanupTask) (int64, error) {
	if j.dryRun {
		var count int64
		if err := j.pool.QueryRow(ctx, t.countQuery).Scan(&count); err != nil {
			return 0, fmt.Errorf("count query: %w", err)
		}
		return count, nil
	}

	var total int64
	for {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}

		tag, err := j.pool.Exec(ctx, t.deleteSQL, batchLimit)
		if err != nil {
			return total, fmt.Errorf("delete: %w", err)
		}
		affected := tag.RowsAffected()
		total += affected
		if affected < int64(batchLimit) {
			break
		}
	}
	return total, nil
}
