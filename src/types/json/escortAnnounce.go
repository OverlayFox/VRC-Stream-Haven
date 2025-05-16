package types

type Announce struct {
	RtspPort  int     `json:"rtspPort"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
