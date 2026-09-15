package solar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Forking-Around/SolarSense/internal/domain"
)

type NASAClient struct {
	HTTP     *http.Client
	Endpoint string
}

func (n NASAClient) Resource(ctx context.Context, lat, lon float64) (domain.SolarResource, error) {
	if n.Endpoint == "" {
		return domain.SolarResource{}, fmt.Errorf("NASA POWER endpoint is not configured")
	}
	q := url.Values{"parameters": {"ALLSKY_SFC_SW_DWN"}, "community": {"RE"}, "longitude": {strconv.FormatFloat(lon, 'f', 5, 64)}, "latitude": {strconv.FormatFloat(lat, 'f', 5, 64)}, "format": {"JSON"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.Endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return domain.SolarResource{}, err
	}
	client := n.HTTP
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return domain.SolarResource{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return domain.SolarResource{}, fmt.Errorf("NASA POWER returned %s", resp.Status)
	}
	var payload struct {
		Properties struct {
			Parameter struct {
				GHI map[string]float64 `json:"ALLSKY_SFC_SW_DWN"`
			} `json:"parameter"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return domain.SolarResource{}, err
	}
	months := [...]string{"JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}
	r := domain.SolarResource{Source: "NASA POWER climatology: ALLSKY_SFC_SW_DWN", Version: "climatology", RetrievedAt: time.Now().UTC()}
	for i, m := range months {
		v, ok := payload.Properties.Parameter.GHI[m]
		if !ok || v <= 0 {
			return domain.SolarResource{}, fmt.Errorf("missing solar value for %s", m)
		}
		r.MonthlyPeakSunHours[i] = v
	}
	return r, nil
}

func IndiaScreeningResource() domain.SolarResource {
	return domain.SolarResource{MonthlyPeakSunHours: [12]float64{5.2, 5.8, 6.2, 6.1, 5.8, 4.7, 4.2, 4.3, 4.8, 5.0, 4.9, 4.8}, Source: "India screening profile — live NASA data unavailable", Version: "screening-v1", RetrievedAt: time.Now().UTC()}
}
