package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"azs_search_2gis_v2/backend/internal/geocode"
	"azs_search_2gis_v2/backend/internal/model"
)

type mockProvider struct {
	geocodeFunc     func(ctx context.Context, q string) (model.Location, error)
	geocodeWithFunc func(ctx context.Context, q string, provider string) (model.Location, error)
	suggestFunc     func(ctx context.Context, q string, limit int) ([]model.Location, error)
	searchFunc      func(ctx context.Context, loc model.Location, radius int, fuel string, provider string) ([]model.Station, error)
}

func (m *mockProvider) Geocode(ctx context.Context, q string) (model.Location, error) {
	return m.geocodeFunc(ctx, q)
}

func (m *mockProvider) GeocodeWithProvider(ctx context.Context, q string, provider string) (model.Location, error) {
	if m.geocodeWithFunc != nil {
		return m.geocodeWithFunc(ctx, q, provider)
	}
	return m.geocodeFunc(ctx, q)
}

func (m *mockProvider) Suggest(ctx context.Context, q string, limit int) ([]model.Location, error) {
	return m.suggestFunc(ctx, q, limit)
}

func (m *mockProvider) SearchStations(ctx context.Context, loc model.Location, radius int, fuel string, provider string) ([]model.Station, error) {
	return m.searchFunc(ctx, loc, radius, fuel, provider)
}

func newHandler(m *mockProvider) *Handler {
	return NewHandler(m, slog.Default())
}

func TestHealth(t *testing.T) {
	h := newHandler(&mockProvider{})
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("body = %v", body)
	}
}

func TestGeocode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newHandler(&mockProvider{
			geocodeWithFunc: func(ctx context.Context, q string, provider string) (model.Location, error) {
				return model.Location{Lat: 55.75, Lon: 37.61, Label: "Москва, Тверская"}, nil
			},
		})

		req := httptest.NewRequest("GET", "/api/geocode?q=Тверская", nil)
		w := httptest.NewRecorder()

		h.Geocode(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp model.GeocodeResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.Lat != 55.75 {
			t.Errorf("Lat = %f", resp.Lat)
		}
		if resp.Lon != 37.61 {
			t.Errorf("Lon = %f", resp.Lon)
		}
		if resp.Label != "Москва, Тверская" {
			t.Errorf("Label = %q", resp.Label)
		}
		if resp.Provider != "2gis" {
			t.Errorf("Provider = %q", resp.Provider)
		}
	})

	t.Run("missing q parameter", func(t *testing.T) {
		h := newHandler(&mockProvider{})
		req := httptest.NewRequest("GET", "/api/geocode", nil)
		w := httptest.NewRecorder()

		h.Geocode(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("address not found", func(t *testing.T) {
		h := newHandler(&mockProvider{
			geocodeWithFunc: func(ctx context.Context, q string, provider string) (model.Location, error) {
				return model.Location{}, geocode.ErrNotFound
			},
		})

		req := httptest.NewRequest("GET", "/api/geocode?q=nonexistent", nil)
		w := httptest.NewRecorder()

		h.Geocode(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("geocode error", func(t *testing.T) {
		h := newHandler(&mockProvider{
			geocodeWithFunc: func(ctx context.Context, q string, provider string) (model.Location, error) {
				return model.Location{}, errors.New("API error")
			},
		})

		req := httptest.NewRequest("GET", "/api/geocode?q=test", nil)
		w := httptest.NewRecorder()

		h.Geocode(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestSuggest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newHandler(&mockProvider{
			suggestFunc: func(ctx context.Context, q string, limit int) ([]model.Location, error) {
				return []model.Location{
					{Lat: 55.75, Lon: 37.61, Label: "Москва, Тверская, 1"},
					{Lat: 55.76, Lon: 37.62, Label: "Москва, Тверская, 10"},
				}, nil
			},
		})

		req := httptest.NewRequest("GET", "/api/suggest?q=Тверская", nil)
		w := httptest.NewRecorder()

		h.Suggest(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var results []model.Location
		json.NewDecoder(w.Body).Decode(&results)
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if results[0].Label != "Москва, Тверская, 1" {
			t.Errorf("result[0].Label = %q", results[0].Label)
		}
	})

	t.Run("missing q parameter", func(t *testing.T) {
		h := newHandler(&mockProvider{})
		req := httptest.NewRequest("GET", "/api/suggest", nil)
		w := httptest.NewRecorder()

		h.Suggest(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("custom limit", func(t *testing.T) {
		capturedLimit := 0
		h := newHandler(&mockProvider{
			suggestFunc: func(ctx context.Context, q string, limit int) ([]model.Location, error) {
				capturedLimit = limit
				return nil, nil
			},
		})

		req := httptest.NewRequest("GET", "/api/suggest?q=test&limit=3", nil)
		w := httptest.NewRecorder()
		h.Suggest(w, req)

		if capturedLimit != 3 {
			t.Errorf("limit = %d, want 3", capturedLimit)
		}
	})

	t.Run("default limit when not provided", func(t *testing.T) {
		capturedLimit := 0
		h := newHandler(&mockProvider{
			suggestFunc: func(ctx context.Context, q string, limit int) ([]model.Location, error) {
				capturedLimit = limit
				return nil, nil
			},
		})

		req := httptest.NewRequest("GET", "/api/suggest?q=test", nil)
		w := httptest.NewRecorder()
		h.Suggest(w, req)

		if capturedLimit != 5 {
			t.Errorf("default limit = %d, want 5", capturedLimit)
		}
	})

	t.Run("invalid limit defaults to 5", func(t *testing.T) {
		capturedLimit := 0
		h := newHandler(&mockProvider{
			suggestFunc: func(ctx context.Context, q string, limit int) ([]model.Location, error) {
				capturedLimit = limit
				return nil, nil
			},
		})

		req := httptest.NewRequest("GET", "/api/suggest?q=test&limit=invalid", nil)
		w := httptest.NewRecorder()
		h.Suggest(w, req)

		if capturedLimit != 5 {
			t.Errorf("default limit = %d, want 5", capturedLimit)
		}
	})

	t.Run("empty result", func(t *testing.T) {
		h := newHandler(&mockProvider{
			suggestFunc: func(ctx context.Context, q string, limit int) ([]model.Location, error) {
				return []model.Location{}, nil
			},
		})

		req := httptest.NewRequest("GET", "/api/suggest?q=nonexistent", nil)
		w := httptest.NewRecorder()
		h.Suggest(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var results []model.Location
		json.NewDecoder(w.Body).Decode(&results)
		if len(results) != 0 {
			t.Errorf("expected empty array, got %d results", len(results))
		}
	})

	t.Run("suggest error", func(t *testing.T) {
		h := newHandler(&mockProvider{
			suggestFunc: func(ctx context.Context, q string, limit int) ([]model.Location, error) {
				return nil, errors.New("API error")
			},
		})

		req := httptest.NewRequest("GET", "/api/suggest?q=test", nil)
		w := httptest.NewRecorder()
		h.Suggest(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestStations(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newHandler(&mockProvider{
			searchFunc: func(ctx context.Context, loc model.Location, radius int, fuel string, provider string) ([]model.Station, error) {
				return []model.Station{
					{
						ID: "1", Name: "АЗС-1", Brand: "Лукойл",
						Lat: 55.75, Lon: 37.61, Distance: 100,
						Fuels: []model.Fuel{
							{Type: "АИ-95", Avail: "yes", Price: 52.50, Currency: "RUB"},
						},
						Queue:  model.Queue{Level: "none", EstWaitMin: 0},
						Source: "2gis-benzin",
					},
				}, nil
			},
		})

		req := httptest.NewRequest("GET", "/api/stations?lat=55.75&lon=37.61&radius=3000&fuel=95", nil)
		w := httptest.NewRecorder()
		h.Stations(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp model.StationsResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Stations) != 1 {
			t.Fatalf("expected 1 station, got %d", len(resp.Stations))
		}
		if resp.Stations[0].ID != "1" {
			t.Errorf("ID = %q", resp.Stations[0].ID)
		}
		if resp.Stations[0].Fuels[0].Type != "АИ-95" {
			t.Errorf("Fuel.Type = %q", resp.Stations[0].Fuels[0].Type)
		}
	})

	t.Run("missing lat", func(t *testing.T) {
		h := newHandler(&mockProvider{})
		req := httptest.NewRequest("GET", "/api/stations?lon=37.61", nil)
		w := httptest.NewRecorder()
		h.Stations(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("missing lon", func(t *testing.T) {
		h := newHandler(&mockProvider{})
		req := httptest.NewRequest("GET", "/api/stations?lat=55.75", nil)
		w := httptest.NewRecorder()
		h.Stations(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid lat", func(t *testing.T) {
		h := newHandler(&mockProvider{})
		req := httptest.NewRequest("GET", "/api/stations?lat=invalid&lon=37.61", nil)
		w := httptest.NewRecorder()
		h.Stations(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("default radius when not provided", func(t *testing.T) {
		capturedRadius := 0
		h := newHandler(&mockProvider{
			searchFunc: func(ctx context.Context, loc model.Location, radius int, fuel string, provider string) ([]model.Station, error) {
				capturedRadius = radius
				return []model.Station{}, nil
			},
		})

		req := httptest.NewRequest("GET", "/api/stations?lat=55.75&lon=37.61", nil)
		w := httptest.NewRecorder()
		h.Stations(w, req)

		if capturedRadius != 3000 {
			t.Errorf("default radius = %d, want 3000", capturedRadius)
		}
	})

	t.Run("invalid radius defaults to 3000", func(t *testing.T) {
		capturedRadius := 0
		h := newHandler(&mockProvider{
			searchFunc: func(ctx context.Context, loc model.Location, radius int, fuel string, provider string) ([]model.Station, error) {
				capturedRadius = radius
				return []model.Station{}, nil
			},
		})

		req := httptest.NewRequest("GET", "/api/stations?lat=55.75&lon=37.61&radius=invalid", nil)
		w := httptest.NewRecorder()
		h.Stations(w, req)

		if capturedRadius != 3000 {
			t.Errorf("default radius = %d, want 3000", capturedRadius)
		}
	})

	t.Run("default provider to 2gis", func(t *testing.T) {
		capturedProvider := ""
		h := newHandler(&mockProvider{
			searchFunc: func(ctx context.Context, loc model.Location, radius int, fuel string, provider string) ([]model.Station, error) {
				capturedProvider = provider
				return []model.Station{}, nil
			},
		})

		req := httptest.NewRequest("GET", "/api/stations?lat=55.75&lon=37.61", nil)
		w := httptest.NewRecorder()
		h.Stations(w, req)

		if capturedProvider != "2gis" {
			t.Errorf("default provider = %q, want %q", capturedProvider, "2gis")
		}
	})

	t.Run("passes provider parameter", func(t *testing.T) {
		capturedProvider := ""
		h := newHandler(&mockProvider{
			searchFunc: func(ctx context.Context, loc model.Location, radius int, fuel string, provider string) ([]model.Station, error) {
				capturedProvider = provider
				return []model.Station{}, nil
			},
		})

		req := httptest.NewRequest("GET", "/api/stations?lat=55.75&lon=37.61&provider=osm", nil)
		w := httptest.NewRecorder()
		h.Stations(w, req)

		if capturedProvider != "osm" {
			t.Errorf("provider = %q, want %q", capturedProvider, "osm")
		}
	})

	t.Run("passes fuel filter", func(t *testing.T) {
		capturedFuel := ""
		h := newHandler(&mockProvider{
			searchFunc: func(ctx context.Context, loc model.Location, radius int, fuel string, provider string) ([]model.Station, error) {
				capturedFuel = fuel
				return []model.Station{}, nil
			},
		})

		req := httptest.NewRequest("GET", "/api/stations?lat=55.75&lon=37.61&fuel=dt", nil)
		w := httptest.NewRecorder()
		h.Stations(w, req)

		if capturedFuel != "dt" {
			t.Errorf("fuel = %q, want %q", capturedFuel, "dt")
		}
	})

	t.Run("empty stations response", func(t *testing.T) {
		h := newHandler(&mockProvider{
			searchFunc: func(ctx context.Context, loc model.Location, radius int, fuel string, provider string) ([]model.Station, error) {
				return []model.Station{}, nil
			},
		})

		req := httptest.NewRequest("GET", "/api/stations?lat=55.75&lon=37.61", nil)
		w := httptest.NewRecorder()
		h.Stations(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp model.StationsResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Stations) != 0 {
			t.Errorf("expected empty stations, got %d", len(resp.Stations))
		}
	})

	t.Run("search error", func(t *testing.T) {
		h := newHandler(&mockProvider{
			searchFunc: func(ctx context.Context, loc model.Location, radius int, fuel string, provider string) ([]model.Station, error) {
				return nil, errors.New("search failed")
			},
		})

		req := httptest.NewRequest("GET", "/api/stations?lat=55.75&lon=37.61", nil)
		w := httptest.NewRecorder()
		h.Stations(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestWriteJSON(t *testing.T) {
	h := newHandler(&mockProvider{})

	t.Run("writes JSON with content type", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.writeJSON(w, map[string]string{"key": "value"})

		ct := w.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}

		body := strings.TrimSpace(w.Body.String())
		if body != `{"key":"value"}` {
			t.Errorf("body = %q", body)
		}
	})
}

func TestWriteError(t *testing.T) {
	h := newHandler(&mockProvider{})

	w := httptest.NewRecorder()
	h.writeError(w, "test error", http.StatusBadRequest)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "test error" {
		t.Errorf("error body = %q", body)
	}
}
