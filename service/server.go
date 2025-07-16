package service

import (
	"github.com/watsonhaw5566/thinko"
)

// bindControllerFunc 绑定路由函数
type bindControllerFunc func(*think.Engine)

// Run 方便 think-go 框架绑定路由控制器用的启动方法
func Run(bindController ...bindControllerFunc) {
	// 创建 Think 引擎
	router := think.New()
	for _, controllerFunc := range bindController {
		controllerFunc(router)
	}
	router.Run()
}
