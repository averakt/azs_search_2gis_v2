package stations

import (
	"math"
	"testing"

	"azs_search_2gis_v2/backend/internal/model"
)

func TestCalculateDistance(t *testing.T) {
	tests := []struct {
		name                   string
		lat1, lon1, lat2, lon2 float64
		want                   float64
		delta                  float64
	}{
		{
			name: "zero distance",
			lat1: 55.751244, lon1: 37.618423,
			lat2: 55.751244, lon2: 37.618423,
			want:  0,
			delta: 1,
		},
		{
			name: "moscow to saint-petersburg ~635km",
			lat1: 59.934280, lon1: 30.335099,
			lat2: 55.755826, lon2: 37.617300,
			want:  635000,
			delta: 10000,
		},
		{
			name: "kremlin to red square ~300m",
			lat1: 55.752023, lon1: 37.617501,
			lat2: 55.754093, lon2: 37.620895,
			want:  313,
			delta: 50,
		},
		{
			name: "equatorial points",
			lat1: 0, lon1: 0,
			lat2: 0, lon2: 1,
			want:  111195,
			delta: 500,
		},
		{
			name: "symmetric",
			lat1: 55.751244, lon1: 37.618423,
			lat2: 55.760000, lon2: 37.630000,
			want:  1270,
			delta: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if math.Abs(got-tt.want) > tt.delta {
				t.Errorf("CalculateDistance() = %.0f, want %.0f ± %.0f", got, tt.want, tt.delta)
			}
		})
	}
}

func TestSortByDistance(t *testing.T) {
	tests := []struct {
		name     string
		stations []model.Station
		lat, lon float64
		want     []float64
	}{
		{
			name:     "empty list",
			stations: []model.Station{},
			lat:      55.75, lon: 37.61,
			want: []float64{},
		},
		{
			name: "already sorted",
			stations: []model.Station{
				{Lat: 55.751, Lon: 37.618},
				{Lat: 55.760, Lon: 37.630},
				{Lat: 55.770, Lon: 37.650},
			},
			lat: 55.750, lon: 37.610,
			want: []float64{513, 1674, 3348},
		},
		{
			name: "reverse order",
			stations: []model.Station{
				{Lat: 55.770, Lon: 37.650},
				{Lat: 55.751, Lon: 37.618},
				{Lat: 55.760, Lon: 37.630},
			},
			lat: 55.750, lon: 37.610,
			want: []float64{513, 1674, 3348},
		},
		{
			name: "single station",
			stations: []model.Station{
				{Lat: 55.751, Lon: 37.618},
			},
			lat: 55.750, lon: 37.610,
			want: []float64{513},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SortByDistance(tt.stations, tt.lat, tt.lon)
			if len(got) != len(tt.want) {
				t.Fatalf("SortByDistance() returned %d stations, want %d", len(got), len(tt.want))
			}
			for i := range got {
				dist := math.Round(got[i].Distance)
				if dist != tt.want[i] {
					t.Errorf("station[%d].Distance = %.0f, want %.0f", i, dist, tt.want[i])
				}
			}
		})
	}
}

func TestFilterByFuel(t *testing.T) {
	stations := []model.Station{
		{
			ID: "1",
			Fuels: []model.Fuel{
				{Type: "АИ-92"},
				{Type: "АИ-95"},
			},
		},
		{
			ID: "2",
			Fuels: []model.Fuel{
				{Type: "ДТ"},
			},
		},
		{
			ID: "3",
			Fuels: []model.Fuel{
				{Type: "АИ-95"},
				{Type: "Газ"},
			},
		},
		{
			ID:    "4",
			Fuels: []model.Fuel{},
		},
	}

	tests := []struct {
		name     string
		fuelType string
		wantIDs  []string
	}{
		{
			name:     "empty filter returns all",
			fuelType: "",
			wantIDs:  []string{"1", "2", "3", "4"},
		},
		{
			name:     "filter AI-95",
			fuelType: "95",
			wantIDs:  []string{"1", "3"},
		},
		{
			name:     "filter DT",
			fuelType: "dt",
			wantIDs:  []string{"2"},
		},
		{
			name:     "filter gas",
			fuelType: "gas",
			wantIDs:  []string{"3"},
		},
		{
			name:     "no match",
			fuelType: "100",
			wantIDs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterByFuel(stations, tt.fuelType)
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("FilterByFuel() returned %d stations, want %d", len(got), len(tt.wantIDs))
			}
			for i, id := range tt.wantIDs {
				if got[i].ID != id {
					t.Errorf("station[%d].ID = %s, want %s", i, got[i].ID, id)
				}
			}
		})
	}
}

func TestMatchFuelType(t *testing.T) {
	tests := []struct {
		fuelType string
		filter   string
		want     bool
	}{
		{"АИ-92", "92", true},
		{"92", "92", true},
		{"petrol_station_fuel_unleaded_92", "92", true},
		{"АИ-95", "95", true},
		{"95", "95", true},
		{"petrol_station_fuel_unleaded_95", "95", true},
		{"АИ-98", "98", true},
		{"АИ-100", "100", true},
		{"ДТ", "dt", true},
		{"ДТ+", "dt", true},
		{"diesel", "dt", true},
		{"petrol_station_fuel_diesel", "dt", true},
		{"Газ", "gas", true},
		{"Пропан", "gas", true},
		{"Метан", "gas", true},
		{"LPG", "gas", true},
		{"CNG", "gas", true},
		{"АИ-92", "95", false},
		{"ДТ", "92", false},
		{"", "92", false},
	}

	for _, tt := range tests {
		t.Run(tt.fuelType+"_"+tt.filter, func(t *testing.T) {
			got := matchFuelType(tt.fuelType, tt.filter)
			if got != tt.want {
				t.Errorf("matchFuelType(%q, %q) = %v, want %v", tt.fuelType, tt.filter, got, tt.want)
			}
		})
	}
}
