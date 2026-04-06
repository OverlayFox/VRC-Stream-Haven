package buffer

import (
	"context"
	"fmt"
	"time"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/rs/zerolog"
	"github.com/yapingcat/gomedia/go-codec"
)

type muxBuffer struct {
	logger zerolog.Logger

	videoBuf types.SubBuffer
	audioBuf types.SubBuffer

	videoReclk *reclocker
}

func NewBuffer(log zerolog.Logger) types.Buffer {
	b := &muxBuffer{
		logger:     log,
		videoBuf:   newBufferStream(log.With().Str("process_name", "video_buffer").Logger(), types.BufferTypeVideo),
		audioBuf:   newBufferStream(log.With().Str("process_name", "audio_buffer").Logger(), types.BufferTypeAudio),
		videoReclk: newReclocker(),
	}
	b.videoReclk.AddStream(codec.CODECID_VIDEO_H264)
	b.videoReclk.AddStream(codec.CODECID_AUDIO_AAC)
	return b
}

func (b *muxBuffer) IsReady() bool {
	return b.videoBuf.IsReady() && b.audioBuf.IsReady()
}

// Write writes a frame to the appropriate buffer based on its codec and also to the interleaved buffer.
func (b *muxBuffer) Write(frame types.Frame) error {
	defer frame.Decommission()

	switch frame.Header().Cid {
	case codec.CODECID_VIDEO_H264:
		vf := frame.Clone()
		if err := b.videoReclk.Reclock(vf); err != nil {
			b.logger.Warn().Str("codec", codec.CodecString(vf.Header().Cid)).Dur("pts", vf.Header().Pts).Dur("dts", vf.Header().Dts).Err(err).Msg("video reclock failed, dropping")
			vf.Decommission()
			return nil
		}
		if err := b.videoBuf.Write(vf); err != nil {
			return err
		}
	case codec.CODECID_AUDIO_AAC:
		af := frame.Clone()
		if err := b.videoReclk.Reclock(af); err != nil {
			b.logger.Warn().Str("codec", codec.CodecString(af.Header().Cid)).Dur("pts", af.Header().Pts).Dur("dts", af.Header().Dts).Err(err).Msg("audio reclock failed, dropping")
			af.Decommission()
			return nil
		}
		if err := b.audioBuf.Write(af); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported codec: %s", codec.CodecString(frame.Header().Cid))
	}

	return nil
}

func (b *muxBuffer) Subscribe(ctx context.Context, offset time.Duration) ([]types.BufferOutput, error) {
	cfg := types.NewSubscribeBuilder().WithPTSOffsetToLive(offset)
	video, err := b.videoBuf.Subscribe(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("video subscribe failed: %w", err)
	}

	cfg = cfg.WithPTSStart(video[0].StartPTS)
	audio, err := b.audioBuf.Subscribe(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("audio subscribe failed: %w", err)
	}

	return append(audio, video...), nil
}

func (b *muxBuffer) Close() {
	b.videoBuf.Cancel()
	b.audioBuf.Cancel()

	b.videoBuf.Close()
	b.audioBuf.Close()
}
