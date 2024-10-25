package tgsv

import (
	"github.com/think-go/tg"
)

// bindControllerFunc 绑定路由函数
type bindControllerFunc func(*tg.Engine)

// Run 方便think-go框架绑定路由控制器用的启动方法
func Run(bindController ...bindControllerFunc) {
	// 创建TG引擎
	router := tg.New()
	for _, controllerFunc := range bindController {
		controllerFunc(router)
	}
	router.Run()
}
