package types

// Haven combines the ServerStruct and NodeStruct information.
type Haven struct {
	Escorts  *[]*Escort `yaml:"nodes"`
	Flagship *Flagship  `yaml:"server"`
	IsServer bool       `yaml:"isServer"`
}
