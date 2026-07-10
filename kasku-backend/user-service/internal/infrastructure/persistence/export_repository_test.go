package persistence

import "testing"

func TestValidTenantSchema(t *testing.T) {
	cases := []struct {
		name   string
		schema string
		want   bool
	}{
		{"valid underscore uuid", "tenant_550e8400_e29b_41d4_a716_446655440000", true},
		{"empty", "", false},
		{"public", "public", false},
		{"sql injection", "tenant_x; DROP TABLE users--", false},
		{"uppercase hex", "tenant_550E8400_E29B_41D4_A716_446655440000", false},
		{"missing prefix", "550e8400_e29b_41d4_a716_446655440000", false},
		{"too short", "tenant_abc", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := validTenantSchema(tc.schema); got != tc.want {
				t.Errorf("validTenantSchema(%q) = %v, want %v", tc.schema, got, tc.want)
			}
		})
	}
}
