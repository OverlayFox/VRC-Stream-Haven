package paths

type RegisterBody struct {
	IpAddress      string  `yaml:"ipAddress"`
	RtspEgressPort int     `yaml:"rtspEgressPort"`
	Latitude       float64 `yaml:"lat"`
	Longitude      float64 `yaml:"lon"`
	Username       string  `yaml:"username"`
	Passphrase     string  `yaml:"passphrase"`
}
