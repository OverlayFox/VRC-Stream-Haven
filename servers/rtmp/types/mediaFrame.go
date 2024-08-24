package types

import "github.com/yapingcat/gomedia/go-codec"

type MediaFrame struct {
	cid   codec.CodecID
	frame []byte
	pts   uint32
	dts   uint32
}

func (f *MediaFrame) clone() *MediaFrame {
	tmp := &MediaFrame{
		cid: f.cid,
		pts: f.pts,
		dts: f.dts,
	}
	tmp.frame = make([]byte, len(f.frame))
	copy(tmp.frame, f.frame)
	return tmp
}
