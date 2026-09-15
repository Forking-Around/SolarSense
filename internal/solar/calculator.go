package solar

import (
	"math"
	"strings"

	"github.com/Forking-Around/SolarSense/internal/domain"
)

const SqFtPerKW = 100.0

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func orientationFactor(direction string, tilt float64) float64 {
	d := strings.ToLower(direction)
	var factor float64
	switch d {
	case "south":
		factor = 1.00
	case "south-east", "south-west":
		factor = .96
	case "east", "west", "mixed":
		factor = .90
	case "north-east", "north-west":
		factor = .82
	case "north":
		factor = .72
	default:
		factor = .88
	}
	if tilt > 45 {
		factor -= math.Min(.12, (tilt-45)/250)
	}
	return clamp(factor, .60, 1)
}

func ZoneUsableSqFt(z domain.RoofZone) float64 {
	// 12% clearance/access reserve plus declared obstacles.
	return math.Max(0, z.GrossSqFt*(1-clamp(z.ObstaclePct/100, 0, .85))*.88)
}

func zoneShadeFactor(s domain.SunlightWindow) float64 {
	// Approximate daily production share: morning 25%, midday 50%, evening 25%.
	return clamp(.25*clamp(s.Morning, 0, 1)+.5*clamp(s.Midday, 0, 1)+.25*clamp(s.Evening, 0, 1), .15, 1)
}

func Calculate(resource domain.SolarResource, roof domain.RoofProfile, targetKW float64) domain.SolarScenario {
	var usable, weightedFactor float64
	for _, z := range roof.Zones {
		u := ZoneUsableSqFt(z)
		usable += u
		weightedFactor += u * orientationFactor(z.Orientation, z.TiltDegrees) * zoneShadeFactor(z.Sun)
	}
	factor := .80 // balance-of-system performance ratio
	if usable > 0 {
		factor *= weightedFactor / usable
	}
	capacity := math.Min(targetKW, usable/SqFtPerKW)
	capacity = math.Floor(capacity*10) / 10
	s := domain.SolarScenario{CapacityKW: capacity, RequiredSqFt: capacity * SqFtPerKW, UsableSqFt: usable, Confidence: "medium"}
	for i, psh := range resource.MonthlyPeakSunHours {
		days := [...]float64{31, 28.25, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}[i]
		s.MonthlyKWh[i] = capacity * psh * days * factor
		s.AnnualKWh += s.MonthlyKWh[i]
	}
	s.ConservativeKWh = s.AnnualKWh * .85
	s.OptimisticKWh = s.AnnualKWh * 1.08
	s.Assumptions = []string{"100 sq.ft of usable roof per kW", "20% system and temperature losses", "User-reported time-of-day shade"}
	if capacity < targetKW {
		s.Warnings = append(s.Warnings, "Usable roof area limits the recommended system size.")
	}
	if roof.Ownership == "shared" {
		s.Warnings = append(s.Warnings, "Written terrace/association permission may be required.")
	}
	return s
}
