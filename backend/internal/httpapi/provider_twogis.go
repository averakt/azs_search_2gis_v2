package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"azs_search_2gis_v2/backend/internal/benzin"
	"azs_search_2gis_v2/backend/internal/cache"
	"azs_search_2gis_v2/backend/internal/geocode"
	"azs_search_2gis_v2/backend/internal/model"
	"azs_search_2gis_v2/backend/internal/osm"
	"azs_search_2gis_v2/backend/internal/stations"
)

type TwoGisProvider struct {
	geocoder       *geocode.TwoGisGeocoder
	yandexGeocoder *geocode.YandexGeocoder
	benzin         *benzin.Client
	osm            *osm.OverpassClient
	nominatim      *osm.NominatimClient
	cache          *cache.Cache
	httpTimeout    time.Duration
	logger         *slog.Logger
}

func NewTwoGisProvider(
	geocoder *geocode.TwoGisGeocoder,
	yandexGeocoder *geocode.YandexGeocoder,
	benzin *benzin.Client,
	osm *osm.OverpassClient,
	nominatim *osm.NominatimClient,
	cache *cache.Cache,
	httpTimeout time.Duration,
	logger *slog.Logger,
) *TwoGisProvider {
	return &TwoGisProvider{
		geocoder:       geocoder,
		yandexGeocoder: yandexGeocoder,
		benzin:         benzin,
		osm:            osm,
		nominatim:      nominatim,
		cache:          cache,
		httpTimeout:    httpTimeout,
		logger:         logger,
	}
}

func (p *TwoGisProvider) Geocode(ctx context.Context, q string) (model.Location, error) {
	cacheKey := fmt.Sprintf("geocode:2gis:%s", q)

	v, err := p.cache.GetOrLoad(cacheKey, func() (interface{}, error) {
		return p.geocoder.Geocode(ctx, q)
	})
	if err != nil {
		return model.Location{}, err
	}

	return v.(model.Location), nil
}

func (p *TwoGisProvider) GeocodeWithProvider(ctx context.Context, q string, provider string) (model.Location, error) {
	if provider == "yandex" {
		cacheKey := fmt.Sprintf("geocode:yandex:%s", q)
		v, err := p.cache.GetOrLoad(cacheKey, func() (interface{}, error) {
			return p.yandexGeocoder.Geocode(ctx, q)
		})
		if err != nil {
			return model.Location{}, err
		}
		return v.(model.Location), nil
	}

	cacheKey := fmt.Sprintf("geocode:2gis:%s", q)

	v, err := p.cache.GetOrLoad(cacheKey, func() (interface{}, error) {
		return p.geocoder.Geocode(ctx, q)
	})
	if err != nil {
		return model.Location{}, err
	}

	return v.(model.Location), nil
}

func (p *TwoGisProvider) Suggest(ctx context.Context, q string, limit int) ([]model.Location, error) {
	cacheKey := fmt.Sprintf("suggest:%s:%d", q, limit)

	v, err := p.cache.GetOrLoad(cacheKey, func() (interface{}, error) {
		nominatimResults, err := p.nominatim.Suggest(ctx, q, limit)
		if err != nil {
			return nil, err
		}
		results := make([]model.Location, 0, len(nominatimResults))
		for _, r := range nominatimResults {
			results = append(results, model.Location{
				Lat:   r.Lat,
				Lon:   r.Lon,
				Label: r.Label,
			})
		}
		return results, nil
	})
	if err != nil {
		return nil, err
	}

	return v.([]model.Location), nil
}

func (p *TwoGisProvider) SearchStations(ctx context.Context, loc model.Location, radius int, fuel string, provider string) ([]model.Station, error) {
	if provider == "osm" {
		cacheKey := fmt.Sprintf("stations:osm:%.3f:%.3f:%d:%s", loc.Lat, loc.Lon, radius, fuel)
		v, err := p.cache.GetOrLoad(cacheKey, func() (interface{}, error) {
			return p.osm.GetStationsWithRetry(ctx, loc.Lat, loc.Lon, radius, fuel, 2)
		})
		if err != nil {
			return nil, err
		}
		stationsList := v.([]model.Station)
		stationsList = stations.FilterByFuel(stationsList, fuel)
		stationsList = stations.SortByDistance(stationsList, loc.Lat, loc.Lon)
		return stationsList, nil
	}

	cacheKey := fmt.Sprintf("stations:2gis:%.3f:%.3f:%d:%s", loc.Lat, loc.Lon, radius, fuel)

	v, err := p.cache.GetOrLoad(cacheKey, func() (interface{}, error) {
		stations, err := p.benzin.GetStations(ctx, loc.Lat, loc.Lon, radius, fuel)
		if err != nil {
			p.logger.Error("benzin failed, fallback to osm", "error", err, "lat", loc.Lat, "lon", loc.Lon, "radius", radius)
			return nil, err
		}
		return stations, nil
	})
	if err != nil {
		p.logger.Error("benzin and cache failed, using osm fallback", "error", err)
		osmStations, fallbackErr := p.osm.GetStationsWithRetry(ctx, loc.Lat, loc.Lon, radius, fuel, 2)
		if fallbackErr != nil {
			return nil, fallbackErr
		}
		filteredStations := stations.FilterByFuel(osmStations, fuel)
		sortedStations := stations.SortByDistance(filteredStations, loc.Lat, loc.Lon)
		return sortedStations, nil
	}

	stationsList := v.([]model.Station)
	stationsList = stations.FilterByFuel(stationsList, fuel)
	stationsList = stations.SortByDistance(stationsList, loc.Lat, loc.Lon)

	return stationsList, nil
}
