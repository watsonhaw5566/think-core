package tg

import (
	"fmt"
	"github.com/think-go/tg/tglog"
	"net/http"
	"runtime"
)

// RecoveryMiddleware 全局异常捕获中间件
func RecoveryMiddleware() HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				// 输出堆栈
				stack := make([]byte, 1024)
				length := runtime.Stack(stack, false)
				stackTrace := string(stack[:length])
				// 控制台打印堆栈
				fmt.Println(stackTrace)
				// 日志记录
				tglog.Log().Error(err)
				tglog.Log().Error(stackTrace)
				// 输出信息
				ctx.Fail(fmt.Sprintf("%v", err), FailOptions{
					StatusCode: http.StatusInternalServerError,
					ErrorCode:  ErrorCode.EXCEPTION,
				})
			}
		}()
		ctx.Next()
	}
}
