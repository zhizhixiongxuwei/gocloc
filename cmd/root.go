// Package cmd 提供 gocloc 的命令行入口与子命令编排。
package cmd

import (
	"gocloc/internal/languages"

	"github.com/spf13/cobra"
)

// Execute 组装根命令并执行。
// version 参数由 main 包注入，便于在 CI/CD 中打包不同版本。
func Execute(version string) error {
	registry := languages.NewRegistry()
	rootCmd := newRootCmd(version, registry)
	return rootCmd.Execute()
}

// newRootCmd 创建根命令并注册全部子命令。
func newRootCmd(version string, registry *languages.Registry) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "gocloc",
		Short: "基于 FSM 的代码度量统计工具",
		Long: "gocloc 是一个基于有限状态机（FSM）的代码统计工具，\n" +
			"用于统计 total/code/comment/blank 行数，支持并发扫描与 JSON 导出。",
		SilenceUsage: true,
	}

	rootCmd.AddCommand(newVersionCmd(version))
	rootCmd.AddCommand(newLanguageCmd(registry))
	rootCmd.AddCommand(newScanCmd(registry))

	return rootCmd
}
