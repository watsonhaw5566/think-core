package tg

import (
	"fmt"
	"net/http"
	"path"
)

// HandlerFunc 定义http执行函数类型
type HandlerFunc func(ctx *Context)

// MiddlewareFunc 定义中间件函数类型
type MiddlewareFunc func() HandlerFunc

// RouterGroup 定义分组路由结构体
type routerGroup struct {
	basePath           string
	handlerFuncMap     map[string]map[string]HandlerFunc
	middlewaresFuncMap map[string]map[string][]MiddlewareFunc
	middlewares        []MiddlewareFunc
	groupMiddlewares   []MiddlewareFunc
}

// Use 添加中间件
func (group *routerGroup) Use(middlewareFunc ...MiddlewareFunc) {
	group.middlewares = append(group.middlewares, middlewareFunc...)
}

// Group 分组路由
func (group *routerGroup) Group(relativePath string, middlewareFunc ...MiddlewareFunc) *routerGroup {
	newRouterGroup := &routerGroup{
		basePath:           path.Join(group.basePath, relativePath),
		handlerFuncMap:     group.handlerFuncMap,
		middlewaresFuncMap: group.middlewaresFuncMap,
		groupMiddlewares:   middlewareFunc,
	}
	return newRouterGroup
}

// GET GET请求
func (group *routerGroup) GET(relativePath string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	group.handler(http.MethodGet, relativePath, handlerFunc, middlewareFunc...)
}

// POST POST请求
func (group *routerGroup) POST(relativePath string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	group.handler(http.MethodPost, relativePath, handlerFunc, middlewareFunc...)
}

// DELETE DELETE请求
func (group *routerGroup) DELETE(relativePath string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	group.handler(http.MethodDelete, relativePath, handlerFunc, middlewareFunc...)
}

// PUT PUT请求
func (group *routerGroup) PUT(relativePath string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	group.handler(http.MethodPut, relativePath, handlerFunc, middlewareFunc...)
}

// PATCH PATCH请求
func (group *routerGroup) PATCH(relativePath string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	group.handler(http.MethodPatch, relativePath, handlerFunc, middlewareFunc...)
}

// OPTIONS OPTIONS请求
func (group *routerGroup) OPTIONS(relativePath string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	group.handler(http.MethodOptions, relativePath, handlerFunc, middlewareFunc...)
}

// HEAD HEAD请求
func (group *routerGroup) HEAD(relativePath string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	group.handler(http.MethodHead, relativePath, handlerFunc, middlewareFunc...)
}

// ALL ALL请求
func (group *routerGroup) ALL(relativePath string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPut, http.MethodPatch, http.MethodOptions, http.MethodHead}
	for _, method := range methods {
		group.handler(method, relativePath, handlerFunc, middlewareFunc...)
	}
}

// Bind 绑定控制器,可以用api结构体的方式定义路由
func (group *routerGroup) Bind() {
	// var req api.SayHelloReq
	// ctx.BindStructValidate(&req)
	// 这功能实际就是省了这两行代码,多加一个参数,有空再写吧
}

// handler http绑定的函数
func (group *routerGroup) handler(method string, relativePath string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	mergePath := path.Join(group.basePath, relativePath)
	_, ok := group.handlerFuncMap[mergePath]
	if !ok {
		group.handlerFuncMap[mergePath] = make(map[string]HandlerFunc)
		group.middlewaresFuncMap[mergePath] = make(map[string][]MiddlewareFunc)
	}
	_, ok = group.handlerFuncMap[mergePath][method]
	if ok {
		panic(Exception{
			StateCode: http.StatusInternalServerError,
			ErrorCode: ErrorCode.EXCEPTION,
			Message:   fmt.Sprintf("路由重复 [%s][%s]", mergePath, method),
		})
	}
	group.handlerFuncMap[mergePath][method] = handlerFunc
	group.middlewaresFuncMap[mergePath][method] = append(append(group.middlewaresFuncMap[mergePath][method], group.groupMiddlewares...), middlewareFunc...)
}
