package types

type IpApi struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	IpAddress string  `json:"query"`
}
