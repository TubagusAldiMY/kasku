package dto

// DashboardStatsDTO adalah body GET /v1/admin/stats/dashboard.
type DashboardStatsDTO struct {
	TotalUsers        int64            `json:"total_users"`
	TotalActiveUsers  int64            `json:"total_active_users"`
	NewUsersLast7Days int64            `json:"new_users_last_7_days"`
	TierDistribution  map[string]int64 `json:"tier_distribution"`
	MRRIDR            int64            `json:"mrr_idr"`
	ChurnRate30dPct   float64          `json:"churn_rate_30d_pct"`
}
