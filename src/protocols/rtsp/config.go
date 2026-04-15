package rtsp

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

type Config struct {
	Port       int
	Address    string
	Passphrase string
	IsFlagship bool

	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxPacketSize  int
	WriteQueueSize int
}

func (c Config) Validate() error {
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if len(c.Passphrase) < 10 {
		return errors.New("passphrase must be at least 10 characters long")
	}

	if url.PathEscape(c.Passphrase) != c.Passphrase {
		return errors.New("passphrase contains characters that are not safe for a URL path")
	}

	return nil
}
