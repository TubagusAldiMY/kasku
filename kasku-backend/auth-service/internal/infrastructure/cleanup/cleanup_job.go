// Package cleanup menjalankan retention/garbage-collection job untuk tabel yang
// menyimpan token dengan TTL pendek di kasku_auth. Job berjalan sebagai
// goroutine background yang di-tick periodik (default 1 jam) dan akan berhenti
// saat context dibatalkan (graceful shutdown).
//
// Yang dibersihkan:
//
//	refresh_tokens          — expired > 30 hari (retention buat audit forensic)
//	email_verifications     — expired > 7 hari
//	password_reset_tokens   — used > 7 hari ATAU expired > 7 hari
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
//	           Cocok untuk first deploy di production sambil verifikasi metrics.
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

// cleanupTask mendeskripsikan satu task delete.
type cleanupTask struct {
	name       string // untuk logging
	countQuery string // SELECT COUNT(*) ... (dipakai di dry-run mode)
	deleteSQL  string // DELETE ... LIMIT $1
}

const batchLimit = 10000

var tasks = []cleanupTask{
	{
		name:       "refresh_tokens",
		countQuery: `SELECT count(*) FROM public.refresh_tokens WHERE expires_at < NOW() - INTERVAL '30 days'`,
		deleteSQL: `DELETE FROM public.refresh_tokens
			WHERE id IN (
				SELECT id FROM public.refresh_tokens
				WHERE expires_at < NOW() - INTERVAL '30 days'
				LIMIT $1
			)`,
	},
	{
		name:       "email_verifications",
		countQuery: `SELECT count(*) FROM public.email_verifications WHERE expires_at < NOW() - INTERVAL '7 days'`,
		deleteSQL: `DELETE FROM public.email_verifications
			WHERE id IN (
				SELECT id FROM public.email_verifications
				WHERE expires_at < NOW() - INTERVAL '7 days'
				LIMIT $1
			)`,
	},
	{
		name: "password_reset_tokens",
		countQuery: `SELECT count(*) FROM public.password_reset_tokens
			WHERE (used_at IS NOT NULL AND used_at < NOW() - INTERVAL '7 days')
			   OR (used_at IS NULL AND expires_at < NOW() - INTERVAL '7 days')`,
		deleteSQL: `DELETE FROM public.password_reset_tokens
			WHERE id IN (
				SELECT id FROM public.password_reset_tokens
				WHERE (used_at IS NOT NULL AND used_at < NOW() - INTERVAL '7 days')
				   OR (used_at IS NULL AND expires_at < NOW() - INTERVAL '7 days')
				LIMIT $1
			)`,
	},
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
// Mengakumulasi jumlah baris yang dihapus per task untuk dilaporkan via log.
// Error pada satu task tidak menghentikan task lain.
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
