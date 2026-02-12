// Package report 提供 gocloc 的输出能力。
// 当前实现支持 table 控制台格式和 JSON 格式（含文件导出）。
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	"gocloc/internal/model"
)

// PrintTable 使用表格展示扫描结果。
func PrintTable(writer io.Writer, result model.ScanResult) error {
	tw := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)

	if _, err := fmt.Fprintf(tw, "SCANNED PATH\t%s\n\n", result.ScannedPath); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(tw, "FILE\tLANGUAGE\tTOTAL\tCODE\tCOMMENT\tBLANK"); err != nil {
		return err
	}
	for _, item := range result.Files {
		if _, err := fmt.Fprintf(
			tw,
			"%s\t%s\t%d\t%d\t%d\t%d\n",
			item.Path,
			item.Language,
			item.Metrics.Total,
			item.Metrics.Code,
			item.Metrics.Comment,
			item.Metrics.Blank,
		); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(tw, "\nLANGUAGE\tFILES\tTOTAL\tCODE\tCOMMENT\tBLANK"); err != nil {
		return err
	}
	for _, item := range result.Languages {
		if _, err := fmt.Fprintf(
			tw,
			"%s\t%d\t%d\t%d\t%d\t%d\n",
			item.Language,
			item.Files,
			item.Metrics.Total,
			item.Metrics.Code,
			item.Metrics.Comment,
			item.Metrics.Blank,
		); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(
		tw,
		"\nTOTAL\t%d\t%d\t%d\t%d\t%d\n",
		result.Total.Files,
		result.Total.Total,
		result.Total.Code,
		result.Total.Comment,
		result.Total.Blank,
	); err != nil {
		return err
	}

	if len(result.Errors) > 0 {
		if _, err := fmt.Fprintln(tw, "\nERROR FILE\tMESSAGE"); err != nil {
			return err
		}
		for _, item := range result.Errors {
			if _, err := fmt.Fprintf(tw, "%s\t%s\n", item.Path, item.Error); err != nil {
				return err
			}
		}
	}

	return tw.Flush()
}

// PrintJSON 把扫描结果按易读 JSON 输出到任意 writer。
func PrintJSON(writer io.Writer, result model.ScanResult) error {
	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	if _, err := writer.Write(content); err != nil {
		return fmt.Errorf("write json: %w", err)
	}
	return nil
}

// WriteJSONFile 将 JSON 结果导出到指定路径。
// 如果目录不存在会自动创建。
func WriteJSONFile(path string, result model.ScanResult) error {
	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	directory := filepath.Dir(path)
	if directory != "." && directory != "" {
		if mkErr := os.MkdirAll(directory, 0o755); mkErr != nil {
			return fmt.Errorf("create output directory: %w", mkErr)
		}
	}

	if writeErr := os.WriteFile(path, content, 0o644); writeErr != nil {
		return fmt.Errorf("write output file: %w", writeErr)
	}
	return nil
}
