package metrics

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// RegisterDBPool meng-register collector untuk pgxpool.Pool. Memakai
// GaugeFunc supaya nilai selalu fresh saat di-scrape tanpa goroutine
// terpisah. Aman dipanggil sekali per pool dari main.go.
//
// Metrics yang di-expose:
//   - kasku_db_pool_max_connections{service}
//   - kasku_db_pool_acquired_connections{service}
//   - kasku_db_pool_idle_connections{service}
//   - kasku_db_pool_acquire_total{service} (counter)
func (r *Registry) RegisterDBPool(pool *pgxpool.Pool) {
	svc := r.Service
	lbl := prometheus.Labels{"service": svc}

	r.Reg.MustRegister(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Name:        "kasku_db_pool_acquire_total",
			Help:        "Total acquire connection dari pgxpool sejak service start.",
			ConstLabels: lbl,
		},
		func() float64 { return float64(pool.Stat().AcquireCount()) },
	))
	r.Reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name:        "kasku_db_pool_max_connections",
			Help:        "Konfigurasi max connection pgxpool.",
			ConstLabels: lbl,
		},
		func() float64 { return float64(pool.Stat().MaxConns()) },
	))
	r.Reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name:        "kasku_db_pool_acquired_connections",
			Help:        "Jumlah koneksi pgxpool yang sedang di-acquire.",
			ConstLabels: lbl,
		},
		func() float64 { return float64(pool.Stat().AcquiredConns()) },
	))
	r.Reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name:        "kasku_db_pool_idle_connections",
			Help:        "Jumlah koneksi pgxpool yang idle.",
			ConstLabels: lbl,
		},
		func() float64 { return float64(pool.Stat().IdleConns()) },
	))
}
