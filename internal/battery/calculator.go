package battery

import (
	"fmt"
	"math"

	"github.com/Forking-Around/SolarSense/internal/domain"
)

func Calculate(o domain.OutageProfile, gridRate, exportRate float64) domain.BatteryScenario {
	desired := math.Max(o.DesiredHours, o.HoursPerOutage)
	usable := o.EssentialWatts / 1000 * desired
	capacity := math.Ceil((usable/.85)*2) / 2
	if capacity < 2.5 {
		capacity = 2.5
	}
	cost := capacity * 28000
	avoided := capacity * .85 * math.Max(0, gridRate-exportRate) * 300
	payback := 99.0
	if avoided > 0 {
		payback = cost / avoided
	}
	savings := "Poor financial return"
	if payback <= 8 {
		savings = "Can make financial sense"
	}
	backup := "Optional"
	if o.FrequencyPerMonth >= 4 || o.FrequencyPerMonth*o.HoursPerOutage >= 8 {
		backup = "Worth considering for backup"
	}
	if o.FrequencyPerMonth >= 10 || o.FrequencyPerMonth*o.HoursPerOutage >= 20 {
		backup = "Strong backup case"
	}
	hours := capacity * .85 * 1000 / math.Max(1, o.EssentialWatts)
	return domain.BatteryScenario{CapacityKWh: capacity, UsableKWh: capacity * .85, BackupHours: hours, IncrementalCostINR: cost, AnnualSavingsINR: avoided, PaybackYears: payback, SavingsVerdict: savings, BackupVerdict: backup, Explanation: fmt.Sprintf("A %.1f kWh battery can run about %.0f W of essential loads for roughly %.1f hours.", capacity, o.EssentialWatts, hours)}
}
