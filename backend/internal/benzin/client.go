package benzin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"azs_search_2gis_v2/backend/internal/model"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type SearchStation struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Brand   string  `json:"brand"`
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

type StationDetail struct {
	Station      StationInfo  `json:"station"`
	FuelStatuses []FuelStatus `json:"fuel_statuses,omitempty"`
	Prices       []PriceInfo  `json:"prices,omitempty"`
	QueueLevel   string       `json:"queue_level"`
	LimitLiters  int          `json:"limit_liters"`
	Status       string       `json:"status"`
}

type StationInfo struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Brand          string   `json:"brand"`
	Address        string   `json:"address"`
	Lat            float64  `json:"lat"`
	Lng            float64  `json:"lng"`
	FuelAssortment []string `json:"fuel_assortment"`
}

type FuelStatus struct {
	StationID   string `json:"station_id"`
	FuelType    string `json:"fuel_type"`
	Available   *bool  `json:"available"`
	QueueLevel  string `json:"queue_level"`
	LimitLiters int    `json:"limit_liters"`
}

type PriceInfo struct {
	StationID string  `json:"station_id"`
	FuelType  string  `json:"fuel_type"`
	Price     float64 `json:"price"`
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) GetStations(ctx context.Context, lat, lon float64, radius int, fuel string) ([]model.Station, error) {
	ids, err := c.searchStationIDs(ctx, lat, lon, radius)
	if err != nil {
		return nil, fmt.Errorf("search stations failed: %w", err)
	}

	if len(ids) == 0 {
		return []model.Station{}, nil
	}

	return c.getStationsDetails(ctx, ids, fuel)
}

func (c *Client) searchStationIDs(ctx context.Context, lat, lon float64, radius int) ([]string, error) {
	bbox := c.calculateBBox(lat, lon, radius)

	params := url.Values{}
	params.Set("q", "АЗС")
	params.Set("minLat", fmt.Sprintf("%.6f", bbox[1]))
	params.Set("maxLat", fmt.Sprintf("%.6f", bbox[3]))
	params.Set("minLon", fmt.Sprintf("%.6f", bbox[0]))
	params.Set("maxLon", fmt.Sprintf("%.6f", bbox[2]))

	reqURL := c.baseURL + "/api/v1/stations/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("benzin search: status %d, body=%s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var stations []SearchStation
	if err := json.Unmarshal(body, &stations); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	ids := make([]string, 0, len(stations))
	for _, s := range stations {
		ids = append(ids, s.ID)
	}

	return ids, nil
}

func (c *Client) getStationsDetails(ctx context.Context, ids []string, fuel string) ([]model.Station, error) {
	if len(ids) == 0 {
		return []model.Station{}, nil
	}

	reqURL := fmt.Sprintf("%s/api/v1/stations/by-ids?ids=%s", c.baseURL, strings.Join(ids, ","))

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("benzin details: status %d, body=%s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var details []StationDetail
	if err := json.Unmarshal(body, &details); err != nil {
		return nil, err
	}

	stations := make([]model.Station, 0, len(details))
	for _, d := range details {
		station := c.mapStation(d, fuel)
		if station != nil {
			stations = append(stations, *station)
		}
	}

	return stations, nil
}

func (c *Client) calculateBBox(lat, lon float64, radius int) [4]float64 {
	latDeg := float64(radius) / 111000.0
	lonDeg := float64(radius) / (111000.0 * math.Cos(lat*math.Pi/180.0))

	return [4]float64{
		lon - lonDeg,
		lat - latDeg,
		lon + lonDeg,
		lat + latDeg,
	}
}

func (c *Client) mapStation(d StationDetail, filterFuel string) *model.Station {
	if filterFuel != "" && !c.hasFuelType(d.Station.FuelAssortment, filterFuel) {
		return nil
	}

	fuels := make([]model.Fuel, 0)
	priceMap := make(map[string]float64)
	for _, p := range d.Prices {
		priceMap[p.FuelType] = p.Price
	}

	for _, fs := range d.FuelStatuses {
		fuelType := c.mapFuelType(fs.FuelType)
		if fuelType == "" {
			continue
		}

		avail := "yes"
		if fs.Available != nil && !*fs.Available {
			avail = "no"
		}

		price := priceMap[fs.FuelType]
		fuels = append(fuels, model.Fuel{
			Type:     fuelType,
			Avail:    avail,
			Price:    price,
			Currency: "RUB",
		})
	}

	// If no fuel statuses from API, fall back to assortment
	if len(fuels) == 0 {
		for _, fa := range d.Station.FuelAssortment {
			fuelType := c.mapFuelType(fa)
			if fuelType == "" {
				continue
			}
			fuels = append(fuels, model.Fuel{
				Type:     fuelType,
				Avail:    "unknown",
				Price:    0,
				Currency: "RUB",
			})
		}
	}

	queueLevel := d.QueueLevel
	if queueLevel == "" {
		queueLevel = "NONE"
	}

	canJerrycan := true
	if d.LimitLiters > 0 {
		canJerrycan = false
	}

	return &model.Station{
		ID:      d.Station.ID,
		Name:    d.Station.Name,
		Brand:   d.Station.Brand,
		Address: d.Station.Address,
		Lat:     d.Station.Lat,
		Lon:     d.Station.Lng,
		Fuels:   fuels,
		Queue: model.Queue{
			Level:      c.mapQueueLevel(queueLevel),
			EstWaitMin: c.queueToWait(queueLevel),
		},
		Limits: &model.Limits{
			MaxLiters:   d.LimitLiters,
			CanJerrycan: canJerrycan,
		},
		UpdatedAt: time.Time{},
		Source:    "2gis-benzin",
	}
}

func (c *Client) mapFuelType(fuelType string) string {
	switch fuelType {
	case "AI_92":
		return "АИ-92"
	case "AI_95":
		return "АИ-95"
	case "AI_98":
		return "АИ-98"
	case "AI_100":
		return "АИ-100"
	case "DT":
		return "ДТ"
	case "GAS":
		return "Газ"
	default:
		return ""
	}
}

func (c *Client) hasFuelType(assortment []string, filter string) bool {
	fuelMap := map[string][]string{
		"92":  {"AI_92"},
		"95":  {"AI_95"},
		"98":  {"AI_98"},
		"100": {"AI_100"},
		"dt":  {"DT"},
		"gas": {"GAS"},
	}

	types, ok := fuelMap[filter]
	if !ok {
		return true
	}

	for _, a := range assortment {
		for _, t := range types {
			if a == t {
				return true
			}
		}
	}
	return false
}

func (c *Client) queueToWait(level string) int {
	switch strings.ToUpper(level) {
	case "NONE":
		return 0
	case "UP_TO_10":
		return 10
	case "UP_TO_25":
		return 25
	case "UP_TO_45":
		return 45
	case "MORE_45":
		return 60
	default:
		return 0
	}
}

func (c *Client) mapQueueLevel(level string) string {
	switch strings.ToUpper(level) {
	case "NONE":
		return "none"
	case "UP_TO_10":
		return "small"
	case "UP_TO_25":
		return "medium"
	case "UP_TO_45":
		return "large"
	case "MORE_45":
		return "very_large"
	default:
		return "unknown"
	}
}
