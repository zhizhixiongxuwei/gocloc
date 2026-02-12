package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gocloc/internal/languages"
)

// writeFixtureFile 是测试辅助函数，用于在临时目录快速落地测试文件。
func writeFixtureFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture file failed: %v", err)
	}
}

// TestScanSingleFile 验证 scan 支持“直接传单文件路径”。
func TestScanSingleFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "single.go")

	writeFixtureFile(t, filePath, strings.Join([]string{
		"package main",
		"// top comment",
		"func main() { x := 1 // inline }",
	}, "\n"))

	service := NewService(languages.NewRegistry(), 2)
	result, err := service.ScanPath(filePath)
	if err != nil {
		t.Fatalf("scan single file failed: %v", err)
	}

	if len(result.Files) != 1 {
		t.Fatalf("expected 1 scanned file, got %d", len(result.Files))
	}
	if result.Total.Files != 1 {
		t.Fatalf("expected total.files=1, got %d", result.Total.Files)
	}
	if result.Total.Total != 3 || result.Total.Code != 2 || result.Total.Comment != 2 || result.Total.Blank != 0 {
		t.Fatalf("unexpected total metrics: %+v", result.Total)
	}

	fileMetrics := result.Files[0]
	if fileMetrics.Path != "single.go" {
		t.Fatalf("expected display path single.go, got %s", fileMetrics.Path)
	}
	if fileMetrics.Language != "Go" {
		t.Fatalf("expected language Go, got %s", fileMetrics.Language)
	}
}

// TestScanDirectoryTotalFiles 验证目录扫描时 total.files 与文件数一致。
func TestScanDirectoryTotalFiles(t *testing.T) {
	tempDir := t.TempDir()

	writeFixtureFile(t, filepath.Join(tempDir, "main.go"), strings.Join([]string{
		"package main",
		"func main() {}",
	}, "\n"))
	writeFixtureFile(t, filepath.Join(tempDir, "web", "app.js"), strings.Join([]string{
		"const x = 1; // js comment",
	}, "\n"))
	writeFixtureFile(t, filepath.Join(tempDir, "README.txt"), "not a source file")

	service := NewService(languages.NewRegistry(), 4)
	result, err := service.ScanPath(tempDir)
	if err != nil {
		t.Fatalf("scan directory failed: %v", err)
	}

	if len(result.Files) != 2 {
		t.Fatalf("expected 2 scanned files, got %d", len(result.Files))
	}
	if result.Total.Files != 2 {
		t.Fatalf("expected total.files=2, got %d", result.Total.Files)
	}
	if len(result.Languages) != 2 {
		t.Fatalf("expected 2 language summaries, got %d", len(result.Languages))
	}
}

// TestScanUnsupportedSingleFile 验证单文件模式下不支持后缀会返回错误。
func TestScanUnsupportedSingleFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "demo.txt")
	writeFixtureFile(t, filePath, "plain text")

	service := NewService(languages.NewRegistry(), 1)
	_, err := service.ScanPath(filePath)
	if err == nil {
		t.Fatalf("expected unsupported extension error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported file extension") {
		t.Fatalf("unexpected error: %v", err)
	}
}
