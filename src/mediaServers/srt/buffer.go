package srt

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/datarhei/gosrt/packet"
	"github.com/rs/zerolog"
)

type packetBuffer struct {
	broadcastCh chan packet.Packet
	subscribers map[chan packet.Packet]types.MediaSession

	subMtx  *sync.RWMutex
	die     sync.Once
	ctx     context.Context
	cancel  context.CancelFunc
	isReady atomic.Bool
	wg      sync.WaitGroup

	logger zerolog.Logger
}

func NewPacketBuffer(capacity int, logger zerolog.Logger) types.PacketBuffer {
	subMtx := &sync.RWMutex{}
	ctx, cancel := context.WithCancel(context.Background())
	pb := &packetBuffer{
		subscribers: map[chan packet.Packet]types.MediaSession{},
		broadcastCh: make(chan packet.Packet, capacity),

		subMtx:  subMtx,
		ctx:     ctx,
		cancel:  cancel,
		die:     sync.Once{},
		isReady: atomic.Bool{},
		wg:      sync.WaitGroup{},

		logger: logger,
	}
	pb.isReady.Store(false)
	go pb.broadcast()

	return pb
}

func (pb *packetBuffer) IsReady() bool {
	return pb.isReady.Load()
}

func (pb *packetBuffer) Reset() {
	pb.logger.Debug().Msg("Resetting packet buffer")

	pb.isReady.Store(false)
	pb.cancel()
	pb.wg.Wait()

	pb.ctx, pb.cancel = context.WithCancel(context.Background())

	oldBroadcastCh := pb.broadcastCh
	pb.broadcastCh = make(chan packet.Packet, cap(oldBroadcastCh))
	close(oldBroadcastCh)
	for packet := range oldBroadcastCh {
		packet.Decommission()
	}

	pb.subMtx.Lock()
	for ch := range pb.subscribers {
		close(ch)
	}
	pb.subscribers = map[chan packet.Packet]types.MediaSession{}
	pb.subMtx.Unlock()

	go pb.broadcast()
}

func (pb *packetBuffer) Close() {
	pb.logger.Debug().Msg("Closing packet buffer")

	pb.die.Do(func() {
		pb.subMtx.Lock()
		defer func() {
			pb.subMtx.Unlock()
		}()

		pb.isReady.Store(false)
		pb.cancel()
		pb.wg.Wait()

		for packet := range pb.broadcastCh {
			packet.Decommission()
		}
		pb.broadcastCh = nil

		for ch := range pb.subscribers {
			close(ch)
		}
		pb.subscribers = nil
	})
}

func (pb *packetBuffer) Subscribe() chan packet.Packet {
	ch := make(chan packet.Packet, 1024)

	pb.subMtx.Lock()
	pb.subscribers[ch] = nil
	pb.subMtx.Unlock()

	return ch
}

func (pb *packetBuffer) Unsubscribe(ch chan packet.Packet) {
	pb.subMtx.Lock()
	if _, ok := pb.subscribers[ch]; ok {
		delete(pb.subscribers, ch)
		close(ch)
	}
	pb.subMtx.Unlock()
}

func (pb *packetBuffer) Write(packet packet.Packet) {
	pb.isReady.CompareAndSwap(false, true)

	select {
	case <-pb.ctx.Done():
		return
	case pb.broadcastCh <- packet:
		// Successfully sent the packet
	default:
		packet.Decommission()
		pb.logger.Warn().Msg("Broadcasting packet buffer is full for SRT stream, dropping packet")
	}
}

func (pb *packetBuffer) broadcast() {
	pb.wg.Add(1)
	defer func() {
		pb.logger.Debug().Msg("Stopped broadcasting packets")
		pb.wg.Done()
	}()

	for {
		select {
		case <-pb.ctx.Done():
			return
		case packet := <-pb.broadcastCh:
			if !pb.isReady.Load() {
				packet.Decommission()
				return
			}

			pb.subMtx.Lock()
			subscribers := pb.subscribers
			pb.subMtx.Unlock()

			for ch := range subscribers {
				packetClone := packet.Clone()
				select {
				case <-pb.ctx.Done():
					packetClone.Decommission()
					return
				case ch <- packetClone:
				default:
					// If the channel is full, we skip sending to that subscriber
					packetClone.Decommission()
					continue
				}
			}
			packet.Decommission()
		}
	}
}
