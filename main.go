// main.go 是 gocloc 的程序入口。
// 该文件仅负责注入版本号并执行 Cobra 根命令，
// 让业务逻辑保持在 cmd/internal 目录中，便于测试和扩展。
package main

import (
	"fmt"
	"os"

	"gocloc/cmd"
)

// version 默认值为 dev。
// 发布时可以通过 -ldflags "-X main.version=vX.Y.Z" 覆盖该值。
var version = "dev"

func main() {
	if err := cmd.Execute(version); err != nil {
		fmt.Fprintf(os.Stderr, "gocloc error: %v\n", err)
		os.Exit(1)
	}
}
