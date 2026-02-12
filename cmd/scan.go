package cmd

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	"gocloc/internal/languages"
	"gocloc/internal/report"
	"gocloc/internal/scanner"

	"github.com/spf13/cobra"
)

// scanOptions 存放 scan 命令的可配置参数。
type scanOptions struct {
	format  string
	output  string
	workers int
}

// newScanCmd 创建 scan 子命令。
// 示例：
//
//	gocloc scan .
//	gocloc scan ./project --format json --output result.json
func newScanCmd(registry *languages.Registry) *cobra.Command {
	options := scanOptions{
		format:  "table",
		output:  "output.json",
		workers: runtime.NumCPU(),
	}

	scanCmd := &cobra.Command{
		Use:   "scan [path]",
		Short: "扫描目录或文件并输出代码度量信息",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format := strings.ToLower(strings.TrimSpace(options.format))
			if format != "table" && format != "json" {
				return errors.New("unsupported format, allowed values: table, json")
			}

			if options.workers <= 0 {
				return errors.New("workers must be greater than 0")
			}

			service := scanner.NewService(registry, options.workers)
			result, err := service.ScanPath(args[0])
			if err != nil {
				return err
			}

			switch format {
			case "table":
				return report.PrintTable(cmd.OutOrStdout(), result)
			case "json":
				if err := report.PrintJSON(cmd.OutOrStdout(), result); err != nil {
					return err
				}

				outputPath := strings.TrimSpace(options.output)
				if outputPath == "" {
					outputPath = "output.json"
				}
				if err := report.WriteJSONFile(outputPath, result); err != nil {
					return err
				}

				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nJSON exported to %s\n", outputPath)
				return nil
			default:
				return errors.New("unsupported format")
			}
		},
	}

	scanCmd.Flags().StringVar(&options.format, "format", options.format, "输出格式: table 或 json")
	scanCmd.Flags().StringVar(&options.output, "output", options.output, "json 导出文件路径，默认 output.json")
	scanCmd.Flags().IntVar(&options.workers, "workers", options.workers, "并发 worker 数量")

	return scanCmd
}
