package geocode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"azs_search_2gis_v2/backend/internal/model"
)

var ErrNotFound = errors.New("not found")

type TwoGisGeocoder struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

type TwoGisGeocodeResponse struct {
	Meta struct {
		ApiVersion string `json:"api_version"`
		Code       int    `json:"code"`
	} `json:"meta"`
	Result struct {
		Items []struct {
			Point struct {
				Lon float64 `json:"lon"`
				Lat float64 `json:"lat"`
			} `json:"point"`
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			Subtype  string `json:"subtype"`
			Type     string `json:"type"`
		} `json:"items"`
		Total int `json:"total"`
	} `json:"result"`
}

func NewTwoGisGeocoder(apiKey string, timeout time.Duration) *TwoGisGeocoder {
	return &TwoGisGeocoder{
		apiKey:  apiKey,
		baseURL: "https://catalog.api.2gis.com/3.0/items/geocode",
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (g *TwoGisGeocoder) geocodeInternal(ctx context.Context, q string) (TwoGisGeocodeResponse, error) {
	if g.apiKey == "" {
		return TwoGisGeocodeResponse{}, fmt.Errorf("2GIS API key not configured")
	}

	params := url.Values{}
	params.Set("q", q)
	params.Set("fields", "items.point")
	params.Set("key", g.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", g.baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return TwoGisGeocodeResponse{}, err
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return TwoGisGeocodeResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TwoGisGeocodeResponse{}, err
	}

	var geoResp TwoGisGeocodeResponse
	if err := json.Unmarshal(body, &geoResp); err != nil {
		return TwoGisGeocodeResponse{}, err
	}
	return geoResp, nil
}

func (g *TwoGisGeocoder) Geocode(ctx context.Context, q string) (model.Location, error) {
	geoResp, err := g.geocodeInternal(ctx, q)
	if err != nil {
		return model.Location{}, err
	}

	if geoResp.Result.Total == 0 || len(geoResp.Result.Items) == 0 {
		return model.Location{}, ErrNotFound
	}

	item := geoResp.Result.Items[0]
	label := item.FullName
	if label == "" {
		label = item.Name
	}
	return model.Location{
		Lat:   item.Point.Lat,
		Lon:   item.Point.Lon,
		Label: label,
	}, nil
}

func (g *TwoGisGeocoder) Suggest(ctx context.Context, q string, limit int) ([]model.Location, error) {
	geoResp, err := g.geocodeInternal(ctx, q)
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit > len(geoResp.Result.Items) {
		limit = len(geoResp.Result.Items)
	}

	results := make([]model.Location, 0, limit)
	for i := 0; i < limit; i++ {
		item := geoResp.Result.Items[i]
		label := item.FullName
		if label == "" {
			label = item.Name
		}
		results = append(results, model.Location{
			Lat:   item.Point.Lat,
			Lon:   item.Point.Lon,
			Label: label,
		})
	}
	return results, nil
}
