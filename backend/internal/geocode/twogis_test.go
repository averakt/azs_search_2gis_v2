package geocode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTwoGisGeocode(t *testing.T) {
	t.Run("successful geocode with full_name", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TwoGisGeocodeResponse{
				Result: struct {
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
				}{
					Items: []struct {
						Point struct {
							Lon float64 `json:"lon"`
							Lat float64 `json:"lat"`
						} `json:"point"`
						Name     string `json:"name"`
						FullName string `json:"full_name"`
						Subtype  string `json:"subtype"`
						Type     string `json:"type"`
					}{
						{
							Point: struct {
								Lon float64 `json:"lon"`
								Lat float64 `json:"lat"`
							}{Lon: 37.618423, Lat: 55.751244},
							Name:     "Тверская улица, 1",
							FullName: "Москва, Тверская улица, 1",
						},
					},
					Total: 1,
				},
			})
		}))
		defer server.Close()

		g := NewTwoGisGeocoder("test-key", 5*time.Second)
		g.baseURL = server.URL

		loc, err := g.Geocode(context.Background(), "Тверская")
		if err != nil {
			t.Fatalf("Geocode() error = %v", err)
		}
		if loc.Lat != 55.751244 {
			t.Errorf("Lat = %f, want %f", loc.Lat, 55.751244)
		}
		if loc.Lon != 37.618423 {
			t.Errorf("Lon = %f, want %f", loc.Lon, 37.618423)
		}
		if loc.Label != "Москва, Тверская улица, 1" {
			t.Errorf("Label = %q, want %q", loc.Label, "Москва, Тверская улица, 1")
		}
	})

	t.Run("fallback to name when full_name empty", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TwoGisGeocodeResponse{
				Result: struct {
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
				}{
					Items: []struct {
						Point struct {
							Lon float64 `json:"lon"`
							Lat float64 `json:"lat"`
						} `json:"point"`
						Name     string `json:"name"`
						FullName string `json:"full_name"`
						Subtype  string `json:"subtype"`
						Type     string `json:"type"`
					}{
						{
							Point: struct {
								Lon float64 `json:"lon"`
								Lat float64 `json:"lat"`
							}{Lon: 37.61, Lat: 55.75},
							Name: "Только название",
						},
					},
					Total: 1,
				},
			})
		}))
		defer server.Close()

		g := NewTwoGisGeocoder("test-key", 5*time.Second)
		g.baseURL = server.URL

		loc, err := g.Geocode(context.Background(), "test")
		if err != nil {
			t.Fatalf("Geocode() error = %v", err)
		}
		if loc.Label != "Только название" {
			t.Errorf("Label = %q, want %q", loc.Label, "Только название")
		}
	})

	t.Run("not found - empty items", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TwoGisGeocodeResponse{
				Result: struct {
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
				}{
					Total: 0,
				},
			})
		}))
		defer server.Close()

		g := NewTwoGisGeocoder("test-key", 5*time.Second)
		g.baseURL = server.URL

		_, err := g.Geocode(context.Background(), "nonexistent")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("no API key", func(t *testing.T) {
		g := NewTwoGisGeocoder("", 5*time.Second)
		_, err := g.Geocode(context.Background(), "test")
		if err == nil {
			t.Error("expected error when api key is empty")
		}
	})

	t.Run("HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		g := NewTwoGisGeocoder("test-key", 5*time.Second)
		g.baseURL = server.URL

		_, err := g.Geocode(context.Background(), "test")
		if err == nil {
			t.Error("expected error on HTTP 500")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{invalid}`))
		}))
		defer server.Close()

		g := NewTwoGisGeocoder("test-key", 5*time.Second)
		g.baseURL = server.URL

		_, err := g.Geocode(context.Background(), "test")
		if err == nil {
			t.Error("expected error on invalid JSON")
		}
	})
}

func TestTwoGisSuggest(t *testing.T) {
	t.Run("suggest returns locations", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TwoGisGeocodeResponse{
				Result: struct {
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
				}{
					Items: []struct {
						Point struct {
							Lon float64 `json:"lon"`
							Lat float64 `json:"lat"`
						} `json:"point"`
						Name     string `json:"name"`
						FullName string `json:"full_name"`
						Subtype  string `json:"subtype"`
						Type     string `json:"type"`
					}{
						{
							Point: struct {
								Lon float64 `json:"lon"`
								Lat float64 `json:"lat"`
							}{Lon: 37.61, Lat: 55.75},
							Name: "Результат 1", FullName: "Москва, Результат 1",
						},
						{
							Point: struct {
								Lon float64 `json:"lon"`
								Lat float64 `json:"lat"`
							}{Lon: 37.62, Lat: 55.76},
							Name: "Результат 2", FullName: "Москва, Результат 2",
						},
					},
					Total: 2,
				},
			})
		}))
		defer server.Close()

		g := NewTwoGisGeocoder("test-key", 5*time.Second)
		g.baseURL = server.URL

		results, err := g.Suggest(context.Background(), "Москва", 5)
		if err != nil {
			t.Fatalf("Suggest() error = %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if results[0].Label != "Москва, Результат 1" {
			t.Errorf("result[0].Label = %q", results[0].Label)
		}
		if results[1].Lat != 55.76 {
			t.Errorf("result[1].Lat = %f, want %f", results[1].Lat, 55.76)
		}
	})

	t.Run("suggest respects limit", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TwoGisGeocodeResponse{
				Result: struct {
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
				}{
					Items: []struct {
						Point struct {
							Lon float64 `json:"lon"`
							Lat float64 `json:"lat"`
						} `json:"point"`
						Name     string `json:"name"`
						FullName string `json:"full_name"`
						Subtype  string `json:"subtype"`
						Type     string `json:"type"`
					}{
						{Point: struct {
							Lon float64 `json:"lon"`
							Lat float64 `json:"lat"`
						}{Lon: 37.61, Lat: 55.75}, Name: "1", FullName: "1"},
						{Point: struct {
							Lon float64 `json:"lon"`
							Lat float64 `json:"lat"`
						}{Lon: 37.62, Lat: 55.76}, Name: "2", FullName: "2"},
						{Point: struct {
							Lon float64 `json:"lon"`
							Lat float64 `json:"lat"`
						}{Lon: 37.63, Lat: 55.77}, Name: "3", FullName: "3"},
					},
					Total: 3,
				},
			})
		}))
		defer server.Close()

		g := NewTwoGisGeocoder("test-key", 5*time.Second)
		g.baseURL = server.URL

		results, err := g.Suggest(context.Background(), "Москва", 2)
		if err != nil {
			t.Fatalf("Suggest() error = %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results with limit=2, got %d", len(results))
		}
	})

	t.Run("empty results", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TwoGisGeocodeResponse{
				Result: struct {
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
				}{
					Items: []struct {
						Point struct {
							Lon float64 `json:"lon"`
							Lat float64 `json:"lat"`
						} `json:"point"`
						Name     string `json:"name"`
						FullName string `json:"full_name"`
						Subtype  string `json:"subtype"`
						Type     string `json:"type"`
					}{},
					Total: 0,
				},
			})
		}))
		defer server.Close()

		g := NewTwoGisGeocoder("test-key", 5*time.Second)
		g.baseURL = server.URL

		results, err := g.Suggest(context.Background(), "test", 5)
		if err != nil {
			t.Fatalf("Suggest() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})
}
