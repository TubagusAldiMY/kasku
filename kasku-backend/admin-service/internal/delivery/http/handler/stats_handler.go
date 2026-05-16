package handler

import (
	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/dto"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/response"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/gin-gonic/gin"
)

// StatsHandler menangani agregat statistik dashboard.
type StatsHandler struct {
	stats usecase.DashboardStatsUseCase
}

// NewStatsHandler membuat instance.
func NewStatsHandler(stats usecase.DashboardStatsUseCase) *StatsHandler {
	return &StatsHandler{stats: stats}
}

// Dashboard menangani GET /v1/admin/stats/dashboard.
func (h *StatsHandler) Dashboard(c *gin.Context) {
	out, err := h.stats.Execute(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, dto.DashboardStatsDTO{
		TotalUsers:        out.TotalUsers,
		TotalActiveUsers:  out.TotalActiveUsers,
		NewUsersLast7Days: out.NewUsersLast7Days,
		TierDistribution:  out.TierDistribution,
		MRRIDR:            out.MRRIDR,
		ChurnRate30dPct:   out.ChurnRate30dPct,
	})
}
