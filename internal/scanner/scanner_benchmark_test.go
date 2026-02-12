package scanner

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"gocloc/internal/languages"
)

// prepareBenchmarkFile 创建一个用于单文件扫描基准测试的 Go 文件。
func prepareBenchmarkFile(b *testing.B) string {
	b.Helper()

	tempDir := b.TempDir()
	filePath := filepath.Join(tempDir, "large.go")

	lines := make([]string, 0, 6000)
	lines = append(lines, "package main", "")
	for i := 0; i < 2000; i++ {
		lines = append(lines, "var value"+strconv.Itoa(i)+" = 1 // inline comment")
		lines = append(lines, "/* block comment */")
		lines = append(lines, "func f"+strconv.Itoa(i)+"() { _ = value"+strconv.Itoa(i)+" }")
	}

	if err := os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		b.Fatalf("write benchmark fixture failed: %v", err)
	}
	return filePath
}

// prepareBenchmarkDirectory 创建目录扫描基准测试数据。
func prepareBenchmarkDirectory(b *testing.B) string {
	b.Helper()

	tempDir := b.TempDir()
	for i := 0; i < 200; i++ {
		goFile := filepath.Join(tempDir, "pkg", "g"+strconv.Itoa(i)+".go")
		jsFile := filepath.Join(tempDir, "web", "j"+strconv.Itoa(i)+".js")

		if err := os.MkdirAll(filepath.Dir(goFile), 0o755); err != nil {
			b.Fatalf("mkdir go fixture dir failed: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(jsFile), 0o755); err != nil {
			b.Fatalf("mkdir js fixture dir failed: %v", err)
		}

		if err := os.WriteFile(goFile, []byte("package p\nvar x = 1 // c"), 0o644); err != nil {
			b.Fatalf("write go fixture failed: %v", err)
		}
		if err := os.WriteFile(jsFile, []byte("const x = 1; // c"), 0o644); err != nil {
			b.Fatalf("write js fixture failed: %v", err)
		}
	}
	return tempDir
}

// BenchmarkScanSingleFile 衡量单文件扫描性能。
func BenchmarkScanSingleFile(b *testing.B) {
	filePath := prepareBenchmarkFile(b)
	service := NewService(languages.NewRegistry(), 1)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := service.ScanPath(filePath); err != nil {
			b.Fatalf("scan failed: %v", err)
		}
	}
}

// BenchmarkScanDirectory 衡量目录并发扫描性能。
func BenchmarkScanDirectory(b *testing.B) {
	dirPath := prepareBenchmarkDirectory(b)
	service := NewService(languages.NewRegistry(), 8)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := service.ScanPath(dirPath); err != nil {
			b.Fatalf("scan failed: %v", err)
		}
	}
}
