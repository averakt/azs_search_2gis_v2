package osm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type NominatimResult struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

type NominatimClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewNominatimClient(timeout time.Duration) *NominatimClient {
	return &NominatimClient{
		baseURL: "https://nominatim.openstreetmap.org",
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type SuggestResult struct {
	Label string  `json:"label"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
}

func (c *NominatimClient) Suggest(ctx context.Context, q string, limit int) ([]SuggestResult, error) {
	params := url.Values{}
	params.Set("q", q)
	params.Set("format", "json")
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("accept-language", "ru")

	reqURL := c.baseURL + "/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "azs_search_2gis_v2/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []NominatimResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, err
	}

	suggestions := make([]SuggestResult, 0, len(results))
	for _, r := range results {
		var lat, lon float64
		if _, err := fmt.Sscanf(r.Lat, "%f", &lat); err != nil {
			continue
		}
		if _, err := fmt.Sscanf(r.Lon, "%f", &lon); err != nil {
			continue
		}
		suggestions = append(suggestions, SuggestResult{
			Label: r.DisplayName,
			Lat:   lat,
			Lon:   lon,
		})
	}

	return suggestions, nil
}
