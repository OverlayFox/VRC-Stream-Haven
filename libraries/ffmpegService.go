package libraries

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func RemuxFlvToRtsp(inputPath, outputPath string) error {
	ffmpegArgs := []string{"-i", inputPath, "-f", "rtsp", outputPath}
	cmd := exec.Command("ffmpeg", ffmpegArgs...)

	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to Run FFMPEG Comand %v", err)
	}

	return nil
}

func NodeHlsPlaylist(rtmpUri string) error {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	inputArgs := []string{"-fflags", "+discardcorrupt+igndts+genpts"}
	rtspInput := []string{"-timeout", "-1", "-i", rtmpUri}
	filterComplex := []string{"-filter_complex", "[0:a]asetpts=PTS-STARTPTS[a0];[0:v]setpts=PTS-STARTPTS[v0]"}
	videoEncode := []string{"-map", "[v0]", "-c:v:0", "h264_nvenc", "-preset", "p2", "-tune", "ll", "-profile:v:0", "main", "-level:v:0", "4.2", "-cbr", "true", "-b:v:0", "2500k", "-g", "50", "-strict_gop", "1"}
	audioEncode := []string{"-map", "[a0]", "-c:a:0", "aac", "-b:a:0", "320k", "-ac:a:0", "2"}
	hlsSettings := []string{"-f", "hls", "-hls_time", "10", "-hls_list_size", "5", "-hls_delete_threshold", "10", "-hls_start_number_source", "epoch", "-hls_allow_cache", "0"}
	hlsFlags := []string{"-hls_flags", "independent_segments"}
	hlsSegments := []string{"-hls_segment_type", "mpegts", "-strftime_mkdir", "1", "-hls_segment_filename", "%v/data_%02d.ts", "-master_pl_name", "playlist.m3u8", "-master_pl_publish_rate", "3"}
	streamMap := []string{"-var_stream_map", "v:0,a:0,name:720p", "stream_%v.m3u8"}

	ffmpegCommand := append(inputArgs, rtspInput...)
	ffmpegCommand = append(ffmpegCommand, filterComplex...)
	ffmpegCommand = append(ffmpegCommand, videoEncode...)
	ffmpegCommand = append(ffmpegCommand, audioEncode...)
	ffmpegCommand = append(ffmpegCommand, hlsSettings...)
	ffmpegCommand = append(ffmpegCommand, hlsFlags...)
	ffmpegCommand = append(ffmpegCommand, hlsSegments...)
	ffmpegCommand = append(ffmpegCommand, streamMap...)

	streamDir := filepath.Join(dir, "streams")
	cmd := exec.Command("ffmpeg", ffmpegCommand...)
	cmd.Dir = streamDir
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to Run HLS FFMPEG Comand: %v", err)
		return err
	}

	return nil
}

func RelayHlsToRtsp(hlsUri, rtspOutput string) error {
	rtspInput := []string{"-timeout", "-1", "-re", "-i", hlsUri}
	filterComplex := []string{"-filter_complex", "[0:a]asetpts=PTS-STARTPTS[a0];[0:v]setpts=PTS-STARTPTS[v0]"}
	videoEncode := []string{"-map", "[v0]", "-c:v:0", "h264_nvenc", "-preset", "p3", "-tune", "ll", "-profile:v:0", "main", "-level:v:0", "3.2", "-cbr", "true", "-b:v:0", "2500k", "-g", "50", "-strict_gop", "1"}
	audioEncode := []string{"-map", "[a0]", "-c:a:0", "aac", "-b:a:0", "320k", "-ac:a:0", "2"}
	rtspSettings := []string{"-f", "rtsp", rtspOutput}

	ffmpegCommand := append(rtspInput, filterComplex...)
	ffmpegCommand = append(ffmpegCommand, videoEncode...)
	ffmpegCommand = append(ffmpegCommand, audioEncode...)
	ffmpegCommand = append(ffmpegCommand, rtspSettings...)

	cmd := exec.Command("ffmpeg", ffmpegCommand...)
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to Run Relay FFMPEG Comand: %v", err)
		return err
	}

	return nil
}
