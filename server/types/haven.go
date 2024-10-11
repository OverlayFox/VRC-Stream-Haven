package types

import "fmt"

// Haven combines the ServerStruct and NodeStruct information.
type Haven struct {
	Escorts  *[]*Escort `yaml:"nodes"`
	Flagship *Flagship  `yaml:"server"`
	IsServer bool       `yaml:"isServer"`
}

func (h *Haven) GetEscorts() *[]*Escort {
	return h.Escorts
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
