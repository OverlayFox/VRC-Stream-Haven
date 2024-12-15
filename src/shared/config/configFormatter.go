package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

type Config struct {
	IsFlagship      bool   `required:"true" default:"true"`
	Passphrase      []byte `required:"true"`
	ApiPort         int    `required:"true" default:"8080"`
	RtspPort        int    `required:"true" default:"554"`
	SrtPort         int    `required:"true" default:"8554"`
	FlagshipIp      net.IP `required:"false"`
	BackendIp       net.IP `required:"false"`
	FlagshipApiPort int    `required:"false"`
}

func getEnvPassphrase(key string, minLength int) ([]byte, error) {
	value := os.Getenv(key)
	if len(value) < minLength {
		return nil, fmt.Errorf("%s not set or shorter than %d characters", key, minLength)
	}
	return []byte(value), nil
}

func getEnvInt(key string, defaultValue, min, max int) (int, error) {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil || value < min || value > max {
		return defaultValue, fmt.Errorf("%s was set to an invalid value, defaulting to %d", key, defaultValue)
	}
	return value, nil
}

func getEnvIP(key string) net.IP {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	return net.ParseIP(value)
}

func CreateConfigFromEnv() (Config, error) {
	passphrase, err := getEnvPassphrase("PASSPHRASE", 10)
	if err != nil {
		return Config{}, fmt.Errorf("failed to get passphrase: %s", err)
	}

	apiPort, err := getEnvInt("API_PORT", 8080, 1, 65535)
	if err != nil {
		return Config{}, fmt.Errorf("failed to get API port: %s", err)
	}

	rtspPort, err := getEnvInt("RTSP_PORT", 554, 1, 65535)
	if err != nil {
		return Config{}, fmt.Errorf("failed to get RTSP port: %s", err)
	}

	srtPort, err := getEnvInt("SRT_PORT", 8554, 1, 65535)
	if err != nil {
		return Config{}, fmt.Errorf("failed to get SRT port: %s", err)
	}

	flagshipIp := getEnvIP("FLAGSHIP_IP")

	var isFlagship = true
	if flagshipIp == nil {
		isFlagship = false
	}

	flagshipApiPort, err := getEnvInt("FLAGSHIP_API_PORT", 8080, 1, 65535)
	if err != nil {
		return Config{}, fmt.Errorf("failed to get Flagship API port: %s", err)
	}

	return Config{
		Passphrase:      passphrase,
		ApiPort:         apiPort,
		RtspPort:        rtspPort,
		SrtPort:         srtPort,
		FlagshipIp:      flagshipIp,      // only set if not flagship
		FlagshipApiPort: flagshipApiPort, // only set if not flagship
		IsFlagship:      isFlagship,
	}, nil
}
