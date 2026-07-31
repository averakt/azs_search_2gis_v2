package geocode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"azs_search_2gis_v2/backend/internal/model"
)

type YandexGeocoder struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

type YandexGeocodeResponse struct {
	Response struct {
		GeoObjectCollection struct {
			MetaDataProperty struct {
				Name string `json:"name"`
			} `json:"metaDataProperty"`
			Feature []struct {
				Geometry struct {
					Coordinates []float64 `json:"coordinates"`
				} `json:"geometry"`
			} `json:"feature"`
		} `json:"GeoObjectCollection"`
	} `json:"response"`
}

func NewYandexGeocoder(apiKey string, timeout time.Duration) *YandexGeocoder {
	return &YandexGeocoder{
		apiKey:  apiKey,
		baseURL: "https://geocode-maps.yandex.ru/1.x/",
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (g *YandexGeocoder) Geocode(ctx context.Context, q string) (model.Location, error) {
	if g.apiKey == "" {
		return model.Location{}, fmt.Errorf("Yandex API key not configured")
	}

	params := url.Values{}
	params.Set("geocode", q)
	params.Set("format", "json")
	params.Set("apikey", g.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", g.baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return model.Location{}, err
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return model.Location{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.Location{}, err
	}

	var geoResp YandexGeocodeResponse
	if err := json.Unmarshal(body, &geoResp); err != nil {
		return model.Location{}, err
	}

	if len(geoResp.Response.GeoObjectCollection.Feature) == 0 {
		return model.Location{}, fmt.Errorf("no results found for query: %s", q)
	}

	feature := geoResp.Response.GeoObjectCollection.Feature[0]
	coords := feature.Geometry.Coordinates
	if len(coords) < 2 {
		return model.Location{}, fmt.Errorf("invalid coordinates in response")
	}

	return model.Location{
		Lat:   coords[1],
		Lon:   coords[0],
		Label: geoResp.Response.GeoObjectCollection.MetaDataProperty.Name,
	}, nil
}
