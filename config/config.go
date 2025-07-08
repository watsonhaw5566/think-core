package config

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
	Redis  map[string]interface{} `yaml:"redis"`
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

// GetRedisSource 获取数据源
func (conf *config) GetRedisSource(key string) gjson.Result {
	extraJSON, err := json.Marshal(conf.Redis)
	if err != nil {
		return gjson.Result{}
	}
	return gjson.Get(string(extraJSON), key)
}

// Get 获取自定义额外配置
func (conf *config) Get(key string) gjson.Result {
	extraJSON, err := json.Marshal(conf.Extra)
	if err != nil {
		return gjson.Result{}
	}
	return gjson.Get(string(extraJSON), key)
}

func init() {
	Config = &config{}
	yamlFile, err := os.ReadFile("./config/config.yaml")
	if err != nil {
		return
	}
	err = yaml.Unmarshal(yamlFile, &Config)
	if err != nil {
		panic(err)
	}
}
