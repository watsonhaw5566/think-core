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
		panic(fmt.Sprintf("路由重复 [%s][%s]", mergePath, method))
	}
	group.handlerFuncMap[mergePath][method] = handlerFunc
	group.middlewaresFuncMap[mergePath][method] = append(append(group.middlewaresFuncMap[mergePath][method], group.groupMiddlewares...), middlewareFunc...)
}
