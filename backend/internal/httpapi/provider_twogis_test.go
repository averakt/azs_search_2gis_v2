package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"azs_search_2gis_v2/backend/internal/benzin"
	"azs_search_2gis_v2/backend/internal/cache"
	"azs_search_2gis_v2/backend/internal/model"
	"azs_search_2gis_v2/backend/internal/osm"
)

func boolPtr(b bool) *bool {
	return &b
}

type overpassResponse struct {
	Elements []struct {
		Type string  `json:"type"`
		ID   int64   `json:"id"`
		Lat  float64 `json:"lat"`
		Lon  float64 `json:"lon"`
		Tags struct {
			Name string `json:"name"`
		} `json:"tags"`
	} `json:"elements"`
}

func newOverpassStation(id int64, name string, lat, lon float64) overpassResponse {
	elem := struct {
		Type string  `json:"type"`
		ID   int64   `json:"id"`
		Lat  float64 `json:"lat"`
		Lon  float64 `json:"lon"`
		Tags struct {
			Name string `json:"name"`
		} `json:"tags"`
	}{
		Type: "node", ID: id, Lat: lat, Lon: lon,
		Tags: struct {
			Name string `json:"name"`
		}{Name: name},
	}
	return overpassResponse{Elements: []struct {
		Type string  `json:"type"`
		ID   int64   `json:"id"`
		Lat  float64 `json:"lat"`
		Lon  float64 `json:"lon"`
		Tags struct {
			Name string `json:"name"`
		} `json:"tags"`
	}{elem}}
}

type searchStation struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Brand   string  `json:"brand"`
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

func TestTwoGisProvider_SearchStations_BenzinFallbackToOSM(t *testing.T) {
	benzinCalled := false
	benzinServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		benzinCalled = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer benzinServer.Close()

	overpassCalled := false
	overpassServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		overpassCalled = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newOverpassStation(1, "OSM АЗС", 55.75, 37.61))
	}))
	defer overpassServer.Close()

	c := cache.New(10 * time.Minute)
	benzinClient := benzin.NewClient(benzinServer.URL, 5*time.Second)
	overpassClient := osm.NewOverpassClient(overpassServer.URL, 5*time.Second)

	provider := NewTwoGisProvider(nil, nil, benzinClient, overpassClient, nil, c, 5*time.Second, slog.Default())

	stations, err := provider.SearchStations(context.Background(), model.Location{Lat: 55.75, Lon: 37.61}, 3000, "", "2gis")
	if err != nil {
		t.Fatalf("SearchStations() error = %v", err)
	}
	if len(stations) != 1 {
		t.Fatalf("expected 1 station from fallback, got %d", len(stations))
	}
	if stations[0].ID != "osm_1" {
		t.Errorf("station.ID = %q, want %q", stations[0].ID, "osm_1")
	}
	if !benzinCalled {
		t.Error("benzin should have been called first")
	}
	if !overpassCalled {
		t.Error("overpass should have been called as fallback")
	}
}

func TestTwoGisProvider_SearchStations_OSMProvider(t *testing.T) {
	overpassServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newOverpassStation(42, "OSM АЗС", 55.75, 37.61))
	}))
	defer overpassServer.Close()

	c := cache.New(10 * time.Minute)
	overpassClient := osm.NewOverpassClient(overpassServer.URL, 5*time.Second)

	provider := NewTwoGisProvider(nil, nil, nil, overpassClient, nil, c, 5*time.Second, slog.Default())

	stations, err := provider.SearchStations(context.Background(), model.Location{Lat: 55.75, Lon: 37.61}, 3000, "", "osm")
	if err != nil {
		t.Fatalf("SearchStations() error = %v", err)
	}
	if len(stations) != 1 {
		t.Fatalf("expected 1 station, got %d", len(stations))
	}
	if stations[0].ID != "osm_42" {
		t.Errorf("station.ID = %q, want %q", stations[0].ID, "osm_42")
	}
}

func TestTwoGisProvider_CacheIntegration(t *testing.T) {
	serverCallCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCallCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]searchStation{
			{ID: "1", Name: "АЗС-1", Brand: "Лукойл", Address: "ул. 1", Lat: 55.75, Lng: 37.61},
		})
	}))
	defer server.Close()

	c := cache.New(10 * time.Minute)
	benzinClient := benzin.NewClient(server.URL, 5*time.Second)

	provider := NewTwoGisProvider(nil, nil, benzinClient, nil, nil, c, 5*time.Second, slog.Default())

	stations1, err := provider.SearchStations(context.Background(), model.Location{Lat: 55.75, Lon: 37.61}, 3000, "", "2gis")
	if err != nil {
		t.Fatalf("first SearchStations() error = %v", err)
	}
	if len(stations1) != 1 {
		t.Fatalf("expected 1 station, got %d", len(stations1))
	}
	firstCallCount := serverCallCount

	stations2, err := provider.SearchStations(context.Background(), model.Location{Lat: 55.75, Lon: 37.61}, 3000, "", "2gis")
	if err != nil {
		t.Fatalf("second SearchStations() error = %v", err)
	}
	if len(stations2) != 1 {
		t.Fatalf("expected 1 station, got %d", len(stations2))
	}

	if serverCallCount != firstCallCount {
		t.Errorf("server was called %d times, expected %d (should use cache)", serverCallCount, firstCallCount)
	}
}

func TestTwoGisProvider_SearchStations_WithFuelFilter(t *testing.T) {
	benzinServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/stations/search" {
			json.NewEncoder(w).Encode([]searchStation{
				{ID: "1", Name: "АЗС-1", Brand: "Лукойл", Address: "ул. 1", Lat: 55.75, Lng: 37.61},
			})
			return
		}
		json.NewEncoder(w).Encode([]struct {
			Station struct {
				ID             string   `json:"id"`
				Name           string   `json:"name"`
				Brand          string   `json:"brand"`
				Address        string   `json:"address"`
				Lat            float64  `json:"lat"`
				Lng            float64  `json:"lng"`
				FuelAssortment []string `json:"fuel_assortment"`
			} `json:"station"`
			FuelStatuses []struct {
				StationID   string `json:"station_id"`
				FuelType    string `json:"fuel_type"`
				Available   *bool  `json:"available"`
				QueueLevel  string `json:"queue_level"`
				LimitLiters int    `json:"limit_liters"`
			} `json:"fuel_statuses"`
			Prices []struct {
				StationID string  `json:"station_id"`
				FuelType  string  `json:"fuel_type"`
				Price     float64 `json:"price"`
			} `json:"prices"`
			QueueLevel  string `json:"queue_level"`
			LimitLiters int    `json:"limit_liters"`
			Status      string `json:"status"`
		}{
			{
				Station: struct {
					ID             string   `json:"id"`
					Name           string   `json:"name"`
					Brand          string   `json:"brand"`
					Address        string   `json:"address"`
					Lat            float64  `json:"lat"`
					Lng            float64  `json:"lng"`
					FuelAssortment []string `json:"fuel_assortment"`
				}{
					ID: "1", Name: "АЗС-1", Brand: "Лукойл", Address: "ул. 1",
					Lat: 55.75, Lng: 37.61, FuelAssortment: []string{"AI_92", "AI_95"},
				},
				FuelStatuses: []struct {
					StationID   string `json:"station_id"`
					FuelType    string `json:"fuel_type"`
					Available   *bool  `json:"available"`
					QueueLevel  string `json:"queue_level"`
					LimitLiters int    `json:"limit_liters"`
				}{
					{StationID: "1", FuelType: "AI_92", Available: boolPtr(true)},
					{StationID: "1", FuelType: "AI_95", Available: boolPtr(true)},
				},
				Prices: []struct {
					StationID string  `json:"station_id"`
					FuelType  string  `json:"fuel_type"`
					Price     float64 `json:"price"`
				}{
					{StationID: "1", FuelType: "AI_92", Price: 50},
					{StationID: "1", FuelType: "AI_95", Price: 55},
				},
				QueueLevel: "NONE", LimitLiters: 50, Status: "active",
			},
		})
	}))
	defer benzinServer.Close()

	c := cache.New(10 * time.Minute)
	benzinClient := benzin.NewClient(benzinServer.URL, 5*time.Second)

	provider := NewTwoGisProvider(nil, nil, benzinClient, nil, nil, c, 5*time.Second, slog.Default())

	t.Run("no fuel filter returns all", func(t *testing.T) {
		stations, err := provider.SearchStations(context.Background(), model.Location{Lat: 55.75, Lon: 37.61}, 3000, "", "2gis")
		if err != nil {
			t.Fatalf("SearchStations() error = %v", err)
		}
		if len(stations) != 1 {
			t.Errorf("expected 1 station, got %d", len(stations))
		}
	})

	t.Run("fuel filter 92 returns station", func(t *testing.T) {
		stations, err := provider.SearchStations(context.Background(), model.Location{Lat: 55.75, Lon: 37.61}, 3000, "92", "2gis")
		if err != nil {
			t.Fatalf("SearchStations() error = %v", err)
		}
		if len(stations) != 1 {
			t.Errorf("expected 1 station, got %d", len(stations))
		}
	})

	t.Run("fuel filter dt returns empty", func(t *testing.T) {
		stations, err := provider.SearchStations(context.Background(), model.Location{Lat: 55.75, Lon: 37.61}, 3000, "dt", "2gis")
		if err != nil {
			t.Fatalf("SearchStations() error = %v", err)
		}
		if len(stations) != 0 {
			t.Errorf("expected 0 stations, got %d", len(stations))
		}
	})
}
