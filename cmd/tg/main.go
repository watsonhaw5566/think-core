package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"runtime"
)

const ThinkCli = `
  _______ _____         _____ _      _____ 
 |__   __/ ____|       / ____| |    |_   _|
    | | | |  __ ______| |    | |      | |  
    | | | | |_ |______| |    | |      | |  
    | | | |__| |      | |____| |____ _| |_ 
    |_|  \_____|       \_____|______|_____|

[TG-CLI]用于快速创建项目,快速生成controller,dao,entity,service通用代码的工具
`

func main() {
	root := &cobra.Command{
		Use:   "tg",
		Short: "ThinkGO框架命令行工具",
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
		Long:  "可通过[tg init 项目名]的方式初始化项目",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				color.Yellow("[TG-CLI]你没有输入项目名称")
				return
			}
			git := exec.Command("git", "clone", "https://github.com/zy598586050/think-ts.git", args[0])
			git.Stdout = os.Stdout
			git.Stderr = os.Stderr
			if err := git.Run(); err != nil {
				color.Red("[TG-CLI]初始化项目失败")
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
		Long:  "可通过[tg create 模块名]的方式快速创建controller,dao,entity,service文件",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				color.Yellow("[TG-CLI]你没有输入模块名称")
				return
			}
		},
	}
}

// syncCmd 同步数据库表结构体
func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "同步数据库表结构体",
		Long:  "可通过[tg sync 表名]的方式快速在entity文件夹下生成对应数据库表的结构体文件，不传表名默认是所有表",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				color.Yellow("[TG-CLI]你没有输入表名")
				return
			}
		},
	}
}

// versionCmd 显示版本
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "查看版本信息",
		Long:  "查看GO版本,ThinkGO版本,TG-CLI版本",
		Run: func(cmd *cobra.Command, args []string) {
			goVersion := runtime.Version()
			fmt.Printf("当前环境GO版本: %s\n", goVersion)
			fmt.Printf("当前环境TG-CLI版本: %s\n", "v1.0.0")
			fmt.Printf("当前项目ThinkGO版本: %s\n", "v1.0.0")
		},
	}
}
