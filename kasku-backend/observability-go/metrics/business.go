package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Counter mengembalikan CounterVec yang sudah ter-register di registry ini.
// Label `service` otomatis ditambahkan sebagai const label. Idempotent —
// register dua kali dengan nama yang sama akan reuse instance lama.
//
// Pakai untuk business metric per service:
//
//	loginAttempts := reg.Counter("kasku_auth_login_attempts_total",
//	    "Jumlah upaya login.", []string{"result"})
//	loginAttempts.WithLabelValues("success").Inc()
func (r *Registry) Counter(name, help string, labels []string) *prometheus.CounterVec {
	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        name,
			Help:        help,
			ConstLabels: prometheus.Labels{"service": r.Service},
		},
		labels,
	)
	if err := r.Reg.Register(c); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return are.ExistingCollector.(*prometheus.CounterVec)
		}
		panic(err)
	}
	return c
}

// Histogram analog ke Counter, untuk metric distribusi seperti latency atau
// payload size. Buckets bawaan = HTTP latency buckets; override via param.
func (r *Registry) Histogram(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	if len(buckets) == 0 {
		buckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}
	}
	h := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        name,
			Help:        help,
			Buckets:     buckets,
			ConstLabels: prometheus.Labels{"service": r.Service},
		},
		labels,
	)
	if err := r.Reg.Register(h); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return are.ExistingCollector.(*prometheus.HistogramVec)
		}
		panic(err)
	}
	return h
}

// Gauge analog ke Counter untuk metric yang bisa naik-turun.
func (r *Registry) Gauge(name, help string, labels []string) *prometheus.GaugeVec {
	g := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        name,
			Help:        help,
			ConstLabels: prometheus.Labels{"service": r.Service},
		},
		labels,
	)
	if err := r.Reg.Register(g); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return are.ExistingCollector.(*prometheus.GaugeVec)
		}
		panic(err)
	}
	return g
}
