package libraries

import (
	"gopkg.in/yaml.v3"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type NodeStruct struct {
	IpAddress     string `yaml:"publicIpAddress"`
	StreamingPort string `yaml:"streamingPort"`
	VpnPort       string `yaml:"vpnPort"`
}

type ConfigStruct struct {
	Nodes  []NodeStruct `yaml:"nodes"`
	Server NodeStruct   `yaml:"server"`
}

var Config ConfigStruct

func InitialiseConfig() *ConfigStruct {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(data, &Config)
	if err != nil {
		log.Fatal(err)
	}

	Config.Nodes = append(Config.Nodes, Config.Server)

	return &Config
}

func InitEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}
