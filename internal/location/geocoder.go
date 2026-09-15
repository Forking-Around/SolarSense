package location

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

type Geocoder struct {
	Endpoint, APIKey string
	HTTP             *http.Client
}

func (g Geocoder) Enabled() bool { return g.Endpoint != "" && g.APIKey != "" }
func (g Geocoder) Search(ctx context.Context, query string) (domain.LocationContext, error) {
	if !g.Enabled() {
		return domain.LocationContext{}, fmt.Errorf("geocoding is not configured")
	}
	q := url.Values{"address": {query}, "key": {g.APIKey}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.Endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return domain.LocationContext{}, err
	}
	client := g.HTTP
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return domain.LocationContext{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return domain.LocationContext{}, fmt.Errorf("geocoder returned %s", resp.Status)
	}
	var p struct {
		Status  string `json:"status"`
		Results []struct {
			Formatted string `json:"formatted_address"`
			Geometry  struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
			Components []struct {
				LongName string   `json:"long_name"`
				Types    []string `json:"types"`
			} `json:"address_components"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return domain.LocationContext{}, err
	}
	if p.Status != "OK" || len(p.Results) == 0 {
		return domain.LocationContext{}, fmt.Errorf("location not found")
	}
	x := p.Results[0]
	out := domain.LocationContext{Name: x.Formatted, Latitude: x.Geometry.Location.Lat, Longitude: x.Geometry.Location.Lng}
	for _, c := range x.Components {
		for _, typ := range c.Types {
			switch typ {
			case "country":
				out.Country = c.LongName
			case "administrative_area_level_1":
				out.State = c.LongName
			case "postal_code":
				out.PostalCode = c.LongName
			}
		}
	}
	return out, nil
}
func ParseCoordinate(raw string) (float64, bool) {
	v, err := strconv.ParseFloat(raw, 64)
	return v, err == nil
}
