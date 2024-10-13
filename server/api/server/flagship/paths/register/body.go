package register

import (
	"encoding/json"
)

type Body struct {
	IpAddress      string  `yaml:"ipAddress"`
	RtspEgressPort uint16  `yaml:"rtspEgressPort"`
	Latitude       float64 `yaml:"lat"`
	Longitude      float64 `yaml:"lon"`
}

func (r *Body) ToJson() (string, error) {
	jsonData, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
