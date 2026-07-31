package geocode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestYandexGeocode(t *testing.T) {
	t.Run("successful geocode", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(YandexGeocodeResponse{
				Response: struct {
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
				}{
					GeoObjectCollection: struct {
						MetaDataProperty struct {
							Name string `json:"name"`
						} `json:"metaDataProperty"`
						Feature []struct {
							Geometry struct {
								Coordinates []float64 `json:"coordinates"`
							} `json:"geometry"`
						} `json:"feature"`
					}{
						MetaDataProperty: struct {
							Name string `json:"name"`
						}{Name: "Москва, Тверская улица, 1"},
						Feature: []struct {
							Geometry struct {
								Coordinates []float64 `json:"coordinates"`
							} `json:"geometry"`
						}{
							{Geometry: struct {
								Coordinates []float64 `json:"coordinates"`
							}{Coordinates: []float64{37.618423, 55.751244}}},
						},
					},
				},
			})
		}))
		defer server.Close()

		g := NewYandexGeocoder("test-key", 5*time.Second)
		g.baseURL = server.URL

		loc, err := g.Geocode(context.Background(), "Москва, Тверская")
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

	t.Run("coordinates are in lon,lat order", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(YandexGeocodeResponse{
				Response: struct {
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
				}{
					GeoObjectCollection: struct {
						MetaDataProperty struct {
							Name string `json:"name"`
						} `json:"metaDataProperty"`
						Feature []struct {
							Geometry struct {
								Coordinates []float64 `json:"coordinates"`
							} `json:"geometry"`
						} `json:"feature"`
					}{
						MetaDataProperty: struct {
							Name string `json:"name"`
						}{Name: "Test"},
						Feature: []struct {
							Geometry struct {
								Coordinates []float64 `json:"coordinates"`
							} `json:"geometry"`
						}{
							{Geometry: struct {
								Coordinates []float64 `json:"coordinates"`
							}{Coordinates: []float64{30.0, 60.0}}},
						},
					},
				},
			})
		}))
		defer server.Close()

		g := NewYandexGeocoder("test-key", 5*time.Second)
		g.baseURL = server.URL

		loc, err := g.Geocode(context.Background(), "test")
		if err != nil {
			t.Fatalf("Geocode() error = %v", err)
		}

		if loc.Lat != 60.0 {
			t.Errorf("Lat = %f, want 60.0 (Yandex returns lon,lat order)", loc.Lat)
		}
		if loc.Lon != 30.0 {
			t.Errorf("Lon = %f, want 30.0 (Yandex returns lon,lat order)", loc.Lon)
		}
	})

	t.Run("no results", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(YandexGeocodeResponse{
				Response: struct {
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
				}{
					GeoObjectCollection: struct {
						MetaDataProperty struct {
							Name string `json:"name"`
						} `json:"metaDataProperty"`
						Feature []struct {
							Geometry struct {
								Coordinates []float64 `json:"coordinates"`
							} `json:"geometry"`
						} `json:"feature"`
					}{
						Feature: []struct {
							Geometry struct {
								Coordinates []float64 `json:"coordinates"`
							} `json:"geometry"`
						}{},
					},
				},
			})
		}))
		defer server.Close()

		g := NewYandexGeocoder("test-key", 5*time.Second)
		g.baseURL = server.URL

		_, err := g.Geocode(context.Background(), "nonexistent")
		if err == nil {
			t.Error("expected error for no results")
		}
	})

	t.Run("no API key", func(t *testing.T) {
		g := NewYandexGeocoder("", 5*time.Second)
		_, err := g.Geocode(context.Background(), "test")
		if err == nil {
			t.Error("expected error when api key is empty")
		}
	})

	t.Run("HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		g := NewYandexGeocoder("test-key", 5*time.Second)
		g.baseURL = server.URL

		_, err := g.Geocode(context.Background(), "test")
		if err == nil {
			t.Error("expected error on HTTP 403")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`invalid`))
		}))
		defer server.Close()

		g := NewYandexGeocoder("test-key", 5*time.Second)
		g.baseURL = server.URL

		_, err := g.Geocode(context.Background(), "test")
		if err == nil {
			t.Error("expected error on invalid JSON")
		}
	})
}
