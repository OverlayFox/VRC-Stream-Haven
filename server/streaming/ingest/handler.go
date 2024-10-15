package ingest

import (
	"errors"
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/types"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
)

// startMediaMtx starts the mediaMTX service
func startMediaMtx() error {
	cmd := exec.Command("supervisorctl", "start", "mediamtx")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s, output: %s", err, string(output))
	}
	return nil
}

// stopMediaMtx stops the mediaMTX service
func stopMediaMtx() error {
	cmd := exec.Command("supervisorctl", "stop", "mediamtx")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s, output: %s", err, string(output))
	}
	return nil
}

// isMediaMtxRunning checks if the mediaMTX service is running
func isMediaMtxRunning() (bool, error) {
	cmd := exec.Command("supervisorctl", "status", "mediamtx")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// supervisorctl returns status codes in std-err https://refspecs.linuxbase.org/LSB_3.0.0/LSB-PDA/LSB-PDA/iniscrptact.html
		var exitError *exec.ExitError
		ok := errors.As(err, &exitError)
		if ok {
			if exitError.ExitCode() == 1 || exitError.ExitCode() == 2 || exitError.ExitCode() == 3 {
				return false, nil
			} else if exitError.ExitCode() == 0 {
				return true, nil
			}
		}
	}

	return false, fmt.Errorf("could not extract state. Std-Output: %s Std-Error: %s", string(output), err)
}

func startMediaMtxWithConfig(config []byte) error {
	running, err := isMediaMtxRunning()
	if err != nil {
		return fmt.Errorf("could not get status of mediaMtx: %s", err)
	}

	if running {
		err := stopMediaMtx()
		if err != nil {
			return fmt.Errorf("could not stop mediaMtx: %s", err)
		}
	}

	err = os.WriteFile(os.Getenv("MEDIA_MTX_CONFIG_PATH"), config, 0644)
	if err != nil {
		return fmt.Errorf("could not write mediaMTX config: %s", err)
	}

	err = startMediaMtx()
	if err != nil {
		return fmt.Errorf("could not start mediaMtx: %s", err)
	}

	return nil
}

// InitFlagshipIngest initializes mediaMTX with a config that allows a user to push an SRT stream to the server
func InitFlagshipIngest() error {
	var config types.MediaMtxConfig
	var paths types.Paths
	paths = config.BuildFlagshipPath(harbor.Haven.Flagship.Passphrase)
	config = config.BuildConfig(harbor.Haven.Flagship.Passphrase, paths)

	newData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("could not marshal mediaMTX config: %s", err)
	}

	return startMediaMtxWithConfig(newData)
}

// InitEscortIngest initializes mediaMTX with a config that pulls an SRT stream from the Flagship
func InitEscortIngest() error {
	var config types.MediaMtxConfig
	var paths types.Paths
	paths = config.BuildEscortPath(harbor.Haven.Flagship)
	config = config.BuildConfig(harbor.Haven.Flagship.Passphrase, paths)

	newData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("could not marshal mediaMTX config: %s", err)
	}

	return startMediaMtxWithConfig(newData)
}
