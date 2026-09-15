package domain

import "time"

type LocationContext struct {
	Name, Country, State, PostalCode, Utility string
	Latitude, Longitude                       float64
}

type SunlightWindow struct {
	Morning, Midday, Evening float64 // visible sun fraction, 0..1
}

type RoofZone struct {
	Name        string
	GrossSqFt   float64
	ObstaclePct float64
	Orientation string
	TiltDegrees float64
	Sun         SunlightWindow
}

type RoofProfile struct {
	Type, Ownership string
	Zones           []RoofZone
}

type OutageProfile struct {
	FrequencyPerMonth float64
	HoursPerOutage    float64
	EssentialWatts    float64
	DesiredHours      float64
}

type AssessmentInput struct {
	Location         LocationContext
	MonthlyBillINR   float64
	MonthlyUnitsKWh  float64
	DaytimeUsePct    float64
	Roof             RoofProfile
	Outages          OutageProfile
	GridRateINR      float64
	ExportRateINR    float64
	FixedChargeINR   float64
	SanctionedLoadKW float64
}

type SolarResource struct {
	MonthlyPeakSunHours [12]float64
	Latitude            float64
	Source, Version     string
	RetrievedAt         time.Time
}

type SolarScenario struct {
	CapacityKW, RequiredSqFt, UsableSqFt      float64
	AnnualKWh, SelfConsumedKWh, ExportedKWh   float64
	ConservativeKWh, OptimisticKWh            float64
	GrossCostINR, SubsidyINR, NetCostINR      float64
	AnnualSavingsINR, RemainingMonthlyBillINR float64
	PaybackYears, NetBenefit25YINR            float64
	MonthlyKWh                                [12]float64
	Verdict, Confidence                       string
	PolicyVersion, PolicySourceURL            string
	Warnings, Assumptions                     []string
}

type BatteryScenario struct {
	CapacityKWh, UsableKWh, BackupHours, IncrementalCostINR float64
	AnnualSavingsINR, PaybackYears                          float64
	SavingsVerdict, BackupVerdict                           string
	Explanation                                             string
}

type AssessmentReport struct {
	Input         AssessmentInput
	Solar         SolarScenario
	Battery       BatteryScenario
	SolarSource   SolarResource
	EngineVersion string
}

type BillExtraction struct {
	Utility, BillingPeriod, TariffCategory string
	UnitsKWh, Amount, SanctionedLoadKW     float64
	Confidence                             map[string]float64
}

type RoofPhotoObservation struct {
	RoofType            string             `json:"roof_type"`
	VisibleObstacles    []string           `json:"visible_obstacles"`
	ObstacleEstimatePct float64            `json:"obstacle_estimate_pct"`
	PartialShadeVisible bool               `json:"partial_shade_visible"`
	HasScaleReference   bool               `json:"has_scale_reference"`
	Confidence          map[string]float64 `json:"confidence"`
}

type PolicyPack struct {
	Jurisdiction, Utility, Version, SourceURL, ReviewStatus string
	EffectiveFrom                                           time.Time
}
type KnowledgeDocument struct{ ID, Title, Content, Language, CountryCode, State, Utility, Topic, SourceURL string }
