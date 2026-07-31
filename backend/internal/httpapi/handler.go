package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"azs_search_2gis_v2/backend/internal/geocode"
	"azs_search_2gis_v2/backend/internal/model"
)

type Handler struct {
	provider Provider
	logger   *slog.Logger
}

type Provider interface {
	Geocode(ctx context.Context, q string) (model.Location, error)
	GeocodeWithProvider(ctx context.Context, q string, provider string) (model.Location, error)
	Suggest(ctx context.Context, q string, limit int) ([]model.Location, error)
	SearchStations(ctx context.Context, loc model.Location, radius int, fuel string, provider string) ([]model.Station, error)
}

func NewHandler(provider Provider, logger *slog.Logger) *Handler {
	return &Handler{
		provider: provider,
		logger:   logger,
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func (h *Handler) Geocode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query().Get("q")
	providerName := r.URL.Query().Get("provider")

	if q == "" {
		h.writeError(w, "query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	if providerName == "" {
		providerName = "2gis"
	}

	loc, err := h.provider.GeocodeWithProvider(ctx, q, providerName)
	if err != nil {
		if errors.Is(err, geocode.ErrNotFound) {
			h.writeError(w, "Адрес не найден", http.StatusNotFound)
			return
		}
		h.logger.Error("geocode error", "query", q, "provider", providerName, "error", err)
		h.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := model.GeocodeResponse{
		Lat:      loc.Lat,
		Lon:      loc.Lon,
		Label:    loc.Label,
		Provider: providerName,
	}

	h.writeJSON(w, resp)
}

func (h *Handler) Suggest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")

	if q == "" {
		h.writeError(w, "query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	limit := 5
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	results, err := h.provider.Suggest(ctx, q, limit)
	if err != nil {
		h.logger.Error("suggest error", "query", q, "error", err)
		h.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, results)
}

func (h *Handler) Stations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

	lat, err := strconv.ParseFloat(query.Get("lat"), 64)
	if err != nil {
		h.writeError(w, "invalid lat parameter", http.StatusBadRequest)
		return
	}

	lon, err := strconv.ParseFloat(query.Get("lon"), 64)
	if err != nil {
		h.writeError(w, "invalid lon parameter", http.StatusBadRequest)
		return
	}

	radius, err := strconv.Atoi(query.Get("radius"))
	if err != nil {
		radius = 3000
	}

	fuel := query.Get("fuel")
	providerName := query.Get("provider")
	if providerName == "" {
		providerName = "2gis"
	}

	loc := model.Location{Lat: lat, Lon: lon}
	stations, err := h.provider.SearchStations(ctx, loc, radius, fuel, providerName)
	if err != nil {
		h.logger.Error("stations search error", "location", loc, "error", err)
		h.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := model.StationsResponse{Stations: stations}
	h.writeJSON(w, resp)
}

func (h *Handler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
