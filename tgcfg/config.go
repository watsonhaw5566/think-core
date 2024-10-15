package tgcfg

import (
	"gopkg.in/yaml.v3"
	"os"
)

var Config *config

// 服务相关配置
type server struct {
	Mode       string `yaml:"mode"`
	Address    string `yaml:"address"`
	StaticPath string `yaml:"staticPath"`
}

// 日志相关配置
type log struct {
	Path   string `yaml:"path"`
	Name   string `yaml:"name"`
	Model  string `yaml:"model"`
	MaxAge int    `yaml:"maxAge"`
}

// 配置
type config struct {
	Server server `yaml:"server"`
	Log    log    `yaml:"log"`
}

func init() {
	yamlFile, err := os.ReadFile("./config/config.yaml")
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(yamlFile, &Config)
	if err != nil {
		panic(err)
	}
}
