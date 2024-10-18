package types

// Flagship holds all information about the Flagship running the Haven.
type Flagship struct {
	Escort
	SrtIngestPort uint16 `yaml:"srtIngestPort"`
	Passphrase    string `yaml:"passphrase"`
}
