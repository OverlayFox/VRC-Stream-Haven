package types

// Flagship holds all information about the Flagship running the Haven.
type Flagship struct {
	Ship           *Escort `yaml:"ship"`
	SrtIngestPort  uint16  `yaml:"srtIngestPort"`
	RtmpIngestPort uint16  `yaml:"rtmpIngestPort"`
	ApiPort        uint16  `yaml:"apiPort"`
	Passphrase     string  `yaml:"passphrase"`
}
