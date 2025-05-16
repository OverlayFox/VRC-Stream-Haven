package types

import (
	pgeo "github.com/paulmach/go.geo"
)

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (l *Location) GetGeoPoint() *pgeo.Point {
	return pgeo.NewPoint(l.Longitude, l.Latitude)
}

func (l *Location) GetDistanceBetween(p2 *pgeo.Point) float64 {
	return l.GetGeoPoint().DistanceFrom(p2)
}
