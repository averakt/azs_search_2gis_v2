package headless

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

type HeadlessStation struct {
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

type HeadlessClient struct {
	benzinBaseURL string
	httpClient    *http.Client
}

func NewHeadlessClient(benzinBaseURL string) *HeadlessClient {
	jar, _ := cookiejar.New(nil)

	return &HeadlessClient{
		benzinBaseURL: benzinBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
	}
}

func (h *HeadlessClient) GetStationsByBBox(ctx context.Context, bbox [4]float64) ([]string, error) {
	benzinURL := fmt.Sprintf("https://benzin.2gis.ru?bbox=%.4f,%.4f,%.4f,%.4f", bbox[0], bbox[1], bbox[2], bbox[3])

	req, err := http.NewRequestWithContext(ctx, "GET", benzinURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://2gis.ru/")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("headless request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var stations []HeadlessStation
	if err := json.Unmarshal(body, &stations); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	ids := make([]string, 0, len(stations))
	for _, s := range stations {
		ids = append(ids, s.Station.ID)
	}

	return ids, nil
}

func (h *HeadlessClient) GetStationsDetails(ctx context.Context, ids []string) ([]HeadlessStation, error) {
	if len(ids) == 0 {
		return []HeadlessStation{}, nil
	}

	idsStr := ""
	for i, id := range ids {
		if i > 0 {
			idsStr += ","
		}
		idsStr += id
	}

	apiURL := fmt.Sprintf("%s/api/v1/stations/by-ids?ids=%s", h.benzinBaseURL, url.PathEscape(idsStr))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://benzin.2gis.ru/")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var stations []HeadlessStation
	if err := json.Unmarshal(body, &stations); err != nil {
		return nil, err
	}

	return stations, nil
}
