package types

import (
	"fmt"
)

type MediaMtxConfig struct {
	// LogLevel sets the verbosity of the program
	//
	// Available values are "error", "warn", "info", "debug".
	LogLevel string `yaml:"logLevel"`
	// LogDestinations sets the destinations for the logs
	//
	// Available values are "stdout", "file", "syslog".
	LogDestinations []string `yaml:"logDestinations"`
	// LogFile sets the file to write the logs to.
	//
	// Only applicable if LogDestinations contains "file".
	LogFile string `yaml:"logFile"`

	// ReadTimeout sets the read timeout.
	ReadTimeout string `yaml:"readTimeout"`
	// WriteTimeout sets the write timeout.
	WriteTimeout string `yaml:"writeTimeout"`
	// WriteQueueSize sets the size of the queue for outgoing packages.
	//
	// A higher value allows to increase throughput, a lower value allows to save RAM.
	WriteQueueSize int `yaml:"writeQueueSize"`
	// UdpMaxPayloadSize sets the maximum size for outgoing UDP packets.
	//
	// This can be decreased to avoid fragmentation on networks with a low UDP MTU.
	UdpMaxPayloadSize int `yaml:"udpMaxPayloadSize"`

	// RunOnConnect is a command that runs when a client connects to the server.
	//
	// The following environment variables are available:
	//
	// * RTSP_PORT: RTSP server port
	//
	// * MTX_CONN_TYPE: connection type
	//
	// * MTX_CONN_ID: connection ID
	RunOnConnect string `yaml:"runOnConnect"`
	// Restart the command if it exits.
	RunOnConnectRestart bool `yaml:"runOnConnectRestart"`
	// RunOnDisconnect runs when a client disconnects from the server.
	//
	// The following environment variables are available:
	//
	// * RTSP_PORT: RTSP server port
	//
	// * MTX_CONN_TYPE: connection type
	//
	// * MTX_CONN_ID: connection ID
	RunOnDisconnect string `yaml:"runOnDisconnect"`

	// AuthMethod sets the authentication method.
	// Every time a  user wants to authenticate, the server will call the authentication method.
	//
	// Available values are "internal", "http", "jwt".
	AuthMethod string `yaml:"authMethod"`
	// AuthHTTPAddress sets the address for the HTTP authentication.
	AuthHTTPAddress string `yaml:"authHTTPAddress"`
	// AuthHTTPExclude sets the actions that are excluded from the HTTP authentication.
	AuthHTTPExclude []map[string]string `yaml:"authHTTPExclude"`

	// Api enables the API.
	Api bool `yaml:"api"`
	// ApiAddress sets the port for the API (e.g. :9997).
	ApiAddress string `yaml:"apiAddress"`
	// ApiEncryption enables the encryption for the API.
	ApiEncryption bool `yaml:"apiEncryption"`
	// ApiServerKey sets the path to server key for the API.
	ApiServerKey string `yaml:"apiServerKey"`
	// ApiServerCert sets the path to server certificate for the API.
	ApiServerCert string `yaml:"apiServerCert"`
	// ApiAllowOrigin sets the value of the Access-Control-Allow-Origin header provided in every HTTP response.
	ApiAllowOrigin string `yaml:"apiAllowOrigin"`
	// ApiTrustedProxies is a list of IPs or CIDRs of proxies placed before the HTTP server
	ApiTrustedProxies []string `yaml:"apiTrustedProxies"`

	// Playback enable downloading recordings from the playback server.
	Playback bool `yaml:"playback"`
	// PlaybackAddress sets the port for the playback server (e.g. :9996).
	PlaybackAddress string `yaml:"playbackAddress"`
	// PlaybackEncryption enables the encryption for the playback server.
	PlaybackEncryption bool `yaml:"playbackEncryption"`
	// PlaybackServerKey sets the path to server key for the playback server.
	PlaybackServerKey string `yaml:"playbackServerKey"`
	// PlaybackServerCert sets the path to server certificate for the playback server.
	PlaybackServerCert string `yaml:"playbackServerCert"`
	// PlaybackAllowOrigin sets the value of the Access-Control-Allow-Origin header provided in every HTTP response.
	PlaybackAllowOrigin string `yaml:"playbackAllowOrigin"`
	// PlaybackTrustedProxies is a list of IPs or CIDRs of proxies placed before the playback server
	PlaybackTrustedProxies []string `yaml:"playbackTrustedProxies"`

	// Rtsp enables the RTSP server.
	Rtsp bool `yaml:"rtsp"`

	// Rtmp enables the RTMP server.
	Rtmp bool `yaml:"rtmp"`
	// RtmpAddress sets the port for the RTMP server (e.g. :1935).
	RtmpAddress string `yaml:"rtmpAddress"`
	// RtmpEncryption enables the encryption for the RTMP server.
	RtmpEncryption string `yaml:"rtmpEncryption"`
	// RtmpsAddress sets the port for the RTMPS server (e.g. :1936).
	RtmpsAddress string `yaml:"rtmpsAddress"`
	// RtmpServerKey sets the path to server key for the RTMP server.
	RtmpServerKey string `yaml:"rtmpServerKey"`
	// RtmpServerCert sets the path to server certificate for the RTMP server.
	RtmpServerCert string `yaml:"rtmpServerCert"`

	// Hls enables the HLS server.
	Hls bool `yaml:"hls"`

	// WebRTC enables the WebRTC server.
	WebRTC bool `yaml:"webRTC"`

	// Srt enables the SRT server.
	Srt bool `yaml:"srt"`
	// SrtAddress sets the port for the SRT server (e.g. :8890).
	SrtAddress string `yaml:"srtAddress"`

	// PathDefaults sets the default values for the paths.
	//
	// These values apply anywhere, unless overridden by the path.
	PathDefaults Path `yaml:"pathDefaults"`

	// Paths sets specific paths and can override the default values set in PathDefaults.
	Paths Paths `yaml:"paths"`
}

type Paths map[string]Path

type Path struct {
	// Source of the stream. See mediamtx.yml for more information.
	//
	// https://github.com/bluenviron/mediamtx/blob/main/mediamtx.yml
	Source string `yaml:"source"`
	// SourceFingerprint provides the fingerprint of the certificate in order to validate it anyway,
	// if the source is a URL, and the source certificate is self-signed or invalid
	SourceFingerprint string `yaml:"sourceFingerprint"`
	// SourceOnDemand enables on-demand source.
	SourceOnDemand bool `yaml:"sourceOnDemand"`
	// SourceOnDemandStartTimeout sets the time after which the on-demand source becomes available for clients.
	//
	// SourceOnDemand needs to be true.
	SourceOnDemandStartTimeout string `yaml:"sourceOnDemandStartTimeout"`
	// SourceOnDemandCloseAfter sets the time after which the Source closes if no clients are connected.
	//
	// SourceOnDemand needs to be true.
	SourceOnDemandCloseAfter string `yaml:"sourceOnDemandCloseAfter"`

	// MaxReaders sets the maximum number of readers for the source.
	//
	// 0 means unlimited.
	MaxReaders int `yaml:"maxReaders"`
	// OverridePublisher allows another client to disconnect the current publisher and publish in its place.
	OverridePublisher bool `yaml:"overridePublisher"`

	// FallBack sets the fallback source if no stream is available.
	Fallback string `yaml:"fallback"`

	// SrtReadPassphrase sets the passphrase for reading SRT streams.
	//
	// The passphrase needs to be between 10 and 79 characters.
	SrtReadPassphrase string `yaml:"srtReadPassphrase"`
	// SrtPublishPassphrase sets the passphrase for publishing SRT streams.
	//
	// The passphrase needs to be between 10 and 79 characters.
	SrtPublishPassphrase string `yaml:"srtPublishPassphrase"`

	// Record enables recording of the stream to disk.
	Record bool `yaml:"record"`
	// RecordPath sets the path to save the recordings.
	//
	// Available variables are %path (path name), %Y %m %d %H %M %S %f %s (time in strftime format)
	RecordPath string `yaml:"recordPath"`
	// RecordFormat sets the format of the recordings.
	//
	// Available formats are "fmp4" (fragmented MP4) and "mpegts" (MPEG-TS).
	RecordFormat string `yaml:"recordFormat"`
	// RecordPartDuration sets the duration of each part of the recording.
	// fMP4 segments are concatenation of small MP4 files (parts), each with this duration.
	// MPEG-TS segments are concatenation of 188-bytes packets, flushed to disk with this period.
	// When a system failure occurs, the last part gets lost.
	// Therefore, the part duration is equal to the RPO (recovery point objective).
	RecordPartDuration string `yaml:"recordPartDuration"`
	// RecordSegmentDuration sets the minimum duration of each segment of the recording.
	RecordSegmentDuration string `yaml:"recordSegmentDuration"`
	// RecordDeleteAfter deletes segments after this timespan.
	RecordDeleteAfter string `yaml:"recordDeleteAfter"`

	// RunOnInit sets the command to run when this path is initialized.
	//
	// The following environment variables are available:
	//
	// * MTX_PATH: path name
	//
	// * RTSP_PORT: RTSP server port
	//
	// * G1, G2, ...: regular expression groups, if path name is a regular expression.
	RunOnInit string `yaml:"runOnInit"`
	// RunOnInitRestart restarts the command if it exits.
	RunOnInitRestart bool `yaml:"runOnInitRestart"`

	RunOnDemand             string `yaml:"runOnDemand"`
	RunOnDemandRestart      bool   `yaml:"runOnDemandRestart"`
	RunOnDemandStartTimeout string `yaml:"runOnDemandStartTimeout"`
	RunOnDemandCloseAfter   string `yaml:"runOnDemandCloseAfter"`
	RunOnUnDemand           string `yaml:"runOnUnDemand"`

	RunOnReady        string `yaml:"runOnReady"`
	RunOnReadyRestart bool   `yaml:"runOnReadyRestart"`
	RunOnNotReady     string `yaml:"runOnNotReady"`

	RunOnRead        string `yaml:"runOnRead"`
	RunOnReadRestart bool   `yaml:"runOnReadRestart"`
	RunOnUnRead      string `yaml:"runOnUnRead"`

	RunOnRecordSegmentCreate string `yaml:"runOnRecordSegmentCreate"`

	RunOnRecordSegmentComplete string `yaml:"runOnRecordSegmentComplete"`
}

func (m *MediaMtxConfig) BuildEscortPath(flagship *Flagship, srtPort, rtspPort int) Paths {
	pullUrl := fmt.Sprintf("srt://%s:%d/egress/flagship?passphrase=%s&latency=8000&mode=caller&smoother=live&transtype=live", flagship.IpAddress.String(), flagship.SrtIngestPort, flagship.Passphrase)

	escortDefault := m.BuildDefaultPath()
	escortDefault.Source = pullUrl
	escortDefault.SrtReadPassphrase = flagship.Passphrase
	escortDefault.SourceOnDemand = true
	escortDefault.SourceOnDemandStartTimeout = "10s"
	escortDefault.SourceOnDemandCloseAfter = "120s"
	escortDefault.MaxReaders = 1
	//escortDefault.RunOnReady = fmt.Sprintf("ffmpeg -y -hide_banner -loglevel info connect_timeout=-1 mode=caller -i srt://127.0.0.1:%d?egress/flagship&passphrase=%s -c copy -f rtsp rtsp://127.0.0.1:%d/egress/%s", srtPort, srtPassphrase, rtspPort, srtPassphrase)

	return Paths{
		"ingest/flagship": escortDefault,
	}
}

func (m *MediaMtxConfig) BuildFlagshipPath(srtPassphrase string, srtPort, rtspPort int) Paths {
	flagshipDefault := m.BuildDefaultPath()
	flagshipDefault.SrtReadPassphrase = srtPassphrase
	flagshipDefault.SrtPublishPassphrase = srtPassphrase
	flagshipDefault.Source = "publisher"
	//flagshipDefault.RunOnReady = fmt.Sprintf("ffmpeg -y -hide_banner -loglevel info connect_timeout=-1 mode=caller -i srt://127.0.0.1:%d?egress/flagship&passphrase=%s -c copy -f rtsp rtsp://127.0.0.1:%d/egress/%s", srtPort, srtPassphrase, rtspPort, srtPassphrase)

	return Paths{
		"egress/flagship": flagshipDefault,
	}
}

func (m *MediaMtxConfig) BuildDefaultPath() Path {
	return Path{
		Source:                     "publisher",
		SourceFingerprint:          "",
		SourceOnDemand:             false,
		SourceOnDemandStartTimeout: "10s",
		SourceOnDemandCloseAfter:   "10s",

		MaxReaders:        0,
		OverridePublisher: false,

		SrtReadPassphrase:    "",
		SrtPublishPassphrase: "",

		Record:                false,
		RecordPartDuration:    "10s",
		RecordSegmentDuration: "10s",
		RecordDeleteAfter:     "10s",
		RecordFormat:          "fmp4",

		RunOnInit:        "",
		RunOnInitRestart: false,

		RunOnDemand:             "",
		RunOnDemandRestart:      false,
		RunOnDemandStartTimeout: "10s",
		RunOnDemandCloseAfter:   "10s",
		RunOnUnDemand:           "",

		RunOnReady:        "",
		RunOnReadyRestart: false,
		RunOnNotReady:     "",

		RunOnRead:        "",
		RunOnReadRestart: false,
		RunOnUnRead:      "",

		RunOnRecordSegmentCreate:   "",
		RunOnRecordSegmentComplete: "",
	}
}

func (m *MediaMtxConfig) BuildConfig(srtPassphrase string, pathsConfig Paths, srtPort int) MediaMtxConfig {
	srtPortStr := fmt.Sprintf(":%d", srtPort)

	defaultPath := m.BuildDefaultPath()
	defaultPath.SrtReadPassphrase = srtPassphrase
	defaultPath.SrtPublishPassphrase = srtPassphrase

	return MediaMtxConfig{
		LogLevel:          "info",
		LogDestinations:   []string{"stdout"},
		LogFile:           "mediamtx.log",
		ReadTimeout:       "10s",
		WriteTimeout:      "10s",
		WriteQueueSize:    512,
		UdpMaxPayloadSize: 1472,

		RunOnConnect:        "",
		RunOnConnectRestart: false,
		RunOnDisconnect:     "",

		AuthMethod:      "internal",
		AuthHTTPAddress: "",
		AuthHTTPExclude: []map[string]string{
			{"action": "api"},
			{"action": "metrics"},
			{"action": "pprof"},
		},

		Api:               false,
		ApiAddress:        ":9997",
		ApiEncryption:     false,
		ApiServerKey:      "server.key",
		ApiServerCert:     "server.crt",
		ApiAllowOrigin:    "*",
		ApiTrustedProxies: []string{},

		Playback:               false,
		PlaybackAddress:        ":9996",
		PlaybackEncryption:     false,
		PlaybackServerKey:      "server.key",
		PlaybackServerCert:     "server.crt",
		PlaybackAllowOrigin:    "*",
		PlaybackTrustedProxies: []string{},

		Rtsp:   false,
		Hls:    false,
		WebRTC: false,

		Rtmp:           false,
		RtmpAddress:    ":1935",
		RtmpEncryption: "no",
		RtmpsAddress:   ":1936",
		RtmpServerKey:  "server.key",
		RtmpServerCert: "server.crt",

		Srt:        true,
		SrtAddress: srtPortStr,

		PathDefaults: Path{
			Source:                     "publisher",
			SourceFingerprint:          "",
			SourceOnDemand:             false,
			SourceOnDemandStartTimeout: "10s",
			SourceOnDemandCloseAfter:   "10s",

			MaxReaders:        0,
			OverridePublisher: false,

			SrtReadPassphrase:    srtPassphrase,
			SrtPublishPassphrase: srtPassphrase,

			Record:                false,
			RecordPartDuration:    "10s",
			RecordSegmentDuration: "10s",
			RecordDeleteAfter:     "10s",
			RecordFormat:          "fmp4",

			RunOnInit:        "",
			RunOnInitRestart: false,

			RunOnDemand:             "",
			RunOnDemandRestart:      false,
			RunOnDemandStartTimeout: "10s",
			RunOnDemandCloseAfter:   "10s",
			RunOnUnDemand:           "",

			RunOnReady:        "",
			RunOnReadyRestart: false,
			RunOnNotReady:     "",

			RunOnRead:        "",
			RunOnReadRestart: false,
			RunOnUnRead:      "",

			RunOnRecordSegmentCreate:   "",
			RunOnRecordSegmentComplete: "",
		},

		Paths: pathsConfig,
	}
}
