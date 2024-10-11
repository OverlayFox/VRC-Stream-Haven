package logger

import (
	"github.com/rs/zerolog"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
)

var HavenLogger zerolog.Logger

func init() {
	initLogger()
}

func initLogger() {
	logFile, err := os.OpenFile("hermes.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Println("Failed to open log file")
		os.Exit(1)
	}
	multi := io.MultiWriter(os.Stderr, logFile)

	output := zerolog.ConsoleWriter{Out: multi}
	HavenLogger = zerolog.New(output).With().Timestamp().Logger()
	HavenLogger.Level(zerolog.InfoLevel)

	HavenLogger = HavenLogger.With().
		Str("app_version", os.Getenv("HERMES_VERSION")).
		Str("OS", runtime.GOOS).
		Int("CPUs", runtime.NumCPU()).
		Str("arch", runtime.GOARCH).
		Logger()
}

func NewLoggerWithName(name string) zerolog.Logger {
	return HavenLogger.With().Str("logger_name", strings.ToUpper(name)).Logger()
}
