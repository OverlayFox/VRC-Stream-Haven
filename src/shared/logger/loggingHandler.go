package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Level represents logging levels
type Level = zerolog.Level

// Expose common logging levels
const (
	DebugLevel = zerolog.DebugLevel
	InfoLevel  = zerolog.InfoLevel
	WarnLevel  = zerolog.WarnLevel
	ErrorLevel = zerolog.ErrorLevel
	FatalLevel = zerolog.FatalLevel
)

// Config holds logger configuration with documented fields
type Config struct {
	// LogFilePath is the path where log files will be stored
	LogFilePath string
	// LogLevel sets the minimum logging level
	LogLevel Level
	// AppVersion represents the application version
	AppVersion string
	// Environment represents the running environment (dev/staging/prod)
	Environment string
	// MaxFileSize is the maximum size in MB before log rotation
	MaxFileSize int64
	// UseConsole determines if logs should also go to stderr
	UseConsole bool
	// UseJSON forces JSON output instead of console format
	UseJSON bool
}

// defaultConfig provides sensible default values
var defaultConfig = Config{
	LogFilePath: "logs/app.log",
	LogLevel:    InfoLevel,
	Environment: "development",
	MaxFileSize: 100, // 100MB
	UseConsole:  true,
	UseJSON:     false,
}

type Logger struct {
	logger zerolog.Logger
	config Config
}

var (
	instance *Logger
	once     sync.Once
	mu       sync.RWMutex
)

// Init initializes the logger with the provided configuration
func Init(cfg Config) error {
	var err error
	once.Do(func() {
		instance, err = newLogger(cfg)
	})
	return err
}

func newLogger(cfg Config) (*Logger, error) {
	cfg = mergeWithDefaults(cfg)

	if err := os.MkdirAll(filepath.Dir(cfg.LogFilePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(cfg.LogFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	var writers []io.Writer
	writers = append(writers, file)
	if cfg.UseConsole {
		if cfg.UseJSON {
			writers = append(writers, os.Stderr)
		} else {
			writers = append(writers, zerolog.ConsoleWriter{
				Out:        os.Stderr,
				TimeFormat: time.RFC3339,
			})
		}
	}

	var writer io.Writer
	if len(writers) > 1 {
		writer = io.MultiWriter(writers...)
	} else {
		writer = writers[0]
	}

	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(cfg.LogLevel)

	logger := zerolog.New(writer).
		Level(cfg.LogLevel).
		With().
		Timestamp().
		Str("app_version", cfg.AppVersion).
		Str("environment", cfg.Environment).
		Str("os", runtime.GOOS).
		Int("cpus", runtime.NumCPU()).
		Str("arch", runtime.GOARCH).
		Logger()

	return &Logger{
		logger: logger,
		config: cfg,
	}, nil
}

// mergeWithDefaults combines provided config with defaults
func mergeWithDefaults(cfg Config) Config {
	if cfg.LogFilePath == "" {
		cfg.LogFilePath = defaultConfig.LogFilePath
	}
	if cfg.LogLevel == 0 {
		cfg.LogLevel = defaultConfig.LogLevel
	}
	if cfg.Environment == "" {
		cfg.Environment = defaultConfig.Environment
	}
	if cfg.MaxFileSize == 0 {
		cfg.MaxFileSize = defaultConfig.MaxFileSize
	}
	if !cfg.UseJSON {
		cfg.UseJSON = defaultConfig.UseJSON
	}
	return cfg
}

// Named creates a new logger with the provided name context
func Named(name string) zerolog.Logger {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("logger not initialized")
	}
	return instance.logger.With().Str("logger_name", name).Logger()
}

// With creates a new logger with the provided fields
func With(fields map[string]interface{}) zerolog.Logger {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("logger not initialized")
	}

	ctx := instance.logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return ctx.Logger()
}

// Get returns the base logger instance
func Get() zerolog.Logger {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("logger not initialized")
	}
	return instance.logger
}

// Shutdown properly closes the logger
func Shutdown() error {
	mu.Lock()
	defer mu.Unlock()

	instance = nil
	return nil
}
