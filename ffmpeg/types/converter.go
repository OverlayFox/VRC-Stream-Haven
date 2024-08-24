package types

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Converter struct {
	Input   InputOutput
	Outputs []InputOutput
}

func (c *Converter) SetupRemuxer() error {
	for _, output := range c.Outputs {
		switch output.Protocol {
		case "rtsp":
			output.Args = append(
				output.Args,
				"-rtsp_transport=udp",
				"-rtsp_flags=skip_rtcp",
				"-buffer_size=256k",
				"-pkt_size=736",
			)

		case "srt":
			output.Args = append(
				output.Args,
				"-latency=400",
				"-mode=caller",
				"-smoother=live",
				"-transtype=live",
				"-passphrase", c.Input.Passphrase,
			)

		default:
			return errors.New("invalid output protocol. Supported protocols are 'rtsp' and 'srt'")
		}
	}

	c.Input.Args = []string{"-ignore_unknown"}

	switch c.Input.Protocol {
	case "rtmp":
		c.Input.Args = append(c.Input.Args, "-rtmp_live", "1", "-timeout", "-1")

	case "srt":
		if c.Input.Passphrase == "" {
			return errors.New("no input passphrase specified")
		}

		c.Input.Args = append(
			c.Input.Args,
			"-mode", "listener",
			"-smoother", "live",
			"-transtype", "live",
			"-listen_timeout", "-1",
			"-passphrase", c.Input.Passphrase,
		)

	default:
		return errors.New("invalid input protocol. Supported protocols are 'rtmp' and 'srt'")
	}

	return nil
}

func (c *Converter) StartRemuxer() error {
	ffmpegArgs := []string{"-ignore_unknown", "-i", c.Input.Path}
	ffmpegArgs = append(ffmpegArgs, c.Input.Args...)

	ffmpegArgs = append(ffmpegArgs, "-c", "copy", "-muxdelay", "0.1", "-f", "tee", "-use_fifo", "1")
	fifoArgs := []string{"restart_with_keyframe=true", "attempt_recovery=true", "drop_pkts_on_overflow=true"}

	for _, output := range c.Outputs {
		ffmpegArgs = append(
			ffmpegArgs,
			fmt.Sprintf(
				"[onfail=ignore:fifo_options=fifo_format=%s\\\\\\:%s\\\\\\:format_opts=%s]%s",
				output.Protocol,
				strings.Join(fifoArgs, "\\\\\\:"),
				strings.Join(output.Args, "\\\\\\:"),
				output.Path),
		)
	}

	cmd := exec.Command("ffmpeg", ffmpegArgs...)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
