package osm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNominatimSuggest(t *testing.T) {
	t.Run("successful suggest", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]NominatimResult{
				{Lat: "55.751244", Lon: "37.618423", DisplayName: "Москва, Тверская улица, 1", Type: "house"},
				{Lat: "55.760000", Lon: "37.630000", DisplayName: "Москва, Тверская улица, 10", Type: "house"},
			})
		}))
		defer server.Close()

		client := &NominatimClient{
			baseURL:    server.URL,
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		results, err := client.Suggest(context.Background(), "Тверская", 5)
		if err != nil {
			t.Fatalf("Suggest() error = %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}

		if results[0].Label != "Москва, Тверская улица, 1" {
			t.Errorf("result[0].Label = %q", results[0].Label)
		}
		if results[0].Lat != 55.751244 {
			t.Errorf("result[0].Lat = %f, want %f", results[0].Lat, 55.751244)
		}
		if results[0].Lon != 37.618423 {
			t.Errorf("result[0].Lon = %f, want %f", results[0].Lon, 37.618423)
		}

		if results[1].Lat != 55.760000 {
			t.Errorf("result[1].Lat = %f", results[1].Lat)
		}
	})

	t.Run("empty result", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]NominatimResult{})
		}))
		defer server.Close()

		client := &NominatimClient{
			baseURL:    server.URL,
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		results, err := client.Suggest(context.Background(), "nonexistent", 5)
		if err != nil {
			t.Fatalf("Suggest() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})

	t.Run("invalid lat/lon in response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]NominatimResult{
				{Lat: "invalid", Lon: "37.61", DisplayName: "Test", Type: "house"},
				{Lat: "55.75", Lon: "37.61", DisplayName: "Valid", Type: "house"},
			})
		}))
		defer server.Close()

		client := &NominatimClient{
			baseURL:    server.URL,
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		results, err := client.Suggest(context.Background(), "test", 5)
		if err != nil {
			t.Fatalf("Suggest() error = %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 valid result, got %d", len(results))
		}
		if results[0].Label != "Valid" {
			t.Errorf("result[0].Label = %q, want %q", results[0].Label, "Valid")
		}
	})

	t.Run("HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := &NominatimClient{
			baseURL:    server.URL,
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		_, err := client.Suggest(context.Background(), "test", 5)
		if err == nil {
			t.Error("expected error on HTTP 500")
		}
	})

	t.Run("User-Agent header set", func(t *testing.T) {
		userAgent := ""
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent = r.Header.Get("User-Agent")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]NominatimResult{})
		}))
		defer server.Close()

		client := &NominatimClient{
			baseURL:    server.URL,
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		client.Suggest(context.Background(), "test", 5)
		if userAgent != "azs_search_2gis_v2/1.0" {
			t.Errorf("User-Agent = %q, want %q", userAgent, "azs_search_2gis_v2/1.0")
		}
	})
}
