package tgcfg

import (
	"gopkg.in/yaml.v3"
	"os"
)

var Config *config

// 服务相关配置
type server struct {
	Address      string `yaml:"address"`
	TplPath      string `yaml:"tplPath"`
	StaticPrefix string `yaml:"staticPrefix"`
	StaticPath   string `yaml:"staticPath"`
	StaticSuffix string `yaml:"staticSuffix"`
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
	Server server                 `yaml:"server"`
	Log    log                    `yaml:"log"`
	Extra  map[string]interface{} `yaml:",inline"`
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
