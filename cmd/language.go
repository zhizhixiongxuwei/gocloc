package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"gocloc/internal/languages"

	"github.com/spf13/cobra"
)

// newLanguageCmd 创建 language 子命令。
// 命令用于展示当前已经实现的语言以及对应文件后缀。
func newLanguageCmd(registry *languages.Registry) *cobra.Command {
	return &cobra.Command{
		Use:   "language",
		Short: "展示已实现语言及后缀",
		RunE: func(cmd *cobra.Command, _ []string) error {
			writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)

			if _, err := fmt.Fprintln(writer, "LANGUAGE\tEXTENSIONS"); err != nil {
				return err
			}

			for _, item := range registry.Languages() {
				if _, err := fmt.Fprintf(writer, "%s\t%s\n", item.Name, strings.Join(item.Extensions, ", ")); err != nil {
					return err
				}
			}

			return writer.Flush()
		},
	}
}
