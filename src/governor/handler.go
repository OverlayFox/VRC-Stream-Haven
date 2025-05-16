package governor

import (
	"sync"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/rs/zerolog"
)

type Governor struct {
	havens []types.Haven

	mtx sync.RWMutex
}

func NewGovernor(logger zerolog.Logger) types.Governor {
	return &Governor{
		havens: []types.Haven{},
	}
}

func (g *Governor) AddHaven(haven types.Haven) {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	g.havens = append(g.havens, haven)
}

func (g *Governor) RemoveHaven(haven types.Haven) error {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	for i, h := range g.havens {
		if h == haven {
			g.havens = append(g.havens[:i], g.havens[i+1:]...)
			return nil
		}
	}

	return types.ErrHavenNotFound
}

func (g *Governor) GetHaven(id string) (types.Haven, error) {
	g.mtx.RLock()
	defer g.mtx.RUnlock()

	for _, h := range g.havens {
		if h.GetStreamId() == id {
			return h, nil
		}
	}
	return nil, types.ErrHavenNotFound
}

func (g *Governor) Start() error {
	return nil
}
