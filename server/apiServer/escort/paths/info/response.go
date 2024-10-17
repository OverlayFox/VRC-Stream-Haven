package info

import "encoding/json"

type Response struct {
	CurrentViewers    int `json:"currentViewers"`
	MaxAllowedViewers int `json:"maxAllowedViewers"`
}

func (r *Response) ToJson() (string, error) {
	jsonData, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
