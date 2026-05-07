package handler

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/TubagusAldiMY/kasku/api-gateway/internal/delivery/http/middleware"
)

// ProxyHandler mengelola reverse proxy ke upstream microservices.
type ProxyHandler struct {
	proxies map[string]*httputil.ReverseProxy
	logger  zerolog.Logger
}

// NewProxyHandler membuat instance ProxyHandler dengan reverse proxy ke setiap upstream.
func NewProxyHandler(upstreams map[string]string, logger zerolog.Logger) (*ProxyHandler, error) {
	proxies := make(map[string]*httputil.ReverseProxy, len(upstreams))

	for name, rawURL := range upstreams {
		target, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("upstream URL tidak valid untuk %s: %w", name, err)
		}

		// Capture loop variable
		upstreamTarget := target
		upstreamName := name

		proxy := &httputil.ReverseProxy{
			// Rewrite menggantikan Director (deprecated sejak Go 1.20).
			// SetURL menyetel scheme, host, dan menyesuaikan path ke upstream.
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(upstreamTarget)
				pr.Out.Host = upstreamTarget.Host
				// Hapus hop-by-hop headers yang tidak perlu diteruskan
				pr.Out.Header.Del("Te")
				pr.Out.Header.Del("Trailers")
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				zerolog.Ctx(r.Context()).Error().
					Err(err).
					Str("upstream", upstreamName).
					Str("path", r.URL.Path).
					Msg("proxy error ke upstream service")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"SERVICE_UNAVAILABLE","message":"Service tidak tersedia sementara."}}`))
			},
		}

		proxies[name] = proxy
	}

	return &ProxyHandler{proxies: proxies, logger: logger}, nil
}

// ProxyTo mengembalikan gin.HandlerFunc yang meneruskan request ke upstream yang ditentukan.
func (h *ProxyHandler) ProxyTo(upstreamName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		proxy, ok := h.proxies[upstreamName]
		if !ok {
			c.JSON(http.StatusBadGateway, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "SERVICE_UNAVAILABLE",
					"message": "Upstream service tidak dikonfigurasi.",
				},
			})
			return
		}

		// Inject correlation ID ke request upstream
		correlationID := middleware.GetCorrelationID(c)
		if correlationID != "" {
			c.Request.Header.Set(middleware.CorrelationIDHeader, correlationID)
		}

		h.logger.Debug().
			Str("upstream", upstreamName).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("correlation_id", correlationID).
			Msg("proxying request ke upstream")

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
