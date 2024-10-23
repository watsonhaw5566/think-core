package tgutl

import (
	"github.com/think-go/tg/tgcfg"
	"strings"
)

// HasSuffix 判断路由后缀是否在文件类型组里
func HasSuffix(url string) bool {
	staticPrefix := strings.Split(tgcfg.Config.Server.StaticSuffix, ",")
	for _, prefix := range staticPrefix {
		if strings.HasSuffix(url, "."+prefix) {
			return true
		}
	}
	return false
}
