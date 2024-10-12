package register

import "encoding/json"

type RegisterResponse struct {
	Success   bool   `json:"success"`
	IpAddress string `json:"ipAddress"`
	SrtPort   uint16 `json:"port"`
}

func (r *RegisterResponse) ToJson() (string, error) {
	jsonData, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
