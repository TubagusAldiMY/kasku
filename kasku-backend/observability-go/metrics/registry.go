// Package metrics menyediakan registry Prometheus + helper instrumentasi
// (HTTP middleware, DB pool collector, business counter) yang dipakai
// bersama oleh semua service Go di KasKu.
//
// Pattern: setiap service punya satu *Registry di main.go yang di-pass ke
// router (untuk middleware HTTP) dan ke pool collector. Registry punya
// Go runtime + process collector default.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Registry membungkus *prometheus.Registry yang sudah ter-pre-register
// dengan Go runtime + process collector + label `service` konstant.
type Registry struct {
	Service string
	Reg     *prometheus.Registry
}

// NewRegistry membuat Registry baru dengan service name yang dipakai sebagai
// label konstant di semua metric yang di-register lewat helper di package ini.
func NewRegistry(serviceName string) *Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return &Registry{Service: serviceName, Reg: reg}
}
