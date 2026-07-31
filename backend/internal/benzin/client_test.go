package benzin

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func boolPtr(b bool) *bool {
	return &b
}

func TestMapFuelType(t *testing.T) {
	c := NewClient("https://example.com", 5*time.Second)

	tests := []struct {
		input string
		want  string
	}{
		{"AI_92", "АИ-92"},
		{"AI_95", "АИ-95"},
		{"AI_98", "АИ-98"},
		{"AI_100", "АИ-100"},
		{"DT", "ДТ"},
		{"GAS", "Газ"},
		{"UNKNOWN", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := c.mapFuelType(tt.input)
			if got != tt.want {
				t.Errorf("mapFuelType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapQueueLevel(t *testing.T) {
	c := NewClient("https://example.com", 5*time.Second)

	tests := []struct {
		input string
		want  string
	}{
		{"NONE", "none"},
		{"UP_TO_10", "small"},
		{"UP_TO_25", "medium"},
		{"UP_TO_45", "large"},
		{"MORE_45", "very_large"},
		{"none", "none"},
		{"up_to_10", "small"},
		{"Unknown", "unknown"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := c.mapQueueLevel(tt.input)
			if got != tt.want {
				t.Errorf("mapQueueLevel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestQueueToWait(t *testing.T) {
	c := NewClient("https://example.com", 5*time.Second)

	tests := []struct {
		input string
		want  int
	}{
		{"NONE", 0},
		{"UP_TO_10", 10},
		{"UP_TO_25", 25},
		{"UP_TO_45", 45},
		{"MORE_45", 60},
		{"none", 0},
		{"Unknown", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := c.queueToWait(tt.input)
			if got != tt.want {
				t.Errorf("queueToWait(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasFuelType(t *testing.T) {
	c := NewClient("https://example.com", 5*time.Second)

	tests := []struct {
		name       string
		assortment []string
		filter     string
		want       bool
	}{
		{"matching 92", []string{"AI_92", "AI_95"}, "92", true},
		{"matching dt", []string{"DT"}, "dt", true},
		{"matching gas", []string{"GAS"}, "gas", true},
		{"not matching", []string{"AI_92", "AI_95"}, "dt", false},
		{"empty assortment", []string{}, "92", false},
		{"unknown filter", []string{"AI_92"}, "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.hasFuelType(tt.assortment, tt.filter)
			if got != tt.want {
				t.Errorf("hasFuelType(%v, %q) = %v, want %v", tt.assortment, tt.filter, got, tt.want)
			}
		})
	}
}

func TestCalculateBBox(t *testing.T) {
	c := NewClient("https://example.com", 5*time.Second)

	bbox := c.calculateBBox(55.75, 37.61, 3000)

	if bbox[1] >= bbox[3] {
		t.Error("minLat should be less than maxLat")
	}
	if bbox[0] >= bbox[2] {
		t.Error("minLon should be less than maxLon")
	}

	centerLat := (bbox[1] + bbox[3]) / 2
	centerLon := (bbox[0] + bbox[2]) / 2
	if math.Abs(centerLat-55.75) > 0.01 {
		t.Errorf("center lat = %.4f, want %.4f", centerLat, 55.75)
	}
	if math.Abs(centerLon-37.61) > 0.01 {
		t.Errorf("center lon = %.4f, want %.4f", centerLon, 37.61)
	}

	latSpan := bbox[3] - bbox[1]
	lonSpan := bbox[2] - bbox[0]
	if latSpan <= 0 {
		t.Error("lat span should be positive")
	}
	if lonSpan <= 0 {
		t.Error("lon span should be positive")
	}
}

func TestMapStation(t *testing.T) {
	c := NewClient("https://example.com", 5*time.Second)

	t.Run("full station with prices and statuses", func(t *testing.T) {
		detail := StationDetail{
			Station: StationInfo{
				ID:             "123",
				Name:           "Лукойл",
				Brand:          "Лукойл",
				Address:        "ул. Пушкина, 1",
				Lat:            55.75,
				Lng:            37.61,
				FuelAssortment: []string{"AI_92", "AI_95", "DT"},
			},
			FuelStatuses: []FuelStatus{
				{StationID: "123", FuelType: "AI_92", Available: boolPtr(true), QueueLevel: "NONE", LimitLiters: 50},
				{StationID: "123", FuelType: "AI_95", Available: boolPtr(false), QueueLevel: "UP_TO_10", LimitLiters: 30},
				{StationID: "123", FuelType: "DT", Available: nil, QueueLevel: "", LimitLiters: 0},
			},
			Prices: []PriceInfo{
				{StationID: "123", FuelType: "AI_92", Price: 52.50},
				{StationID: "123", FuelType: "AI_95", Price: 56.80},
			},
			QueueLevel:  "UP_TO_10",
			LimitLiters: 50,
			Status:      "active",
		}

		station := c.mapStation(detail, "")
		if station == nil {
			t.Fatal("mapStation() returned nil")
		}

		if station.ID != "123" {
			t.Errorf("ID = %q, want %q", station.ID, "123")
		}
		if station.Name != "Лукойл" {
			t.Errorf("Name = %q, want %q", station.Name, "Лукойл")
		}
		if station.Brand != "Лукойл" {
			t.Errorf("Brand = %q, want %q", station.Brand, "Лукойл")
		}
		if station.Address != "ул. Пушкина, 1" {
			t.Errorf("Address = %q, want %q", station.Address, "ул. Пушкина, 1")
		}
		if station.Lat != 55.75 {
			t.Errorf("Lat = %f, want %f", station.Lat, 55.75)
		}
		if station.Lon != 37.61 {
			t.Errorf("Lon = %f, want %f", station.Lon, 37.61)
		}
		if station.Source != "2gis-benzin" {
			t.Errorf("Source = %q, want %q", station.Source, "2gis-benzin")
		}

		if len(station.Fuels) != 3 {
			t.Fatalf("expected 3 fuels, got %d", len(station.Fuels))
		}

		if station.Fuels[0].Type != "АИ-92" {
			t.Errorf("Fuel[0].Type = %q, want %q", station.Fuels[0].Type, "АИ-92")
		}
		if station.Fuels[0].Avail != "yes" {
			t.Errorf("Fuel[0].Avail = %q, want %q", station.Fuels[0].Avail, "yes")
		}
		if station.Fuels[0].Price != 52.50 {
			t.Errorf("Fuel[0].Price = %f, want %f", station.Fuels[0].Price, 52.50)
		}
		if station.Fuels[0].Currency != "RUB" {
			t.Errorf("Fuel[0].Currency = %q, want %q", station.Fuels[0].Currency, "RUB")
		}

		if station.Fuels[1].Type != "АИ-95" {
			t.Errorf("Fuel[1].Type = %q, want %q", station.Fuels[1].Type, "АИ-95")
		}
		if station.Fuels[1].Avail != "no" {
			t.Errorf("Fuel[1].Avail = %q, want %q", station.Fuels[1].Avail, "no")
		}

		if station.Queue.Level != "small" {
			t.Errorf("Queue.Level = %q, want %q", station.Queue.Level, "small")
		}
		if station.Queue.EstWaitMin != 10 {
			t.Errorf("Queue.EstWaitMin = %d, want %d", station.Queue.EstWaitMin, 10)
		}

		if station.Limits == nil {
			t.Fatal("Limits should not be nil")
		}
		if station.Limits.MaxLiters != 50 {
			t.Errorf("Limits.MaxLiters = %d, want %d", station.Limits.MaxLiters, 50)
		}
		if station.Limits.CanJerrycan {
			t.Error("Limits.CanJerrycan should be false when limit is set")
		}

		if !station.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should be zero time")
		}
	})

	t.Run("nil available is unknown status", func(t *testing.T) {
		detail := StationDetail{
			Station: StationInfo{
				ID: "124", Name: "АЗС", Brand: "Газпром", Address: "ул. Ленина, 1",
				Lat: 55.75, Lng: 37.61, FuelAssortment: []string{"AI_92"},
			},
			FuelStatuses: []FuelStatus{
				{StationID: "124", FuelType: "AI_92", Available: nil},
			},
			Prices:     []PriceInfo{{StationID: "124", FuelType: "AI_92", Price: 50}},
			QueueLevel: "",
			Status:     "active",
		}

		station := c.mapStation(detail, "")
		if station == nil {
			t.Fatal("mapStation() returned nil")
		}

		if station.Fuels[0].Avail != "yes" {
			t.Errorf("nil Available should map to 'yes', got %q", station.Fuels[0].Avail)
		}
		if station.Queue.Level != "none" {
			t.Errorf("empty queue level should default to 'none', got %q", station.Queue.Level)
		}
	})

	t.Run("fallback to fuel assortment when no statuses", func(t *testing.T) {
		detail := StationDetail{
			Station: StationInfo{
				ID: "125", Name: "АЗС", Brand: "Татнефть", Address: "ул. Мира, 1",
				Lat: 55.75, Lng: 37.61,
				FuelAssortment: []string{"AI_92", "AI_95", "AI_98", "DT", "GAS"},
			},
			FuelStatuses: nil,
			Prices:       nil,
			QueueLevel:   "NONE",
			Status:       "active",
		}

		station := c.mapStation(detail, "")
		if station == nil {
			t.Fatal("mapStation() returned nil")
		}

		if len(station.Fuels) != 5 {
			t.Fatalf("expected 5 fuels from assortment, got %d", len(station.Fuels))
		}
		for _, f := range station.Fuels {
			if f.Avail != "unknown" {
				t.Errorf("assortment fallback fuel %q should have avail=unknown, got %q", f.Type, f.Avail)
			}
		}
	})

	t.Run("filtered out by fuel type", func(t *testing.T) {
		detail := StationDetail{
			Station: StationInfo{
				ID: "126", Name: "АЗС", Brand: "Shell", Address: "ул. Тверская, 1",
				Lat: 55.75, Lng: 37.61,
				FuelAssortment: []string{"AI_92"},
			},
			FuelStatuses: []FuelStatus{
				{StationID: "126", FuelType: "AI_92", Available: boolPtr(true)},
			},
			Prices:     []PriceInfo{{StationID: "126", FuelType: "AI_92", Price: 50}},
			QueueLevel: "NONE",
			Status:     "active",
		}

		station := c.mapStation(detail, "95")
		if station != nil {
			t.Error("station without 95 fuel should be filtered out")
		}
	})

	t.Run("unknown fuel type in statuses is skipped", func(t *testing.T) {
		detail := StationDetail{
			Station: StationInfo{
				ID: "127", Name: "АЗС", Brand: "Lukoil", Address: "ул. Садовая, 1",
				Lat: 55.75, Lng: 37.61,
				FuelAssortment: []string{"AI_92", "UNKNOWN_FUEL"},
			},
			FuelStatuses: []FuelStatus{
				{StationID: "127", FuelType: "AI_92", Available: boolPtr(true)},
				{StationID: "127", FuelType: "UNKNOWN_FUEL", Available: boolPtr(true)},
			},
			Prices:     []PriceInfo{{StationID: "127", FuelType: "AI_92", Price: 50}},
			QueueLevel: "NONE",
			Status:     "active",
		}

		station := c.mapStation(detail, "")
		if station == nil {
			t.Fatal("mapStation() returned nil")
		}
		if len(station.Fuels) != 1 {
			t.Fatalf("expected 1 fuel (unknown skipped), got %d", len(station.Fuels))
		}
		if station.Fuels[0].Type != "АИ-92" {
			t.Errorf("Fuel.Type = %q, want %q", station.Fuels[0].Type, "АИ-92")
		}
	})

	t.Run("unknown fuel type in assortment fallback is skipped", func(t *testing.T) {
		detail := StationDetail{
			Station: StationInfo{
				ID: "128", Name: "АЗС", Brand: "Brand", Address: "ул. Новая, 1",
				Lat: 55.75, Lng: 37.61,
				FuelAssortment: []string{"AI_92", "UNKNOWN_FUEL"},
			},
			FuelStatuses: nil,
			Prices:       nil,
			QueueLevel:   "NONE",
			Status:       "active",
		}

		station := c.mapStation(detail, "")
		if station == nil {
			t.Fatal("mapStation() returned nil")
		}
		if len(station.Fuels) != 1 {
			t.Fatalf("expected 1 fuel (unknown skipped), got %d", len(station.Fuels))
		}
	})
}

func TestGetStations_SearchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewClient(server.URL, 5*time.Second)
	_, err := c.GetStations(context.Background(), 55.75, 37.61, 3000, "")
	if err == nil {
		t.Error("expected error on HTTP 500")
	}
}

func TestGetStations_InvalidJSON(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	c := NewClient(server.URL, 5*time.Second)
	_, err := c.GetStations(context.Background(), 55.75, 37.61, 3000, "")
	if err == nil {
		t.Error("expected error on invalid JSON")
	}
}

func TestGetStations_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL, 5*time.Second)
	stations, err := c.GetStations(context.Background(), 55.75, 37.61, 3000, "")
	if err != nil {
		t.Fatalf("GetStations() error = %v", err)
	}
	if len(stations) != 0 {
		t.Errorf("expected 0 stations, got %d", len(stations))
	}
}

func TestGetStations_FullFlow(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			json.NewEncoder(w).Encode([]SearchStation{
				{ID: "1", Name: "АЗС-1", Brand: "Лукойл", Address: "ул. 1", Lat: 55.75, Lng: 37.61},
				{ID: "2", Name: "АЗС-2", Brand: "Газпром", Address: "ул. 2", Lat: 55.76, Lng: 37.62},
			})
			return
		}

		json.NewEncoder(w).Encode([]StationDetail{
			{
				Station: StationInfo{
					ID: "1", Name: "АЗС-1", Brand: "Лукойл", Address: "ул. 1",
					Lat: 55.75, Lng: 37.61, FuelAssortment: []string{"AI_92", "AI_95"},
				},
				FuelStatuses: []FuelStatus{
					{StationID: "1", FuelType: "AI_92", Available: boolPtr(true)},
					{StationID: "1", FuelType: "AI_95", Available: boolPtr(true)},
				},
				Prices: []PriceInfo{
					{StationID: "1", FuelType: "AI_92", Price: 50},
					{StationID: "1", FuelType: "AI_95", Price: 55},
				},
				QueueLevel: "NONE", LimitLiters: 50, Status: "active",
			},
			{
				Station: StationInfo{
					ID: "2", Name: "АЗС-2", Brand: "Газпром", Address: "ул. 2",
					Lat: 55.76, Lng: 37.62, FuelAssortment: []string{"DT"},
				},
				FuelStatuses: []FuelStatus{
					{StationID: "2", FuelType: "DT", Available: boolPtr(true)},
				},
				Prices: []PriceInfo{
					{StationID: "2", FuelType: "DT", Price: 60},
				},
				QueueLevel: "UP_TO_10", LimitLiters: 30, Status: "active",
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, 5*time.Second)
	stations, err := c.GetStations(context.Background(), 55.75, 37.61, 3000, "")
	if err != nil {
		t.Fatalf("GetStations() error = %v", err)
	}

	if len(stations) != 2 {
		t.Fatalf("expected 2 stations, got %d", len(stations))
	}

	if stations[0].ID != "1" {
		t.Errorf("station[0].ID = %q", stations[0].ID)
	}
	if stations[0].Fuels[0].Type != "АИ-92" {
		t.Errorf("station[0].fuel[0].Type = %q", stations[0].Fuels[0].Type)
	}
	if stations[0].Fuels[1].Price != 55 {
		t.Errorf("station[0].fuel[1].Price = %f", stations[0].Fuels[1].Price)
	}
	if stations[0].Queue.Level != "none" {
		t.Errorf("station[0].Queue.Level = %q", stations[0].Queue.Level)
	}

	if stations[1].ID != "2" {
		t.Errorf("station[1].ID = %q", stations[1].ID)
	}
	if stations[1].Fuels[0].Type != "ДТ" {
		t.Errorf("station[1].fuel[0].Type = %q", stations[1].Fuels[0].Type)
	}
}

func TestGetStations_SearchWithFuelFilter(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			json.NewEncoder(w).Encode([]SearchStation{
				{ID: "1", Name: "АЗС-1", Brand: "Лукойл", Address: "ул. 1", Lat: 55.75, Lng: 37.61},
			})
			return
		}

		json.NewEncoder(w).Encode([]StationDetail{
			{
				Station: StationInfo{
					ID: "1", Name: "АЗС-1", Brand: "Лукойл", Address: "ул. 1",
					Lat: 55.75, Lng: 37.61, FuelAssortment: []string{"AI_92"},
				},
				FuelStatuses: []FuelStatus{
					{StationID: "1", FuelType: "AI_92", Available: boolPtr(true)},
				},
				Prices:     []PriceInfo{{StationID: "1", FuelType: "AI_92", Price: 50}},
				QueueLevel: "NONE", LimitLiters: 50, Status: "active",
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, 5*time.Second)

	t.Run("matching fuel filter", func(t *testing.T) {
		stations, err := c.GetStations(context.Background(), 55.75, 37.61, 3000, "92")
		if err != nil {
			t.Fatalf("GetStations() error = %v", err)
		}
		if len(stations) != 1 {
			t.Errorf("expected 1 station, got %d", len(stations))
		}
	})

	t.Run("non-matching fuel filter", func(t *testing.T) {
		stations, err := c.GetStations(context.Background(), 55.75, 37.61, 3000, "dt")
		if err != nil {
			t.Fatalf("GetStations() error = %v", err)
		}
		if len(stations) != 0 {
			t.Errorf("expected 0 station, got %d", len(stations))
		}
	})
}

func TestGetStations_AllStationTypes(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			json.NewEncoder(w).Encode([]SearchStation{
				{ID: "1", Name: "АЗС-1", Brand: "Brand", Address: "ул. 1", Lat: 55.75, Lng: 37.61},
			})
			return
		}

		json.NewEncoder(w).Encode([]StationDetail{
			{
				Station: StationInfo{
					ID: "1", Name: "АЗС-1", Brand: "Brand", Address: "ул. 1",
					Lat: 55.75, Lng: 37.61,
					FuelAssortment: []string{"AI_92", "AI_95", "AI_98", "AI_100", "DT", "GAS"},
				},
				FuelStatuses: []FuelStatus{
					{StationID: "1", FuelType: "AI_92", Available: boolPtr(true)},
					{StationID: "1", FuelType: "AI_95", Available: boolPtr(true)},
					{StationID: "1", FuelType: "AI_98", Available: boolPtr(true)},
					{StationID: "1", FuelType: "AI_100", Available: boolPtr(true)},
					{StationID: "1", FuelType: "DT", Available: boolPtr(true)},
					{StationID: "1", FuelType: "GAS", Available: boolPtr(true)},
				},
				Prices: []PriceInfo{
					{StationID: "1", FuelType: "AI_92", Price: 50},
					{StationID: "1", FuelType: "AI_95", Price: 55},
					{StationID: "1", FuelType: "AI_98", Price: 60},
					{StationID: "1", FuelType: "AI_100", Price: 65},
					{StationID: "1", FuelType: "DT", Price: 45},
					{StationID: "1", FuelType: "GAS", Price: 30},
				},
				QueueLevel: "MORE_45", LimitLiters: 100, Status: "active",
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, 5*time.Second)
	stations, err := c.GetStations(context.Background(), 55.75, 37.61, 3000, "")
	if err != nil {
		t.Fatalf("GetStations() error = %v", err)
	}
	if len(stations) != 1 {
		t.Fatalf("expected 1 station, got %d", len(stations))
	}

	s := stations[0]
	if len(s.Fuels) != 6 {
		t.Fatalf("expected 6 fuel types, got %d", len(s.Fuels))
	}

	expected := []struct {
		type_, avail string
		price        float64
	}{
		{"АИ-92", "yes", 50},
		{"АИ-95", "yes", 55},
		{"АИ-98", "yes", 60},
		{"АИ-100", "yes", 65},
		{"ДТ", "yes", 45},
		{"Газ", "yes", 30},
	}
	for i, exp := range expected {
		if s.Fuels[i].Type != exp.type_ {
			t.Errorf("Fuel[%d].Type = %q, want %q", i, s.Fuels[i].Type, exp.type_)
		}
		if s.Fuels[i].Avail != exp.avail {
			t.Errorf("Fuel[%d].Avail = %q, want %q", i, s.Fuels[i].Avail, exp.avail)
		}
		if s.Fuels[i].Price != exp.price {
			t.Errorf("Fuel[%d].Price = %f, want %f", i, s.Fuels[i].Price, exp.price)
		}
	}

	if s.Queue.Level != "very_large" {
		t.Errorf("Queue.Level = %q, want %q", s.Queue.Level, "very_large")
	}
	if s.Queue.EstWaitMin != 60 {
		t.Errorf("Queue.EstWaitMin = %d, want %d", s.Queue.EstWaitMin, 60)
	}
}

func TestGetStations_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(server.URL, 5*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.GetStations(ctx, 55.75, 37.61, 3000, "")
	if err == nil {
		t.Error("expected error on cancelled context")
	}
}
