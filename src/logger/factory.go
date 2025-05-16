package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

// LoggerFactory creates and configures zerolog loggers
type LoggerFactory struct {
	level      zerolog.Level
	timeFormat string
	logDir     string
}

// NewLoggerFactory creates a new logger factory with default settings
func NewLoggerFactory(level zerolog.Level, logDir string) *LoggerFactory {
	if logDir != "" {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Printf("Failed to create log directory: %v\n", err)
		}
	}

	return &LoggerFactory{
		level:      level,
		timeFormat: time.RFC3339,
		logDir:     logDir,
	}
}

// NewLogger creates a new zerolog.Logger that logs to both console and file
func (f *LoggerFactory) NewLogger(name string) zerolog.Logger {
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: f.timeFormat,
	}

	writers := []io.Writer{consoleWriter}

	if f.logDir != "" {
		fileName := fmt.Sprintf("%s-%s.log", name, time.Now().Format("2006-01-02"))
		logFilePath := filepath.Join(f.logDir, fileName)

		logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Failed to open log file: %v\n", err)
		} else {
			writers = append(writers, logFile)
		}
	}

	multiWriter := io.MultiWriter(writers...)

	logger := zerolog.New(multiWriter).
		Level(f.level).
		With().
		Timestamp().
		Str("service", name).
		Logger()

	return logger
}

func (f *LoggerFactory) SetLevel(level zerolog.Level) {
	f.level = level
}

func (f *LoggerFactory) SetTimeFormat(format string) {
	f.timeFormat = format
}
