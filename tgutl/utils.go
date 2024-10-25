package tgutl

import (
	"github.com/think-go/tg/tgcfg"
	"reflect"
	"strconv"
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

// ConvType 类型转换
func ConvType(t reflect.Type, value string) any {
	switch t.Kind() {
	case reflect.String:
		return value
	case reflect.Int:
		v, err := strconv.Atoi(value)
		if err != nil {
			panic(err)
		}
		return v
	case reflect.Int64:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			panic(err)
		}
		return v
	case reflect.Float32:
		v, err := strconv.ParseFloat(value, 32)
		if err != nil {
			panic(err)
		}
		return float32(v)
	case reflect.Float64:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			panic(err)
		}
		return v
	case reflect.Bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			panic(err)
		}
		return v
	default:
		panic("类型转换错误")
	}
}
