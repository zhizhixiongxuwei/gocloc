package cmd

import "github.com/spf13/cobra"

// newVersionCmd 创建 version 子命令。
// 命令示例：gocloc version
func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示当前版本号",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Printf("gocloc version %s\n", version)
		},
	}
}
