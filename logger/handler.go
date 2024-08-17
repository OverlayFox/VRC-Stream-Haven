package logger

import (
	"github.com/jaypipes/ghw"
	"github.com/rs/zerolog"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
)

var Log zerolog.Logger

func getGpu() []string {
	gpu, err := ghw.GPU()
	if err != nil {
		log.Printf("Error getting GPU info: %v", err)
		return []string{}
	}

	var gpus []string
	for _, card := range gpu.GraphicsCards {
		gpus = append(gpus, card.DeviceInfo.Product.Name)
	}

	return gpus
}

func InitLogger() {
	logFile, err := os.OpenFile("vrc-haven.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Println("Failed to open log file")
		os.Exit(1)
	}
	multi := io.MultiWriter(os.Stderr, logFile)

	output := zerolog.ConsoleWriter{Out: multi}
	Log = zerolog.New(output).With().Timestamp().Logger()
	Log.Level(zerolog.InfoLevel)

	Log = Log.With().
		Str("app_version", "0.0.1").
		Str("OS", runtime.GOOS).
		Int("CPUs", runtime.NumCPU()).
		Str("arch", runtime.GOARCH).
		Str("gpus", strings.Join(getGpu(), ", ")).
		Logger()
}
