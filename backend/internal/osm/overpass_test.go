package osm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBuildOverpassQuery(t *testing.T) {
	c := NewOverpassClient("", 5*time.Second)

	tests := []struct {
		name     string
		lat, lon float64
		radius   int
		fuel     string
		checks   []string
	}{
		{
			name: "no fuel filter",
			lat:  55.75, lon: 37.61, radius: 3000, fuel: "",
			checks: []string{`"amenity"="fuel"`, "around:3000", "[out:json]"},
		},
		{
			name: "dt fuel filter",
			lat:  55.0, lon: 37.0, radius: 1000, fuel: "dt",
			checks: []string{"fuel:diesel", "around:1000,55.000000,37.000000"},
		},
		{
			name: "gas fuel filter",
			lat:  55.0, lon: 37.0, radius: 5000, fuel: "gas",
			checks: []string{"fuel:lpg", "fuel:cng"},
		},
		{
			name: "octane filter",
			lat:  55.0, lon: 37.0, radius: 2000, fuel: "95",
			checks: []string{"fuel:octane_95"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := c.buildOverpassQuery(tt.lat, tt.lon, tt.radius, tt.fuel)
			for _, check := range tt.checks {
				if !strings.Contains(query, check) {
					t.Errorf("query should contain %q, got:\n%s", check, query)
				}
			}
		})
	}
}

func TestMapStation(t *testing.T) {
	c := NewOverpassClient("", 5*time.Second)

	t.Run("full station with address", func(t *testing.T) {
		elem := OverpassElement{
			Type: "node", ID: 12345,
			Lat: 55.75, Lon: 37.61,
			Tags: OverpassTags{
				Name:         "Лукойл АЗС",
				Brand:        "Лукойл",
				Address:      "Москва, ул. Тверская, 1",
				FuelOctane92: "yes",
				FuelOctane95: "yes",
				FuelDiesel:   "yes",
			},
		}

		station := c.mapStation(elem)
		if station == nil {
			t.Fatal("mapStation() returned nil")
		}

		if station.ID != "osm_12345" {
			t.Errorf("ID = %q, want %q", station.ID, "osm_12345")
		}
		if station.Name != "Лукойл АЗС" {
			t.Errorf("Name = %q", station.Name)
		}
		if station.Brand != "Лукойл" {
			t.Errorf("Brand = %q", station.Brand)
		}
		if station.Address != "Москва, ул. Тверская, 1" {
			t.Errorf("Address = %q", station.Address)
		}
		if station.Lat != 55.75 {
			t.Errorf("Lat = %f", station.Lat)
		}
		if station.Lon != 37.61 {
			t.Errorf("Lon = %f", station.Lon)
		}
		if station.Source != "osm-overpass" {
			t.Errorf("Source = %q", station.Source)
		}

		if len(station.Fuels) != 3 {
			t.Fatalf("expected 3 fuels, got %d", len(station.Fuels))
		}

		if station.Fuels[0].Type != "АИ-92" {
			t.Errorf("Fuel[0].Type = %q", station.Fuels[0].Type)
		}
		if station.Fuels[1].Type != "АИ-95" {
			t.Errorf("Fuel[1].Type = %q", station.Fuels[1].Type)
		}
		if station.Fuels[2].Type != "ДТ" {
			t.Errorf("Fuel[2].Type = %q", station.Fuels[2].Type)
		}

		for _, f := range station.Fuels {
			if f.Avail != "yes" {
				t.Errorf("Fuel %q Avail = %q, want %q", f.Type, f.Avail, "yes")
			}
			if f.Price != 0 {
				t.Errorf("Fuel %q Price = %f, want 0", f.Type, f.Price)
			}
		}

		if station.Queue.Level != "unknown" {
			t.Errorf("Queue.Level = %q", station.Queue.Level)
		}
		if station.Queue.EstWaitMin != 0 {
			t.Errorf("Queue.EstWaitMin = %d", station.Queue.EstWaitMin)
		}
		if station.Limits != nil {
			t.Error("Limits should be nil for OSM")
		}
	})

	t.Run("name fallback to brand", func(t *testing.T) {
		elem := OverpassElement{
			Type: "node", ID: 54321, Lat: 55.0, Lon: 37.0,
			Tags: OverpassTags{Brand: "Газпром"},
		}
		station := c.mapStation(elem)
		if station.Name != "Газпром" {
			t.Errorf("Name should fallback to brand, got %q", station.Name)
		}
	})

	t.Run("name fallback to default", func(t *testing.T) {
		elem := OverpassElement{
			Type: "node", ID: 99999, Lat: 55.0, Lon: 37.0,
			Tags: OverpassTags{},
		}
		station := c.mapStation(elem)
		if station.Name != "АЗС" {
			t.Errorf("Name should fallback to 'АЗС', got %q", station.Name)
		}
	})

	t.Run("address composition from street + housenumber", func(t *testing.T) {
		elem := OverpassElement{
			Type: "node", ID: 1, Lat: 55.0, Lon: 37.0,
			Tags: OverpassTags{
				Name: "АЗС", Street: "ул. Ленина", Housenumber: "10",
			},
		}
		station := c.mapStation(elem)
		if station.Address != "ул. Ленина, 10" {
			t.Errorf("Address = %q, want %q", station.Address, "ул. Ленина, 10")
		}
	})

	t.Run("address from street only", func(t *testing.T) {
		elem := OverpassElement{
			Type: "node", ID: 2, Lat: 55.0, Lon: 37.0,
			Tags: OverpassTags{Name: "АЗС", Street: "ул. Мира"},
		}
		station := c.mapStation(elem)
		if station.Address != "ул. Мира" {
			t.Errorf("Address = %q, want %q", station.Address, "ул. Мира")
		}
	})

	t.Run("brand fallback to name", func(t *testing.T) {
		elem := OverpassElement{
			Type: "node", ID: 3, Lat: 55.0, Lon: 37.0,
			Tags: OverpassTags{Name: "АЗС №1"},
		}
		station := c.mapStation(elem)
		if station.Brand != "АЗС №1" {
			t.Errorf("Brand should fallback to name, got %q", station.Brand)
		}
	})

	t.Run("all fuel types from tags", func(t *testing.T) {
		elem := OverpassElement{
			Type: "node", ID: 4, Lat: 55.0, Lon: 37.0,
			Tags: OverpassTags{
				Name:         "АЗС",
				FuelOctane92: "yes",
				FuelOctane95: "yes",
				FuelOctane98: "yes",
				FuelDiesel:   "yes",
				FuelLPG:      "yes",
				FuelCNG:      "yes",
			},
		}
		station := c.mapStation(elem)
		if len(station.Fuels) != 5 {
			t.Fatalf("expected 5 fuel types (lpg+cng → Газ), got %d", len(station.Fuels))
		}

		expectedTypes := []string{"АИ-92", "АИ-95", "АИ-98", "ДТ", "Газ"}
		for i, expType := range expectedTypes {
			if station.Fuels[i].Type != expType {
				t.Errorf("Fuel[%d].Type = %q, want %q", i, station.Fuels[i].Type, expType)
			}
		}
	})
}

func TestGetStations_Overpass(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(OverpassResponse{
				Elements: []OverpassElement{
					{
						Type: "node", ID: 1, Lat: 55.75, Lon: 37.61,
						Tags: OverpassTags{Name: "АЗС-1", FuelOctane92: "yes"},
					},
				},
			})
		}))
		defer server.Close()

		c := NewOverpassClient(server.URL, 5*time.Second)
		stations, err := c.GetStations(context.Background(), 55.75, 37.61, 3000, "")
		if err != nil {
			t.Fatalf("GetStations() error = %v", err)
		}
		if len(stations) != 1 {
			t.Fatalf("expected 1 station, got %d", len(stations))
		}
		if stations[0].ID != "osm_1" {
			t.Errorf("station.ID = %q, want %q", stations[0].ID, "osm_1")
		}
	})

	t.Run("empty response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(OverpassResponse{Elements: []OverpassElement{}})
		}))
		defer server.Close()

		c := NewOverpassClient(server.URL, 5*time.Second)
		stations, err := c.GetStations(context.Background(), 55.75, 37.61, 3000, "")
		if err != nil {
			t.Fatalf("GetStations() error = %v", err)
		}
		if len(stations) != 0 {
			t.Errorf("expected 0 stations, got %d", len(stations))
		}
	})

	t.Run("HTML error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html>error</html>`))
		}))
		defer server.Close()

		c := NewOverpassClient(server.URL, 5*time.Second)
		_, err := c.GetStations(context.Background(), 55.75, 37.61, 3000, "")
		if err == nil {
			t.Error("expected error on HTML response")
		}
	})

	t.Run("HTTP error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusGatewayTimeout)
		}))
		defer server.Close()

		c := NewOverpassClient(server.URL, 5*time.Second)
		_, err := c.GetStations(context.Background(), 55.75, 37.61, 3000, "")
		if err == nil {
			t.Error("expected error on HTTP 504")
		}
	})
}

func TestGetStationsWithRetry(t *testing.T) {
	t.Run("success on first try", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(OverpassResponse{
				Elements: []OverpassElement{
					{Type: "node", ID: 1, Lat: 55.75, Lon: 37.61, Tags: OverpassTags{Name: "АЗС"}},
				},
			})
		}))
		defer server.Close()

		c := NewOverpassClient(server.URL, 5*time.Second)
		stations, err := c.GetStationsWithRetry(context.Background(), 55.75, 37.61, 3000, "", 3)
		if err != nil {
			t.Fatalf("GetStationsWithRetry() error = %v", err)
		}
		if len(stations) != 1 {
			t.Errorf("expected 1 station, got %d", len(stations))
		}
		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("retry on failure then success", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts <= 2 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(OverpassResponse{
				Elements: []OverpassElement{
					{Type: "node", ID: 1, Lat: 55.75, Lon: 37.61, Tags: OverpassTags{Name: "АЗС"}},
				},
			})
		}))
		defer server.Close()

		c := NewOverpassClient(server.URL, 5*time.Second)
		stations, err := c.GetStationsWithRetry(context.Background(), 55.75, 37.61, 3000, "", 3)
		if err != nil {
			t.Fatalf("GetStationsWithRetry() error = %v", err)
		}
		if len(stations) != 1 {
			t.Errorf("expected 1 station, got %d", len(stations))
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("all retries fail", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := NewOverpassClient(server.URL, 5*time.Second)
		_, err := c.GetStationsWithRetry(context.Background(), 55.75, 37.61, 3000, "", 2)
		if err == nil {
			t.Error("expected error after all retries fail")
		}
		if attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}
	})
}

func TestNewOverpassClientEndpoint(t *testing.T) {
	t.Run("default endpoint", func(t *testing.T) {
		c := NewOverpassClient("", 5*time.Second)
		if c.endpoint != "https://overpass-api.de/api/interpreter" {
			t.Errorf("default endpoint = %q", c.endpoint)
		}
	})

	t.Run("custom endpoint", func(t *testing.T) {
		c := NewOverpassClient("https://custom.overpass.example.com/api", 5*time.Second)
		if c.endpoint != "https://custom.overpass.example.com/api" {
			t.Errorf("custom endpoint = %q", c.endpoint)
		}
	})
}

func TestGetStations_WithFuels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OverpassResponse{
			Elements: []OverpassElement{
				{
					Type: "node", ID: 10, Lat: 55.75, Lon: 37.61,
					Tags: OverpassTags{
						Name:         "АЗС-1",
						Brand:        "Лукойл",
						FuelOctane92: "yes",
						FuelOctane95: "yes",
						FuelOctane98: "yes",
						FuelDiesel:   "yes",
						FuelLPG:      "yes",
						FuelCNG:      "yes",
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewOverpassClient(server.URL, 5*time.Second)
	stations, err := c.GetStations(context.Background(), 55.75, 37.61, 3000, "")
	if err != nil {
		t.Fatalf("GetStations() error = %v", err)
	}
	if len(stations) != 1 {
		t.Fatalf("expected 1 station, got %d", len(stations))
	}

	s := stations[0]
	if s.Brand != "Лукойл" {
		t.Errorf("Brand = %q, want %q", s.Brand, "Лукойл")
	}
}
