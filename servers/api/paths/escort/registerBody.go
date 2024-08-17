package escort

type RegisterBody struct {
	IpAddress      string  `yaml:"publicIpAddress"`
	RtspEgressPort int     `yaml:"rtspEgressPort"`
	Latitude       float64 `yaml:"lat"`
	Longitude      float64 `yaml:"lon"`
	Username       string  `yaml:"username"`
	Passphrase     string  `yaml:"passphrase"`
}
