package tg

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"github.com/think-go/tg/tgcfg"
	"github.com/think-go/tg/tgutl"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

const StartText = `
  _______ _     _       _     _____  ____  
 |__   __| |   (_)     | |   / ____|/ __ \ 
    | |  | |__  _ _ __ | | _| |  __| |  | |
    | |  | '_ \| | '_ \| |/ / | |_ | |  | |
    | |  | | | | | | | |   <| |__| | |__| |
    |_|  |_| |_|_|_| |_|_|\_\\_____|\____/

`

// Engine 定义引擎结构体
type Engine struct {
	routerGroup
	pool sync.Pool
}

// New 初始化tg引擎
func New() *Engine {
	engine := &Engine{
		routerGroup: routerGroup{
			basePath:           "/",
			handlerFuncMap:     make(map[string]map[string]HandlerFunc),
			middlewaresFuncMap: make(map[string]map[string][]MiddlewareFunc),
		},
	}
	engine.pool.New = func() any {
		return engine.allocateContext()
	}
	return engine
}

func (engine *Engine) allocateContext() any {
	return &Context{
		engine:   engine,
		index:    -1,
		handlers: make([]HandlerFunc, 0),
	}
}

// methodHandle 执行handler和中间件
func (group *routerGroup) methodHandler(name string, method string, h HandlerFunc, ctx *Context) {
	if group.middlewares != nil {
		for i := len(group.middlewares) - 1; i >= 0; i-- {
			ctx.handlers = append(ctx.handlers, group.middlewares[i]())
		}
	}
	middlewaresFunc := group.middlewaresFuncMap[name][method]
	if middlewaresFunc != nil {
		for i := len(middlewaresFunc) - 1; i >= 0; i-- {
			ctx.handlers = append(ctx.handlers, middlewaresFunc[i]())
		}
	}
	ctx.handlers = append(ctx.handlers, h)
	ctx.Next()
}

// ServeHTTP
func (engine *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := engine.pool.Get().(*Context)
	ctx.Response = w
	ctx.Request = r
	method := r.Method
	_, ok := engine.handlerFuncMap[r.URL.Path][method]
	if ok || tgutl.HasSuffix(r.URL.Path) {
		engine.methodHandler(r.URL.Path, method, engine.handlerFuncMap[r.URL.Path][method], ctx)
	} else {
		ctx.Fail("路由不存在", FailOptions{
			StatusCode: http.StatusNotFound,
			ErrorCode:  http.StatusNotFound,
		})
	}
	engine.pool.Put(ctx)
}

// Run 独立使用启动, 在调用前自行绑定路由和控制器
func (engine *Engine) Run() {
	// 全局异常捕获中间件
	engine.Use(recoveryMiddleware)
	// 静态文件服务
	engine.Use(fileServerMiddleware)

	// http服务
	cmd := &http.Server{
		Addr:    tgcfg.Config.Server.Address,
		Handler: engine,
	}

	// 异步启动服务
	go func() {
		color.Yellow("[ThinkGO]服务正在启动...")
		if err := cmd.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			color.Red("[ThinkGO]服务启动失败")
			return
		}
	}()

	// 检查端口的通道
	done := make(chan bool, 1)
	// 检查端口是否已经打开
	go func() {
		for {
			conn, err := net.Dial("tcp", tgcfg.Config.Server.Address)
			if err == nil {
				conn.Close()
				fmt.Print(strings.TrimPrefix(StartText, "\n"))
				color.Green("[ThinkGO]服务启动成功")
				color.Blue(fmt.Sprintf("[ThinkGO]服务地址: http://127.0.0.1%s", tgcfg.Config.Server.Address))
				color.Blue(fmt.Sprintf("[ThinkGO]文档地址: http://127.0.0.1%s/api.json", tgcfg.Config.Server.Address))
				done <- true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// 监听超时
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		color.Red("[ThinkGO]服务启动超时,请手动重启")
		return
	}

	// 主 goroutine 继续执行
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cmd.Shutdown(ctx); err != nil {
		color.Red("[ThinkGO]服务未能正常关闭")
		return
	}
	color.Green("[ThinkGO]服务正常关闭")
}
