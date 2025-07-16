<p align="center" style="font-weight:bold;font-size:24px;padding-top:5px;">Thinko </p>
<p align="center" style="font-weight:bold;font-size:14px;padding-top:5px;">一款轻量级 GO WEB 应用框架 </p>

<p align="center">
    <img src="https://img.shields.io/github/v/release/watsonhaw5566/thinko.svg?style=flat-square">
    <img src="https://pkg.go.dev/badge/github.com/watsonahaw5566/thinko?status.svg">
  <br>
</p>

- 💪 Think ORM 思想链式操作 CRUD
- 🔥 应用级提炼封装更贴近业务场景
- 🚀 高效路由管理，支持灵活的URL映射
- 🛠️ 自动化的代码生成工具，快速搭建项目基础结构

## Thinko 框架

Thinko 是一款类 ThinkPHP 轻量级 GO WEB 框架，提供一套结构化、模块化的开发环境，为减少开发学习成本，提高团队的开发效率。

## 目录结构

```
thinko
├── cmd               // 命令行工具
│   └── think
│       └── main.go
├── config            // 配置文件
│   └── config.go
├── log               // 日志
│   └── logger.go
├── service           // 服务
│   └── server.go
├── token             // jwt相关
│   └── token.go
├── util             // 工具
│   └── utils.go
├── validate.go      // 验证器
├── README.md
├── context.go       // 中间件
├── go.mod
├── go.sum
├── middleware.go   // 中间件
├── mysql.go        // MySQL 数据库
├── redis.go        // Redis 数据库
├── router.go       // 路由
└── think.go        // 引擎
```

## 安装

#### 方式一

安装到您自己的项目中

```
go get -u github.com/watsonhaw55566/thinko
```

然后在您项目中可以像下面这样去编写

```
think.Config.Server.Address = ":8000"

engine := think.New()

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
git clone https://github.com/watsonhaw5566/thinko.git && cd think/cmd/think && go install
```

然后就可以在全局通过 ``think`` 命令去创建项目

```
think init demo
```

2.也或者可以直接克隆项目使用

```
git clone https://github.com/watsonhaw5566/thinko-template.git
```

安装依赖

```
go mod tidy
```

启动项目

```
go run main.go
```

## 说明

``Thinko`` 是基于 ``thinko`` 核心包构建基础工程项目，旨在为开发者提供一套结构化、模块化的开发环境。

``Thinko`` 精心设计了路由管理、中间件配置以及控制器实现等核心组件的组织方式与实现路径，确保了代码的高可读性和维护性。

通过明确规定各功能模块的存放目录及实现方法，不仅简化了项目的搭建过程，还极大地方便了后续的迭代与扩展，使团队协作更加高效顺畅。

无论是初学者还是有经验的开发者，都能在 ``Thinko`` 的帮助下快速上手，专注于业务逻辑的实现，而无需从零开始搭建项目架构。
