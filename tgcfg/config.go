package tgcfg

import (
	"encoding/json"
	"github.com/tidwall/gjson"
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
	MySql  map[string]interface{} `yaml:"mysql"`
	Extra  map[string]interface{} `yaml:"extra"`
}

// GetMySqlSource 获取数据源
func (conf *config) GetMySqlSource(key string) gjson.Result {
	extraJSON, err := json.Marshal(conf.MySql)
	if err != nil {
		return gjson.Result{}
	}
	return gjson.Get(string(extraJSON), key)
}

// Get 获取配置
func (conf *config) Get(key string) gjson.Result {
	extraJSON, err := json.Marshal(conf.Extra)
	if err != nil {
		return gjson.Result{}
	}
	return gjson.Get(string(extraJSON), key)
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
