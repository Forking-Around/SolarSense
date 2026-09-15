package battery

import (
	"testing"

	"github.com/Forking-Around/SolarSense/internal/domain"
)

func TestOutageExperienceChangesBackupVerdict(t *testing.T) {
	rare := Calculate(domain.OutageProfile{FrequencyPerMonth: 1, HoursPerOutage: 1, EssentialWatts: 500, DesiredHours: 4}, 8, 2.5)
	frequent := Calculate(domain.OutageProfile{FrequencyPerMonth: 12, HoursPerOutage: 2, EssentialWatts: 500, DesiredHours: 4}, 8, 2.5)
	if rare.BackupVerdict == frequent.BackupVerdict {
		t.Fatalf("expected outage experience to change recommendation: %q", rare.BackupVerdict)
	}
	if frequent.BackupHours < 3.9 {
		t.Fatalf("expected requested backup, got %.1f hours", frequent.BackupHours)
	}
}
