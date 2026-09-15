package solar

import (
	"testing"

	"github.com/Forking-Around/SolarSense/internal/domain"
)

func TestTimeOfDayShadeReducesGeneration(t *testing.T) {
	r := IndiaScreeningResource()
	clear := domain.RoofProfile{Zones: []domain.RoofZone{{GrossSqFt: 500, Orientation: "south", TiltDegrees: 10, Sun: domain.SunlightWindow{Morning: 1, Midday: 1, Evening: 1}}}}
	shaded := domain.RoofProfile{Zones: []domain.RoofZone{{GrossSqFt: 500, Orientation: "south", TiltDegrees: 10, Sun: domain.SunlightWindow{Morning: .2, Midday: 1, Evening: .2}}}}
	a, b := Calculate(r, clear, 3), Calculate(r, shaded, 3)
	if b.AnnualKWh >= a.AnnualKWh {
		t.Fatalf("shaded roof %.0f should generate less than clear roof %.0f", b.AnnualKWh, a.AnnualKWh)
	}
}

func TestObstacleAndClearanceLimitCapacity(t *testing.T) {
	r := IndiaScreeningResource()
	roof := domain.RoofProfile{Zones: []domain.RoofZone{{GrossSqFt: 300, ObstaclePct: 50, Sun: domain.SunlightWindow{Morning: 1, Midday: 1, Evening: 1}}}}
	s := Calculate(r, roof, 3)
	if s.CapacityKW >= 2 {
		t.Fatalf("capacity %.1f ignores unusable roof area", s.CapacityKW)
	}
}
