package model

import "time"

type Station struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Brand     string    `json:"brand"`
	Address   string    `json:"address"`
	Lat       float64   `json:"lat"`
	Lon       float64   `json:"lon"`
	Distance  float64   `json:"distance"`
	Fuels     []Fuel    `json:"fuels"`
	Queue     Queue     `json:"queue"`
	Limits    *Limits   `json:"limits,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	Source    string    `json:"source"`
}

type Fuel struct {
	Type     string  `json:"type"`
	Avail    string  `json:"avail"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
}

type Queue struct {
	Level      string `json:"level"`
	EstWaitMin int    `json:"est_wait_min"`
}

type Limits struct {
	MaxLiters   int  `json:"max_liters"`
	CanJerrycan bool `json:"can_jerrycan"`
}

type Location struct {
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Label string  `json:"label"`
}

type GeocodeResponse struct {
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Label    string  `json:"label"`
	Provider string  `json:"provider"`
}

type StationsResponse struct {
	Stations []Station `json:"stations"`
}
