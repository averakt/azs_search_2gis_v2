package provider

import (
	"context"

	"azs_search_2gis_v2/backend/internal/model"
)

type Provider interface {
	Geocode(ctx context.Context, q string) (model.Location, error)
	SearchStations(ctx context.Context, loc model.Location, radius int, fuel string) ([]model.Station, error)
}
