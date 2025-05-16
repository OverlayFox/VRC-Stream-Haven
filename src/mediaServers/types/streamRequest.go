package types

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	globalTypes "github.com/OverlayFox/VRC-Stream-Haven/src/types"
	gJson "github.com/OverlayFox/VRC-Stream-Haven/src/types/json"
)

type StreamRequest struct {
	StreamId       string         `json:"streamId"`
	ConnectionType ConnectionType `json:"connectionType"`
	Announce       gJson.Announce `json:"announce"`
}

func NewStreamRequestFromStreamId(request string) (StreamRequest, error) {
	parts := strings.Split(request, ":")
	partsLen := len(parts)

	if partsLen != 3 && partsLen != 2 {
		return StreamRequest{}, globalTypes.ErrInvalidStreamRequestId
	}

	connType, err := ConnectionTypeFromString(parts[0])
	if err != nil {
		return StreamRequest{}, err
	}

	if partsLen == 2 {
		return StreamRequest{
			StreamId:       parts[1],
			ConnectionType: connType,
			Announce:       gJson.Announce{},
		}, nil
	}

	announceBytes, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return StreamRequest{}, err
	}
	announce := gJson.Announce{}
	if err := json.Unmarshal(announceBytes, &announce); err != nil {
		return StreamRequest{}, err
	}

	return StreamRequest{
		StreamId:       parts[1],
		ConnectionType: connType,
		Announce:       announce,
	}, nil
}

func (sr *StreamRequest) ToStreamIdString() string {
	announceBytes, err := json.Marshal(sr.Announce)
	if err != nil {
		return ""
	}
	announceBase64 := base64.StdEncoding.EncodeToString(announceBytes)

	return strings.Join([]string{
		sr.ConnectionType.String(),
		sr.StreamId,
		announceBase64,
	}, ":")
}
