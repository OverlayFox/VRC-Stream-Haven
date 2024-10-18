package types

import (
	"fmt"
	"github.com/oschwald/geoip2-golang"
	"net"
	"sort"
)

// Haven combines the ServerStruct and NodeStruct information.
type Haven struct {
	Escorts  *[]*Escort `yaml:"nodes"`
	Flagship *Flagship  `yaml:"server"`
}

// GetClosestEscort returns a sorted list. The first element is the closest escort to the client.
// The city should be extracted from the IP request made by a client wanting to watch the stream.
func (h *Haven) GetClosestEscort(city *geoip2.City) []*Escort {
	type EscortWithDistance struct {
		Escort   *Escort
		Distance float64
	}

	var escortsWithDistances []EscortWithDistance
	for _, escort := range *h.Escorts {
		distance, err := escort.GetDistance(city)
		if err != nil {
			flagship := &Escort{
				IpAddress:      h.Flagship.IpAddress,
				RtspEgressPort: h.Flagship.RtspEgressPort,
				Latitude:       h.Flagship.Latitude,
				Longitude:      h.Flagship.Longitude,
			}
			return []*Escort{flagship}
		}

		escortsWithDistances = append(escortsWithDistances, EscortWithDistance{
			Escort:   escort,
			Distance: distance,
		})
	}

	sort.Slice(escortsWithDistances, func(i, j int) bool {
		return escortsWithDistances[i].Distance < escortsWithDistances[j].Distance
	})

	var sortedEscorts []*Escort
	for _, escortWithDistance := range escortsWithDistances {
		sortedEscorts = append(sortedEscorts, escortWithDistance.Escort)
	}

	return sortedEscorts
}

func (h *Haven) GetEscort(ip net.IP) (*Escort, error) {
	for _, escort := range *h.Escorts {
		if ip.Equal(escort.IpAddress) {
			return escort, nil
		}
	}

	return nil, fmt.Errorf("could not find escort with IP: %s", ip.String())
}

func (h *Haven) AddEscort(newEscort *Escort) {
	*h.Escorts = append(*h.Escorts, newEscort)
}

func (h *Haven) RemoveEscort(ip net.IP) error {
	for i, escort := range *h.Escorts {
		if ip.Equal(escort.IpAddress) {
			*h.Escorts = append((*h.Escorts)[:i], (*h.Escorts)[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("could not find escort with IP: %s", ip)
}
