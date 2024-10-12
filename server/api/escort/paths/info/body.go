package info

import (
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/types"
)

type InfoResponse struct {
	FlagShipIp string `json:"flagShipIp"`
}

func (ir *InfoResponse) ToJson() (string, error) {
	jsonData, err := json.Marshal(ir)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

func FromJson(jsonData string) (InfoResponse, error) {
	var ir InfoResponse
	err := json.Unmarshal([]byte(jsonData), &ir)
	if err != nil {
		return InfoResponse{}, err
	}
	return ir, nil
}

func BuildBody(flagship *types.Flagship) InfoResponse {
	return InfoResponse{
		FlagShipIp: flagship.IpAddress.String(),
	}
}
