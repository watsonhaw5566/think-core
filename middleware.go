package think

import (
	"encoding/json"
	"fmt"
	thinkconfg "github.com/watsonhaw5566/think-core/config"
	"github.com/watsonhaw5566/think-core/log"
	thinkutil "github.com/watsonhaw5566/think-core/util"
	"net/http"
)

// recoveryMiddleware 全局异常捕获中间件
func recoveryMiddleware() HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				if e, ok := err.(Exception); ok {
					jsonStr, _ := json.Marshal(e)
					log.Log().Error(string(jsonStr))
					ctx.Fail(e.Message, FailOption{
						StatusCode: e.StateCode,
						ErrorCode:  e.ErrorCode,
					})
				} else {
					fmt.Println(err)
				}
			}
		}()
		ctx.Next()
	}
}

// fileServerMiddleware 静态资源服务中间件
func fileServerMiddleware() HandlerFunc {
	return func(ctx *Context) {
		// 如果是静态资源路径，使用文件服务器处理请求
		if thinkutil.HasSuffix(ctx.Request.RequestURI) {
			staticPrefix := thinkconfg.Config.Server.StaticPrefix
			if staticPrefix != "/" {
				staticPrefix = "/" + staticPrefix + "/"
			}
			http.StripPrefix(staticPrefix, http.FileServer(http.Dir(thinkconfg.Config.Server.StaticPath))).ServeHTTP(ctx.Response, ctx.Request)
			return
		}
		ctx.Next()
	}
}
