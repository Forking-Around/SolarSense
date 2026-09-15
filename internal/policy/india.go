package policy

import "math"

// IndiaResidentialSubsidy implements the central residential CFA screening rule.
// Final eligibility must be checked against the active, sourced policy record.
func IndiaResidentialSubsidy(capacityKW float64) float64 {
	return math.Min(capacityKW, 2)*30000 + math.Max(0, math.Min(capacityKW, 3)-2)*18000
}
