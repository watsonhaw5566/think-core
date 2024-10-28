package tg

import (
	"encoding/json"
	"fmt"
	"github.com/think-go/tg/tgcfg"
	"github.com/think-go/tg/tglog"
	"github.com/think-go/tg/tgutl"
	"net/http"
)

// recoveryMiddleware 全局异常捕获中间件
func recoveryMiddleware() HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				error := err.(*Exception)
				jsonStr, _ := json.Marshal(error)
				tglog.Log().Error(string(jsonStr))
				if error.Error != nil {
					fmt.Println(error.Error)
				}
				ctx.Fail(error.Message, FailOptions{
					StatusCode: error.StateCode,
					ErrorCode:  error.ErrorCode,
				})
			}
		}()
		ctx.Next()
	}
}

// fileServerMiddleware 静态资源服务中间件
func fileServerMiddleware() HandlerFunc {
	return func(ctx *Context) {
		// 如果是静态资源路径，使用文件服务器处理请求
		if tgutl.HasSuffix(ctx.Request.RequestURI) {
			staticPrefix := tgcfg.Config.Server.StaticPrefix
			if staticPrefix != "/" {
				staticPrefix = "/" + staticPrefix + "/"
			}
			http.StripPrefix(staticPrefix, http.FileServer(http.Dir(tgcfg.Config.Server.StaticPath))).ServeHTTP(ctx.Response, ctx.Request)
			return
		}
		ctx.Next()
	}
}
