package osm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"azs_search_2gis_v2/backend/internal/model"
)

type OverpassClient struct {
	endpoint   string
	httpClient *http.Client
}

type OverpassResponse struct {
	Elements []OverpassElement `json:"elements"`
}

type OverpassElement struct {
	Type string       `json:"type"`
	ID   int64        `json:"id"`
	Lat  float64      `json:"lat"`
	Lon  float64      `json:"lon"`
	Tags OverpassTags `json:"tags"`
}

type OverpassTags struct {
	Name         string `json:"name,omitempty"`
	Brand        string `json:"brand,omitempty"`
	Address      string `json:"addr:full,omitempty"`
	Street       string `json:"addr:street,omitempty"`
	Housenumber  string `json:"addr:housenumber,omitempty"`
	FuelDiesel   string `json:"fuel:diesel,omitempty"`
	FuelOctane76 string `json:"fuel:octane_76,omitempty"`
	FuelOctane80 string `json:"fuel:octane_80,omitempty"`
	FuelOctane91 string `json:"fuel:octane_91,omitempty"`
	FuelOctane92 string `json:"fuel:octane_92,omitempty"`
	FuelOctane93 string `json:"fuel:octane_93,omitempty"`
	FuelOctane95 string `json:"fuel:octane_95,omitempty"`
	FuelOctane98 string `json:"fuel:octane_98,omitempty"`
	FuelLPG      string `json:"fuel:lpg,omitempty"`
	FuelCNG      string `json:"fuel:cng,omitempty"`
}

func NewOverpassClient(endpoint string, timeout time.Duration) *OverpassClient {
	if endpoint == "" {
		endpoint = "https://overpass-api.de/api/interpreter"
	}
	return &OverpassClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *OverpassClient) GetStations(ctx context.Context, lat, lon float64, radius int, fuel string) ([]model.Station, error) {
	query := c.buildOverpassQuery(lat, lon, radius, fuel)

	reqURL := c.endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(query))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "azs_search_2gis_v2/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("overpass request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK || len(body) > 0 && body[0] == '<' {
		return nil, fmt.Errorf("overpass returned %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	var overpassResp OverpassResponse
	if err := json.Unmarshal(body, &overpassResp); err != nil {
		return nil, err
	}

	stations := make([]model.Station, 0, len(overpassResp.Elements))
	for _, elem := range overpassResp.Elements {
		if station := c.mapStation(elem); station != nil {
			stations = append(stations, *station)
		}
	}

	return stations, nil
}

func (c *OverpassClient) buildOverpassQuery(lat, lon float64, radius int, fuel string) string {
	fuelFilter := ""
	if fuel != "" {
		switch fuel {
		case "92", "95", "98", "100":
			fuelFilter = fmt.Sprintf("[\"fuel:octane_%s\"]", fuel)
		case "dt":
			fuelFilter = "[\"fuel:diesel\"]"
		case "gas":
			fuelFilter = "[\"fuel:lpg\"|\"fuel:cng\"]"
		}
	}

	query := fmt.Sprintf(`
		[out:json][timeout:25];
		(
		  node["amenity"="fuel"]%s(around:%d,%f,%f);
		  way["amenity"="fuel"]%s(around:%d,%f,%f);
		);
		out center;
	`, fuelFilter, radius, lat, lon, fuelFilter, radius, lat, lon)

	return query
}

func (c *OverpassClient) mapStation(elem OverpassElement) *model.Station {
	tags := elem.Tags

	address := tags.Address
	if address == "" && tags.Street != "" {
		if tags.Housenumber != "" {
			address = tags.Street + ", " + tags.Housenumber
		} else {
			address = tags.Street
		}
	}

	name := tags.Name
	if name == "" {
		name = tags.Brand
	}
	if name == "" {
		name = "АЗС"
	}

	brand := tags.Brand
	if brand == "" {
		brand = name
	}

	fuels := make([]model.Fuel, 0)

	if tags.FuelOctane92 != "" {
		fuels = append(fuels, model.Fuel{Type: "АИ-92", Avail: "yes", Price: 0, Currency: "RUB"})
	}
	if tags.FuelOctane95 != "" {
		fuels = append(fuels, model.Fuel{Type: "АИ-95", Avail: "yes", Price: 0, Currency: "RUB"})
	}
	if tags.FuelOctane98 != "" {
		fuels = append(fuels, model.Fuel{Type: "АИ-98", Avail: "yes", Price: 0, Currency: "RUB"})
	}
	if tags.FuelDiesel != "" {
		fuels = append(fuels, model.Fuel{Type: "ДТ", Avail: "yes", Price: 0, Currency: "RUB"})
	}
	if tags.FuelLPG != "" || tags.FuelCNG != "" {
		fuels = append(fuels, model.Fuel{Type: "Газ", Avail: "yes", Price: 0, Currency: "RUB"})
	}

	lat := elem.Lat
	lon := elem.Lon

	return &model.Station{
		ID:      fmt.Sprintf("osm_%d", elem.ID),
		Name:    name,
		Brand:   brand,
		Address: address,
		Lat:     lat,
		Lon:     lon,
		Fuels:   fuels,
		Queue: model.Queue{
			Level:      "unknown",
			EstWaitMin: 0,
		},
		Limits:    nil,
		UpdatedAt: time.Time{},
		Source:    "osm-overpass",
	}
}

func (c *OverpassClient) GetStationsWithRetry(ctx context.Context, lat, lon float64, radius int, fuel string, maxRetries int) ([]model.Station, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		stations, err := c.GetStations(ctx, lat, lon, radius, fuel)
		if err == nil {
			return stations, nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second * time.Duration(i+1)):
		}
	}
	return nil, lastErr
}
