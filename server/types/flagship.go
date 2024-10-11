package types

// Flagship holds all information about the Flagship running the Haven.
type Flagship struct {
	Ship          *Escort `yaml:"ship"`
	SrtIngestPort uint16  `yaml:"srtIngestPort"`
	Application   string  `yaml:"application"`
	Passphrase    string  `yaml:"passphrase"`
}
