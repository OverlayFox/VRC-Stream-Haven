package types

import (
	"fmt"
	"sync"
)

type MediaProducer struct {
	name      string
	session   *MediaSession
	mtx       sync.Mutex
	consumers []*MediaSession
	quit      chan struct{}
	die       sync.Once
}

func newMediaProducer(name string, sess *MediaSession) *MediaProducer {
	return &MediaProducer{
		name:      name,
		session:   sess,
		consumers: make([]*MediaSession, 0, 10),
		quit:      make(chan struct{}),
	}
}

func (producer *MediaProducer) start() {
	go producer.dispatch()
}

func (producer *MediaProducer) stop() {
	producer.die.Do(func() {
		close(producer.quit)
		center.unRegister(producer.name)
	})
}

func (producer *MediaProducer) dispatch() {
	defer func() {
		fmt.Println("quit dispatch")
		producer.stop()
	}()
	for {
		select {
		case frame := <-producer.session.C:
			if frame == nil {
				continue
			}
			producer.mtx.Lock()
			tmp := make([]*MediaSession, len(producer.consumers))
			copy(tmp, producer.consumers)
			producer.mtx.Unlock()
			for _, c := range tmp {
				if c.ready() {
					tmp := frame.clone()
					c.play(tmp)
				}
			}
		case <-producer.session.quit:
			return
		case <-producer.quit:
			return
		}
	}
}

func (producer *MediaProducer) addConsumer(consumer *MediaSession) {
	producer.mtx.Lock()
	defer producer.mtx.Unlock()
	producer.consumers = append(producer.consumers, consumer)
}

func (producer *MediaProducer) removeConsumer(id string) {
	producer.mtx.Lock()
	defer producer.mtx.Unlock()
	for i, consume := range producer.consumers {
		if consume.id == id {
			producer.consumers = append(producer.consumers[i:], producer.consumers[i+1:]...)
		}
	}
}
