package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler mengembalikan http.Handler yang serve Prometheus exposition
// format dari registry ini. Pakai gin.WrapH(reg.Handler()) di router untuk
// expose endpoint /metrics.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.Reg, promhttp.HandlerOpts{
		Registry:          r.Reg,
		EnableOpenMetrics: true,
	})
}
