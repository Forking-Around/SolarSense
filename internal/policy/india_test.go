package policy

import "testing"

func TestIndiaResidentialSubsidySlabs(t *testing.T) {
	tests := []struct{ kw, want float64 }{{1, 30000}, {2, 60000}, {3, 78000}, {5, 78000}}
	for _, tc := range tests {
		if got := IndiaResidentialSubsidy(tc.kw); got != tc.want {
			t.Errorf("%.1f kW: got %.0f, want %.0f", tc.kw, got, tc.want)
		}
	}
}
