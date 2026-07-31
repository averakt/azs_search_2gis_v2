package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"azs_search_2gis_v2/backend/internal/benzin"
	"azs_search_2gis_v2/backend/internal/cache"
	"azs_search_2gis_v2/backend/internal/config"
	"azs_search_2gis_v2/backend/internal/geocode"
	"azs_search_2gis_v2/backend/internal/httpapi"
	"azs_search_2gis_v2/backend/internal/osm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))

	cache := cache.New(cfg.CacheTTL)
	benzinClient := benzin.NewClient(cfg.BenzinBaseURL, cfg.HTTPTimeout)
	osmClient := osm.NewOverpassClient("", cfg.HTTPTimeout)
	nominatimClient := osm.NewNominatimClient(cfg.HTTPTimeout)

	geocoder := geocode.NewTwoGisGeocoder(cfg.TwoGISAPIKey, cfg.HTTPTimeout)
	yandexGeocoder := geocode.NewYandexGeocoder(cfg.YandexAPIKey, cfg.HTTPTimeout)

	provider := httpapi.NewTwoGisProvider(
		geocoder,
		yandexGeocoder,
		benzinClient,
		osmClient,
		nominatimClient,
		cache,
		cfg.HTTPTimeout,
		logger,
	)

	handler := httpapi.NewHandler(provider, logger)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	r.Get("/api/health", handler.Health)
	r.Get("/api/geocode", handler.Geocode)
	r.Get("/api/suggest", handler.Suggest)
	r.Get("/api/stations", handler.Stations)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting server", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
