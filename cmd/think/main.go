package main

import (
	"database/sql"
	"fmt"
	"github.com/fatih/color"
	"github.com/iancoleman/strcase"
	"github.com/spf13/cobra"
	"github.com/watsonhaw5566/think-core"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const ThinkCli = `
  _______ _     _       _     _____ _      _____ 
 |__   __| |   (_)     | |   / ____| |    |_   _|
    | |  | |__  _ _ __ | | _| |    | |      | |  
    | |  | '_ \| | '_ \| |/ / |    | |      | |  
    | |  | | | | | | | |   <| |____| |____ _| |_ 
    |_|  |_| |_|_|_| |_|_|\_\\_____|______|_____|
[Think-CLI] 用于快速创建项目,快速生成 controller, model , service 通用代码的工具
`

const EntityTemplate = `
package entity

// %s 有NULL值的注意加 * 防止转换报错
type %s struct {
	%s
}
`

const ApiTemplate = "package api\n\n" +
	"type Create%sReq struct{}\n\n" +
	"type Delete%sReq struct {\n" +
	"    Id int `p:\"id\" v:\"required\"`\n" +
	"}\n\n" +
	"type Edit%sReq struct {\n" +
	"    Id int `p:\"id\" v:\"required\"`\n" +
	"}\n\n" +
	"type %sListReq struct {\n" +
	"    Id       int `p:\"id\"`\n" +
	"    PageNum  int `p:\"pageNum\"`\n" +
	"    PageSize int `p:\"pageSize\"`\n" +
	"}\n\n" +
	"type %sListRes struct {\n" +
	"    List  []entity.User `json:\"list\"`\n" +
	"    Total int           `json:\"total\"`\n" +
	"}"

const ControllerTemplate = `
package controller

func Create%s(ctx *think.Context) {
	var req api.Create%sReq
	ctx.BindStructValidate(&req)
	err := service.%sService().Create%s(req)
	if err != nil {
		ctx.Fail("创建失败")
		return
	}
	ctx.Success("ok")
}

func Delete%s(ctx *think.Context) {
	var req api.Delete%sReq
	ctx.BindStructValidate(&req)
	err := service.%sService().Delete%s(req.Id)
	if err != nil {
		ctx.Fail("删除失败")
		return
	}
	ctx.Success("ok")
}

func Edit%s(ctx *think.Context) {
	var req api.Edit%sReq
	ctx.BindStructValidate(&req)
	err := service.%sService().Edit%s(req)
	if err != nil {
		ctx.Fail("更新失败")
		return
	}
	ctx.Success("ok")
}

func %sList(ctx *think.Context) {
	var req api.%sListReq
	ctx.BindStructValidate(&req)
	res, err := service.%sService().%sList(req)
	if err != nil {
		ctx.Fail("查询失败")
		return
	}
	ctx.Success(res)
}
`

const ServiceTemplate = `
package service

type i%sService interface {
	Create%s(req api.Create%sReq) (err error)
	Delete%s(id int) (err error)
	Edit%s(req api.Edit%sReq) (err error)
	%sList(req api.%sListReq) (res api.%sListRes, err error)
}

type %sServiceImpl struct{}

func (i %sServiceImpl) CreateUser(req api.Create%sReq) (err error) {
	err = dao.Create%s(req)
	return
}

func (i %sServiceImpl) Delete%s(id int) (err error) {
	err = dao.Delete%s(id)
	return
}

func (i %sServiceImpl) Edit%s(req api.Edit%sReq) (err error) {
	err = dao.Edit%s(req)
	return
}

func (i %sServiceImpl) %sList(req api.%sListReq) (res api.%sListRes, err error) {
	res, err = dao.%sList(req)
	return
}

func %sService() i%sService {
	return &%sServiceImpl{}
}
`

const DaoTemplate = `
package dao

func Create%s(req api.Create%sReq) (err error) {
	_, err = think.Db("%s").Insert(req)
	return
}

func Delete%s(id int) (err error) {
	err = think.Db("%s").Where("id", "=", id).Delete()
	return
}

func Edit%s(req api.Edit%sReq) (err error) {
	err = think.Db("%s").Where("id", "=", req.Id).Update(req)
	return
}

func %sList(req api.%sListReq) (res api.%sListRes, err error) {
	m := think.Db("%s")
	if req.PageNum == 0 {
		req.PageNum = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}
	if req.Id != 0 {
		m = m.Where("id", "=", req.Id)
	}
	var user []entity.%s
	count, err := m.Count()
	err = m.Page(req.PageNum, req.PageSize).Select(&user)
	res.List = user
	res.Total = count
	return
}
`

func main() {
	root := &cobra.Command{
		Use:   "think",
		Short: "Think GO 框架命令行工具",
		Long:  ThinkCli,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
			}
		},
	}

	// 添加子命令
	root.AddCommand(initCmd(), createCmd(), syncCmd(), versionCmd())

	// 添加简写命令
	root.PersistentFlags().BoolP("help", "h", false, "帮助文档")
	root.PersistentFlags().BoolP("init", "i", false, "初始化一个项目")
	root.PersistentFlags().BoolP("create", "c", false, "创建一个业务模块")
	root.PersistentFlags().BoolP("sync", "s", false, "同步数据库表结构体")
	root.PersistentFlags().BoolP("version", "v", false, "查看版本信息")

	// 自定义帮助描述
	root.SetHelpCommand(&cobra.Command{
		Use:   "help",
		Short: "帮助文档",
		Long:  "帮助文档",
	})

	if err := root.Execute(); err != nil {
		fmt.Println(err)
	}
}

// initCmd 创建项目
func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "初始化一个项目",
		Long:  "可通过[think init 项目名]的方式初始化项目",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				color.Yellow("[Think CLI] 请输入项目名称")
				return
			}
			git := exec.Command("git", "clone", "https://github.com/think-go/think-go.git", args[0])
			git.Stdout = os.Stdout
			git.Stderr = os.Stderr
			if err := git.Run(); err != nil {
				color.Red("[Think CLI]初始化项目失败")
				return
			}
			color.Green(fmt.Sprintf("初始化项目[%s]成功", args[0]))
		},
	}
}

// createCmd 创建业务模块
func createCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "创建一个业务模块",
		Long:  "可通过[think create 模块名]的方式快速创建 api,controller,dao,service 示例文件,可根据具体业务在其之上修改",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				color.Yellow("[Think CLI]你没有输入模块名称")
				return
			}
			createModule(args[0], ApiTemplate, 5, "api")
			createModule(args[0], ControllerTemplate, 16, "app/controller")
			createModule(args[0], ServiceTemplate, 28, "app/service")
			createModule(args[0], DaoTemplate, 13, "app/dao")
		},
	}
}

func getModuleName(n int, name string) []interface{} {
	var result []interface{}
	for i := 0; i < n; i++ {
		result = append(result, strcase.ToCamel(name))
	}
	return result
}

func createModule(name string, template string, num int, url string) {
	code := fmt.Sprintf(template, getModuleName(num, name)...)
	code = strings.Replace(code, fmt.Sprintf(`"%s"`, strcase.ToCamel(name)), fmt.Sprintf(`"%s"`, name), 1)
	structCode, err := format.Source([]byte(code))
	if err != nil {
		color.Red("代码格式化出错%v", err)
		return
	}
	path := filepath.Join(url, fmt.Sprintf("%s.go", name))
	err = os.WriteFile(path, structCode, 7777)
	if err != nil {
		color.Red("文件写入出错%v", err)
		return
	}
	color.Green("生成%s文件成功", path)
}

// syncCmd 同步数据库表结构体
func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "同步数据库表结构体",
		Long:  "可通过[think sync 表名]的方式快速在entity文件夹下生成对应数据库表的结构体文件，不传表名默认是所有表",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				db := think.ExecSql()
				rows, err := db.Queryx("SHOW TABLES")
				if err != nil {
					color.Red("数据库连接错误%v", err)
					return
				}
				defer rows.Close()

				for rows.Next() {
					var tableName string
					if err = rows.Scan(&tableName); err != nil {
						color.Red("数据表读取出错%v", err)
						return
					}
					createCode(tableName)
				}
				return
			} else {
				createCode(args[0])
			}
		},
	}
}

// createCode 生成代码
func createCode(tableName string) {
	db := think.ExecSql()
	rows, err := db.Queryx(fmt.Sprintf("DESCRIBE %s", tableName))
	if err != nil {
		color.Red("数据表读取出错%v", err)
		return
	}
	defer rows.Close()

	typeStr := ""
	for rows.Next() {
		var columnName, columnType, null, key, extra string
		var defaultVal sql.NullString
		err = rows.Scan(&columnName, &columnType, &null, &key, &defaultVal, &extra)
		if err != nil {
			color.Red("数据表字段读取出错%v", err)
			return
		}

		goType, ok := sqlToGoType(columnType, null)
		if !ok {
			color.Red("无法识别数据类型%s", columnType)
			return
		}

		typeStr += fmt.Sprintf("%s %s `db:\"%s\" json:\"%s\"`\n", strcase.ToCamel(columnName), goType, strcase.ToSnake(columnName), strcase.ToSnake(columnName))
	}

	code := fmt.Sprintf(EntityTemplate, strcase.ToCamel(tableName), strcase.ToCamel(tableName), typeStr)

	structCode, err := format.Source([]byte(code))
	if err != nil {
		color.Red("代码格式化出错%v", err)
		return
	}

	path := filepath.Join("app/entity", fmt.Sprintf("%s.go", tableName))
	err = os.WriteFile(path, structCode, 7777)
	if err != nil {
		color.Red("文件写入出错%v", err)
		return
	}
	color.Green("生成%s文件成功", path)
}

// SQL数据类型转GO数据类型
func sqlToGoType(sqlType string, null string) (goType string, ok bool) {
	switch {
	case strings.HasPrefix(sqlType, "int"):
		goType = ""
		if null == "YES" {
			goType += "*"
		}
		goType += "int"
	case strings.HasPrefix(sqlType, "bigint"):
		goType = ""
		if null == "YES" {
			goType += "*"
		}
		goType += "int64"
	case strings.HasPrefix(sqlType, "varchar") || strings.HasPrefix(sqlType, "text") || strings.HasPrefix(sqlType, "char") || strings.HasPrefix(sqlType, "longtext"):
		goType = ""
		if null == "YES" {
			goType += "*"
		}
		goType += "string"
	case strings.HasPrefix(sqlType, "datetime") || strings.HasPrefix(sqlType, "timestamp"):
		goType = ""
		if null == "YES" {
			goType += "*"
		}
		goType += "time.Time"
	case strings.HasPrefix(sqlType, "float"):
		goType = ""
		if null == "YES" {
			goType += "*"
		}
		goType += "float32"
	case strings.HasPrefix(sqlType, "double"):
		goType = ""
		if null == "YES" {
			goType += "*"
		}
		goType += "float64"
	default:
		return "", false
	}
	return goType, true
}

// versionCmd 显示版本
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "查看版本信息",
		Long:  "查看 GO版本,ThinkGO 版本, Think CLI 版本",
		Run: func(cmd *cobra.Command, args []string) {
			goVersion := runtime.Version()
			fmt.Printf("当前环境GO版本: %s\n", goVersion)
			fmt.Printf("当前环境Think CLI版本: %s\n", "v1.0.0")
			fmt.Printf("当前项目 ThinkGO 版本: %s\n", "v1.0.0")
		},
	}
}
