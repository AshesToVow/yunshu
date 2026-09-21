package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configPath string

var rootCmd = &cobra.Command{
	Use:   "yunshu",
	Short: "A cloud platform built with Gin, MySQL, Redis and Casbin",
	// 运行期错误（如配置校验失败）不应打印 flag 用法说明——那会把真正的
	// 报错顶到屏幕上方、看起来像参数写错了。用法说明只在参数解析失败时打印。
	SilenceUsage: true,
	// cobra 默认会自己打印一遍 error，Execute() 里又打印一遍，导致同一条
	// 报错重复输出两次；统一由 Execute() 负责输出。
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "configs/config.yaml", "config file path")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
