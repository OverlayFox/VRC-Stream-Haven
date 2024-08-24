package types

import "sync"

type MediaCenter map[string]*MediaProducer

var center MediaCenter
var mtx sync.Mutex

func init() {
	center = make(map[string]*MediaProducer)
}

func (c *MediaCenter) register(name string, p *MediaProducer) {
	mtx.Lock()
	defer mtx.Unlock()
	(*c)[name] = p
}

func (c *MediaCenter) unRegister(name string) {
	mtx.Lock()
	defer mtx.Unlock()
	delete(*c, name)
}

func (c *MediaCenter) find(name string) *MediaProducer {
	mtx.Lock()
	defer mtx.Unlock()
	if p, found := (*c)[name]; found {
		return p
	} else {
		return nil
	}
}
