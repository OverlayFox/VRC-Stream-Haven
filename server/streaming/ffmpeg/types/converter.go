package types

import (
	"fmt"
	"os/exec"
	"strings"
)

type Converter struct {
	Input   InputOutput
	Outputs InputOutput
}

func (c *Converter) StartRemuxer() error {
	ffmpegArgs := []string{"-ignore_unknown"}
	ffmpegArgs = append(ffmpegArgs, c.Input.Args...)

	ffmpegArgs = append(ffmpegArgs, "-c", "copy", "-muxdelay", "0.1", "-f", "tee", "-use_fifo", "1")
	fifoArgs := []string{"restart_with_keyframe=true", "attempt_recovery=true", "drop_pkts_on_overflow=true"}
	ffmpegArgs = append(
		ffmpegArgs,
		fmt.Sprintf(
			"[onfail=ignore:fifo_options=fifo_format=rtsp\\\\\\:%s\\\\\\:format_opts=%s]%s",
			strings.Join(fifoArgs, "\\\\\\:"),
			strings.Join(c.Outputs.Args, "\\\\\\:"),
			c.Outputs.Path),
	)

	cmd := exec.Command("ffmpeg", ffmpegArgs...)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func SetupRemuxer(Input, Output InputOutput) Converter {
	Input.Args = []string{
		"-mode=caller",
		"-smoother=live",
		"-transtype=live",
		"-listen_timeout=-1",
		"-passphrase", Input.Passphrase,
		"-i", Input.Path,
	}

	Output.Args = []string{
		"-rtsp_transport=udp",
		"-rtsp_flags=skip_rtcp",
		"-buffer_size=256k",
		"-pkt_size=736",
	}

	return Converter{
		Input:   Input,
		Outputs: Output,
	}
}
