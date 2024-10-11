package flagship

import "encoding/json"

type RegisterResponse struct {
	Success     bool   `json:"success"`
	IpAddress   string `json:"ipAddress"`
	Port        uint16 `json:"port"`
	Protocol    string `json:"protocol"`
	Application string `json:"application"`
	StreamKey   string `json:"streamKey"`
}

func (r *RegisterResponse) ToJson() (string, error) {
	jsonData, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
