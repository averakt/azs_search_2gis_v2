package stations

import (
	"math"

	"azs_search_2gis_v2/backend/internal/model"
)

func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371e3
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*
			math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func SortByDistance(stations []model.Station, lat, lon float64) []model.Station {
	for i := range stations {
		stations[i].Distance = CalculateDistance(lat, lon, stations[i].Lat, stations[i].Lon)
	}

	for i := 0; i < len(stations)-1; i++ {
		for j := i + 1; j < len(stations); j++ {
			if stations[i].Distance > stations[j].Distance {
				stations[i], stations[j] = stations[j], stations[i]
			}
		}
	}

	return stations
}

func FilterByFuel(stations []model.Station, fuelType string) []model.Station {
	if fuelType == "" {
		return stations
	}

	filtered := make([]model.Station, 0)
	for _, s := range stations {
		for _, f := range s.Fuels {
			if matchFuelType(f.Type, fuelType) {
				filtered = append(filtered, s)
				break
			}
		}
	}

	return filtered
}

func matchFuelType(fuelType, filter string) bool {
	fuelMap := map[string][]string{
		"92":  {"АИ-92", "92", "petrol_station_fuel_unleaded_92"},
		"95":  {"АИ-95", "95", "petrol_station_fuel_unleaded_95"},
		"98":  {"АИ-98", "98", "petrol_station_fuel_unleaded_98"},
		"100": {"АИ-100", "100", "petrol_station_fuel_unleaded_100"},
		"dt":  {"ДТ", "ДТ+", "diesel", "petrol_station_fuel_diesel"},
		"gas": {"Газ", "Пропан", "Метан", "LPG", "CNG"},
	}

	types, ok := fuelMap[filter]
	if !ok {
		types = []string{filter}
	}

	for _, t := range types {
		if fuelType == t {
			return true
		}
	}

	return false
}
