<p align="center">
  <img width="300px" src="https://www.think-ts.cn/icon.png">
</p>

<p align="center">
  <a href="https://www.think-go.com.cn">
    <img src="https://img.shields.io/github/release/think-go/tg.svg?style=flat-square">
  </a>
  <a href="https://www.think-go.com.cn">
    <img src="https://pkg.go.dev/badge/github.com/think-go/tg?status.svg">
  </a>
  <a href="https://www.think-go.com.cn">
    <img src="https://codecov.io/gh/think-go/tg/branch/master/graph/badge.svg"/>
  </a>
  <a href="https://www.think-go.com.cn">
    <img src="https://img.shields.io/badge/%E4%BD%9C%E8%80%85-zhangyu-7AD6FD.svg"/>
  </a>
  <br>
</p>

<p align="center">一个轻量级的GO WEB应用框架</p>

- 💪 ORM思想链式操作CRUD
- 🔥 应用级提炼封装更贴近业务场景

## ThinkGO框架

[ThinkGO](https://www.think-go.com.cn) 是一个轻量级的GO WEB应用框架，整合了各种常用SDK以及企业级常用的技术方案，为减少了开发人员的学习成本，提高团队的开发效率而生。

## 目录结构

```
tg
├── cmd                // 命令行工具
│   └── tg
│       └── main.go
├── tgcfg              // 配置文件
│   └── config.go
├── tglog              // 日志
│   └── logger.go
├── tgsv               // 服务
│   └── server.go
├── tgtoken            // jwt相关
│   └── token.go
├── tgutl              // 工具
│   └── utils.go
├── validate.go        // 验证器
├── README.md
├── context.go         // 链路
├── go.mod
├── go.sum
├── middleware.go      // 中间件
├── mysql.go           // mysql数据库
├── router.go          // 路由
└── tg.go              // 引擎
```

## 安装

#### 方式一

安装到您自己的项目中

```
go get -u github.com/think-go/tg
```

然后在您项目中可以像下面这样去编写

```
engine := tg.New()
engine.GET("/hello", func(ctx *tg.Context) {
  ctx.Success("ok")
})
router := engine.Group("/api/v1")
{
  router.GET("user/list", func(ctx *tg.Context) {
    ctx.Success("ok")
  })
  router.POST("user/delete", func(ctx *tg.Context) {
    ctx.Success("ok")
  })
}
engine.Run()
```

#### 方式二

通过框架工程去编写

1.通过命令行去初始化项目,先安装命令行工具

```
git clone https://github.com/think-go/tg.git && cd tg/cmd/tg && go install
```

然后就可以在全局通过 ``tg`` 命令去创建项目

```
tg init demoApp
```

2.也或者可以直接克隆项目使用

```
git clone https://github.com/think-go/think-go.git
```

安装依赖

```
go mod tidy
```

启动项目

```
go run main.go
```

<p align="center">
  <img src="https://think-go.com.cn/think-go.png">
</p>


## 说明

``think-go`` 是基于 ``tg`` 核心包构建的基础工程项目，旨在为开发者提供一套结构化、模块化的开发环境。``think-go`` 精心设计了路由管理、中间件配置以及控制器实现等核心组件的组织方式与实现路径，确保了代码的高可读性和维护性。通过明确规定各功能模块的存放目录及实现方法，不仅简化了项目的搭建过程，还极大地方便了后续的迭代与扩展，使团队协作更加高效顺畅。无论是初学者还是有经验的开发者，都能在 ``think-go`` 的帮助下快速上手，专注于业务逻辑的实现，而无需从零开始搭建项目架构。
