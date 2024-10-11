package types

import (
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/haven"
	"github.com/yapingcat/gomedia/go-codec"
	"github.com/yapingcat/gomedia/go-rtmp"
	"math/rand"
	"net"
	"sync"
)

type MediaSession struct {
	handle    *rtmp.RtmpServerHandle
	conn      net.Conn
	lists     []*MediaFrame
	mtx       sync.Mutex
	id        string
	isReady   bool
	frameCome chan struct{}
	quit      chan struct{}
	source    *MediaProducer
	die       sync.Once
	C         chan *MediaFrame
}

func NewMediaSession(conn net.Conn) *MediaSession {
	id := fmt.Sprintf("%d", rand.Uint64())
	return &MediaSession{
		id:        id,
		conn:      conn,
		handle:    rtmp.NewRtmpServerHandle(),
		quit:      make(chan struct{}),
		frameCome: make(chan struct{}, 1),
		C:         make(chan *MediaFrame, 30),
	}
}

func (sess *MediaSession) Init() {
	sess.handle.OnPlay(func(app, streamName string, start, duration float64, reset bool) rtmp.StatusCode {
		if source := center.find(streamName); source == nil {
			return rtmp.NETSTREAM_PLAY_NOTFOUND
		}
		return rtmp.NETSTREAM_PLAY_START
	})

	sess.handle.OnPublish(func(app, streamName string) rtmp.StatusCode {
		if len(center) == 1 || app != "ingest" || haven.Haven.Flagship.Passphrase != streamName {
			return rtmp.NETSTREAM_CONNECT_REJECTED
		}
		return rtmp.NETSTREAM_PUBLISH_START
	})

	sess.handle.SetOutput(func(b []byte) error {
		_, err := sess.conn.Write(b)
		return err
	})

	sess.handle.OnStateChange(func(newState rtmp.RtmpState) {
		if newState == rtmp.STATE_RTMP_PLAY_START {
			name := sess.handle.GetStreamName()
			source := center.find(name)
			sess.source = source
			if source != nil {
				source.addConsumer(sess)
				sess.isReady = true
				go sess.sendToClient()
			}
		} else if newState == rtmp.STATE_RTMP_PUBLISH_START {
			sess.handle.OnFrame(func(cid codec.CodecID, pts, dts uint32, frame []byte) {
				f := &MediaFrame{
					cid:   cid,
					frame: frame, //make([]byte, len(frame)),
					pts:   pts,
					dts:   dts,
				}
				//copy(f.frame, frame)
				sess.C <- f
			})
			name := sess.handle.GetStreamName()
			p := newMediaProducer(name, sess)
			go p.dispatch()
			center.register(name, p)
		}
	})
}

func (sess *MediaSession) Start() {
	defer sess.Stop()
	for {
		buf := make([]byte, 65536)
		n, err := sess.conn.Read(buf)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = sess.handle.Input(buf[:n])
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func (sess *MediaSession) Stop() {
	sess.die.Do(func() {
		close(sess.quit)
		if sess.source != nil {
			sess.source.removeConsumer(sess.id)
			sess.source = nil
		}
		sess.conn.Close()
	})
}

func (sess *MediaSession) ready() bool {
	return sess.isReady
}

func (sess *MediaSession) play(frame *MediaFrame) {
	sess.mtx.Lock()
	sess.lists = append(sess.lists, frame)
	sess.mtx.Unlock()
	select {
	case sess.frameCome <- struct{}{}:
	default:
	}
}

func (sess *MediaSession) sendToClient() {
	firstVideo := true
	for {
		select {
		case <-sess.frameCome:
			sess.mtx.Lock()
			frames := sess.lists
			sess.lists = nil
			sess.mtx.Unlock()
			for _, frame := range frames {
				if firstVideo { //wait for I frame
					if frame.cid == codec.CODECID_VIDEO_H264 && codec.IsH264IDRFrame(frame.frame) {
						firstVideo = false
					} else if frame.cid == codec.CODECID_VIDEO_H265 && codec.IsH265IDRFrame(frame.frame) {
						firstVideo = false
					} else {
						continue
					}
				}
				err := sess.handle.WriteFrame(frame.cid, frame.frame, frame.pts, frame.dts)
				if err != nil {
					sess.Stop()
					return
				}
			}
		case <-sess.quit:
			return
		}
	}
}
