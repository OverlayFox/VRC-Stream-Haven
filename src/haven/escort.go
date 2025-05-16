package haven

import (
	"encoding/base64"
	"encoding/json"

	"github.com/OverlayFox/VRC-Stream-Haven/src/geo"
	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
)

type Escort struct {
	types.MediaSession

	location types.Location
}

func NewEscort(srtSession types.MediaSession) (types.Escort, error) {
	escort := &Escort{
		MediaSession: srtSession,
	}

	location, err := geo.GetPublicLocation(srtSession.GetAddr())
	if err != nil {
		return nil, err
	}
	escort.location = location

	return escort, nil
}

func (e *Escort) GetLocation() types.Location {
	return e.location
}

func (e *Escort) GetAnnounceStruct() (Announce, error) {
	rtspPort, err := e.GetRtspPort()
	if err != nil {
		return Announce{}, err
	}

	return Announce{
		RtspPort:  rtspPort,
		Latitude:  e.location.Latitude,
		Longitude: e.location.Longitude,
	}, nil
}

func (e *Escort) GetAnnounceBase64() (string, error) {
	announce, err := e.GetAnnounceStruct()
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(announce)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(jsonData), nil
}
