package types

import (
	"fmt"
	"github.com/oschwald/geoip2-golang"
)

// Haven combines the ServerStruct and NodeStruct information.
type Haven struct {
	Escorts  *[]*Escort `yaml:"nodes"`
	Flagship *Flagship  `yaml:"server"`
	IsServer bool       `yaml:"isServer"`
}

// GetClosestEscort returns the closest escort to a client.
// The city should be extracted from the IP request made by a client wanting to watch the stream.
func (h *Haven) GetClosestEscort(city *geoip2.City) *Escort {
	type ClosestEscort struct {
		Escort   *Escort
		Distance float64
	}

	var closestEscort ClosestEscort
	for _, escort := range *h.Escorts {
		distance, err := escort.GetDistance(city)
		if err != nil {
			return h.Flagship.Ship
		}

		if closestEscort.Escort == nil || distance < closestEscort.Distance {
			closestEscort.Escort = escort
			closestEscort.Distance = distance
		}
	}

	return closestEscort.Escort
}

func (h *Haven) GetEscort(username string) (*Escort, error) {
	for _, escort := range *h.Escorts {
		if escort.Username == username {
			return escort, nil
		}
	}

	return nil, fmt.Errorf("could not find escort with username: %s", username)
}

func (h *Haven) AddEscort(newEscort *Escort) {
	*h.Escorts = append(*h.Escorts, newEscort)
}

func (h *Haven) RemoveEscort(username string) error {
	for i, escort := range *h.Escorts {
		if escort.Username == username {
			*h.Escorts = append((*h.Escorts)[:i], (*h.Escorts)[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("could not find escort with username: %s", username)
}
