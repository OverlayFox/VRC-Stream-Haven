package types

// Flagship holds all information about the Flagship running the Haven.
type Flagship struct {
	Ship           *Escort `yaml:"ship"`
	SrtIngestPort  int     `yaml:"srtIngestPort"`
	RtmpIngestPort int     `yaml:"rtmpIngestPort"`
	ApiPort        int     `yaml:"apiPort"`
	Passphrase     string  `yaml:"passphrase"`
}
