package engine

import (
	"github.com/Forking-Around/SolarSense/internal/battery"
	"github.com/Forking-Around/SolarSense/internal/domain"
	"github.com/Forking-Around/SolarSense/internal/policy"
	"github.com/Forking-Around/SolarSense/internal/solar"
	"math"
)

const Version = "0.1.0"

func Assess(in domain.AssessmentInput, resource domain.SolarResource) domain.AssessmentReport {
	units := in.MonthlyUnitsKWh
	if units <= 0 && in.GridRateINR > 0 {
		units = math.Max(0, in.MonthlyBillINR-in.FixedChargeINR) / in.GridRateINR
	}
	target := math.Min(10, math.Max(1, units*12/1450))
	target = math.Ceil(target*10) / 10
	s := solar.Calculate(resource, in.Roof, target)
	selfRatio := math.Min(.90, .35+.55*in.DaytimeUsePct/100)
	s.SelfConsumedKWh = math.Min(s.AnnualKWh*selfRatio, units*12)
	s.ExportedKWh = s.AnnualKWh - s.SelfConsumedKWh
	s.GrossCostINR = s.CapacityKW * 60000
	s.SubsidyINR = policy.IndiaResidentialSubsidy(s.CapacityKW)
	s.NetCostINR = s.GrossCostINR - s.SubsidyINR
	s.AnnualSavingsINR = s.SelfConsumedKWh*in.GridRateINR + s.ExportedKWh*in.ExportRateINR
	if s.AnnualSavingsINR > 0 {
		s.PaybackYears = s.NetCostINR / s.AnnualSavingsINR
	} else {
		s.PaybackYears = 99
	}
	s.NetBenefit25YINR = s.AnnualSavingsINR*22.5 - s.NetCostINR - 25000
	s.RemainingMonthlyBillINR = math.Max(in.FixedChargeINR, in.MonthlyBillINR-s.AnnualSavingsINR/12)
	s.Verdict = "Caution"
	if s.CapacityKW < 1 {
		s.Verdict = "Infeasible"
	} else if s.PaybackYears <= 5 {
		s.Verdict = "Strong Yes"
	} else if s.PaybackYears <= 8 {
		s.Verdict = "Yes"
	} else if s.NetBenefit25YINR <= 0 {
		s.Verdict = "No financial case"
	}
	if in.SanctionedLoadKW > 0 && s.CapacityKW > in.SanctionedLoadKW {
		s.Warnings = append(s.Warnings, "Recommended capacity exceeds the entered sanctioned load; confirm the current DISCOM rule or load enhancement process.")
	}
	return domain.AssessmentReport{Input: in, Solar: s, Battery: battery.Calculate(in.Outages, in.GridRateINR, in.ExportRateINR), SolarSource: resource, EngineVersion: Version}
}
