package tgsv

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/think-go/tg/tgcfg"
	"github.com/think-go/tg/tglog"
	"github.com/think-go/tg/tgutl"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"
)

// HandlerFunc 回调函数
type HandlerFunc func(*gin.Engine) *gin.Engine

// RecoveryMiddleware 全局异常捕获中间件
func RecoveryMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 输出堆栈
				stack := make([]byte, 1024)
				length := runtime.Stack(stack, false)
				stackTrace := string(stack[:length])
				fmt.Println(stackTrace)
				// 日志记录
				tglog.Log().Error(err)
				tglog.Log().Error(stackTrace)
				// 输出信息
				tgutl.Fail(ctx, fmt.Sprintf("%v", err), tgutl.FailOptions{
					StatusCode: http.StatusInternalServerError,
					ErrorCode:  tgutl.ErrorCode.EXCEPTION,
				})
				ctx.Abort()
			}
		}()
		ctx.Next()
	}
}

func Run(handlerFunc HandlerFunc) {
	// 引入日志
	log := tglog.Log()

	// 设置启动模式
	gin.SetMode(tgcfg.Config.Server.Mode)

	// 创建Gin引擎
	router := gin.New()
	// 捕获panic并恢复执行
	router.Use(RecoveryMiddleware())
	// 静态资源路径
	router.StaticFS(tgcfg.Config.Server.StaticPath, http.Dir(tgcfg.Config.Server.StaticPath))

	// http服务
	cmd := &http.Server{
		Addr:    tgcfg.Config.Server.Address,
		Handler: handlerFunc(router),
	}

	fmt.Print(`
  _______ _     _       _     _____  ____  
 |__   __| |   (_)     | |   / ____|/ __ \ 
    | |  | |__  _ _ __ | | _| |  __| |  | |
    | |  | '_ \| | '_ \| |/ / | |_ | |  | |
    | |  | | | | | | | |   <| |__| | |__| |
    |_|  |_| |_|_|_| |_|_|\_\\_____|\____/
`)

	// 异步启动服务
	go func() {
		if err := cmd.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("服务启动成功")
		}
		fmt.Println("服务启动失败")
	}()

	// 主 goroutine 继续执行
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cmd.Shutdown(ctx); err != nil {
		log.Info("服务正常关闭")
	}
}
