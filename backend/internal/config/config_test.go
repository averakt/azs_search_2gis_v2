package config

import (
	"os"
	"testing"
	"time"
)

func setEnv(key, value string) func() {
	old := os.Getenv(key)
	os.Setenv(key, value)
	return func() { os.Setenv(key, old) }
}

func TestLoadDefaults(t *testing.T) {
	// Clear all relevant env vars
	for _, key := range []string{
		"PORT", "TWOGIS_API_KEY", "YANDEX_API_KEY",
		"BENZIN_BASE_URL", "PASSEPARTOUT_URL",
		"CACHE_TTL", "HTTP_TIMEOUT", "DEFAULT_RADIUS",
		"ENABLE_HEADLESS_FALLBACK", "LOG_LEVEL", "STATION_PROVIDER",
	} {
		unset := setEnv(key, "")
		defer unset()
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.BenzinBaseURL != "https://benzin.api.2gis.ru" {
		t.Errorf("BenzinBaseURL = %q", cfg.BenzinBaseURL)
	}
	if cfg.PassepartoutURL != "https://passepartout.2gis.com" {
		t.Errorf("PassepartoutURL = %q", cfg.PassepartoutURL)
	}
	if cfg.CacheTTL != 10*time.Minute {
		t.Errorf("CacheTTL = %v, want 10m", cfg.CacheTTL)
	}
	if cfg.HTTPTimeout != 10*time.Second {
		t.Errorf("HTTPTimeout = %v, want 10s", cfg.HTTPTimeout)
	}
	if cfg.DefaultRadius != 3000 {
		t.Errorf("DefaultRadius = %d, want 3000", cfg.DefaultRadius)
	}
	if cfg.EnableHeadless != false {
		t.Errorf("EnableHeadless = %v, want false", cfg.EnableHeadless)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.StationProvider != "2gis" {
		t.Errorf("StationProvider = %q, want %q", cfg.StationProvider, "2gis")
	}
}

func TestLoadCustomValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		check func(*Config) bool
	}{
		{"custom port", "PORT", "9090", func(c *Config) bool { return c.Port == "9090" }},
		{"custom cache ttl", "CACHE_TTL", "5m", func(c *Config) bool { return c.CacheTTL == 5*time.Minute }},
		{"custom http timeout", "HTTP_TIMEOUT", "30s", func(c *Config) bool { return c.HTTPTimeout == 30*time.Second }},
		{"custom default radius", "DEFAULT_RADIUS", "5000", func(c *Config) bool { return c.DefaultRadius == 5000 }},
		{"enable headless", "ENABLE_HEADLESS_FALLBACK", "true", func(c *Config) bool { return c.EnableHeadless == true }},
		{"custom log level", "LOG_LEVEL", "debug", func(c *Config) bool { return c.LogLevel == "debug" }},
		{"osm station provider", "STATION_PROVIDER", "osm", func(c *Config) bool { return c.StationProvider == "osm" }},
		{"custom benzin url", "BENZIN_BASE_URL", "https://custom.benzin.ru", func(c *Config) bool { return c.BenzinBaseURL == "https://custom.benzin.ru" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setEnv(tt.key, tt.value)
			defer cleanup()

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if !tt.check(cfg) {
				t.Errorf("check failed for %s=%q", tt.key, tt.value)
			}
		})
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	t.Run("invalid cache ttl falls back to default", func(t *testing.T) {
		cleanup := setEnv("CACHE_TTL", "invalid")
		defer cleanup()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.CacheTTL != 10*time.Minute {
			t.Errorf("CacheTTL = %v, want 10m", cfg.CacheTTL)
		}
	})

	t.Run("invalid http timeout falls back to default", func(t *testing.T) {
		cleanup := setEnv("HTTP_TIMEOUT", "bad")
		defer cleanup()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.HTTPTimeout != 10*time.Second {
			t.Errorf("HTTPTimeout = %v, want 10s", cfg.HTTPTimeout)
		}
	})
}

func TestLoadInvalidRadius(t *testing.T) {
	cleanup := setEnv("DEFAULT_RADIUS", "notanumber")
	defer cleanup()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DefaultRadius != 3000 {
		t.Errorf("DefaultRadius = %d, want 3000", cfg.DefaultRadius)
	}
}

func TestLoadInvalidStationProvider(t *testing.T) {
	cleanup := setEnv("STATION_PROVIDER", "invalid")
	defer cleanup()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.StationProvider != "2gis" {
		t.Errorf("StationProvider = %q, want %q (fallback to '2gis')", cfg.StationProvider, "2gis")
	}
}

func TestLoadAPITokenEmpty(t *testing.T) {
	cleanup := setEnv("TWOGIS_API_KEY", "")
	defer cleanup()
	cleanup2 := setEnv("YANDEX_API_KEY", "")
	defer cleanup2()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.TwoGISAPIKey != "" {
		t.Errorf("TwoGISAPIKey = %q, want empty", cfg.TwoGISAPIKey)
	}
	if cfg.YandexAPIKey != "" {
		t.Errorf("YandexAPIKey = %q, want empty", cfg.YandexAPIKey)
	}
}

func TestGetEnv(t *testing.T) {
	t.Run("existing env var", func(t *testing.T) {
		cleanup := setEnv("TEST_VAR", "value")
		defer cleanup()

		got := getEnv("TEST_VAR", "default")
		if got != "value" {
			t.Errorf("getEnv() = %q, want %q", got, "value")
		}
	})

	t.Run("missing env var uses default", func(t *testing.T) {
		got := getEnv("NONEXISTENT_VAR_12345", "default_value")
		if got != "default_value" {
			t.Errorf("getEnv() = %q, want %q", got, "default_value")
		}
	})
}
