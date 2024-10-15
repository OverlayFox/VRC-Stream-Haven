package register

import "encoding/json"

type Response struct {
	IpAddress string `json:"ipAddress"`
	SrtPort   uint16 `json:"port"`
}

func (r *Response) ToJson() (string, error) {
	jsonData, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
