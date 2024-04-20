package libraries

import (
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

type NodeStruct struct {
	IsoCountry    string `yaml:"isoCountry"`
	IpAddress     string `yaml:"publicIpAddress"`
	StreamingPort string `yaml:"streamingPort"`
	VpnPort       string `yaml:"vpnPort"`
}

type NodesStruct struct {
	Nodes []NodeStruct `yaml:"nodes"`
}

func ReadNodes() NodesStruct {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	var nodes NodesStruct
	err = yaml.Unmarshal(data, &nodes)
	if err != nil {
		log.Fatal(err)
	}

	return nodes
}
