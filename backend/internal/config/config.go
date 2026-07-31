package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	TwoGISAPIKey    string
	YandexAPIKey    string
	BenzinBaseURL   string
	PassepartoutURL string
	CacheTTL        time.Duration
	HTTPTimeout     time.Duration
	DefaultRadius   int
	EnableHeadless  bool
	LogLevel        string
	StationProvider string
}

func Load() (*Config, error) {
	godotenv.Load()

	cacheTTL, err := time.ParseDuration(getEnv("CACHE_TTL", "10m"))
	if err != nil {
		cacheTTL = 10 * time.Minute
	}

	httpTimeout, err := time.ParseDuration(getEnv("HTTP_TIMEOUT", "10s"))
	if err != nil {
		httpTimeout = 10 * time.Second
	}

	defaultRadius, err := strconv.Atoi(getEnv("DEFAULT_RADIUS", "3000"))
	if err != nil {
		defaultRadius = 3000
	}

	enableHeadless, _ := strconv.ParseBool(getEnv("ENABLE_HEADLESS_FALLBACK", "false"))
	stationProvider := getEnv("STATION_PROVIDER", "2gis")
	if stationProvider != "2gis" && stationProvider != "osm" {
		stationProvider = "2gis"
	}

	return &Config{
		Port:            getEnv("PORT", "8080"),
		TwoGISAPIKey:    getEnv("TWOGIS_API_KEY", ""),
		YandexAPIKey:    getEnv("YANDEX_API_KEY", ""),
		BenzinBaseURL:   getEnv("BENZIN_BASE_URL", "https://benzin.api.2gis.ru"),
		PassepartoutURL: getEnv("PASSEPARTOUT_URL", "https://passepartout.2gis.com"),
		CacheTTL:        cacheTTL,
		HTTPTimeout:     httpTimeout,
		DefaultRadius:   defaultRadius,
		EnableHeadless:  enableHeadless,
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		StationProvider: stationProvider,
	}, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
