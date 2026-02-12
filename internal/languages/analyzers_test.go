package languages

import (
	"strings"
	"testing"

	"gocloc/internal/model"
)

// analyzeText 是测试辅助函数，用于快速运行某个分析器并返回统计结果。
func analyzeText(t *testing.T, analyzer Analyzer, content string) model.LineMetrics {
	t.Helper()

	metrics, err := analyzer.Analyze(strings.NewReader(content))
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	return metrics
}

// TestGoInlineCodeAndComment 验证同一行 code + comment 的计数能力。
func TestGoInlineCodeAndComment(t *testing.T) {
	analyzer := &GoAnalyzer{}
	content := "package main\n" +
		"func main() {\n" +
		"    x := 1 // comment\n" +
		"}\n"

	metrics := analyzeText(t, analyzer, content)

	if metrics.Total != 4 || metrics.Code != 4 || metrics.Comment != 1 || metrics.Blank != 0 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

// TestGoStringContainsCommentToken 验证字符串内的 // 不会误判为注释。
func TestGoStringContainsCommentToken(t *testing.T) {
	analyzer := &GoAnalyzer{}
	content := "package main\n" +
		"func main() {\n" +
		"    s := \"hello // world\"\n" +
		"}\n"

	metrics := analyzeText(t, analyzer, content)

	if metrics.Total != 4 || metrics.Code != 4 || metrics.Comment != 0 || metrics.Blank != 0 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

// TestRustNestedBlockComment 验证 Rust 嵌套块注释。
func TestRustNestedBlockComment(t *testing.T) {
	analyzer := &RustAnalyzer{}
	content := "fn main() {\n" +
		"    let x = 1; /* outer /* inner */ tail */\n" +
		"}\n"

	metrics := analyzeText(t, analyzer, content)

	if metrics.Total != 3 || metrics.Code != 3 || metrics.Comment != 1 || metrics.Blank != 0 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

// TestRubyBeginEndComment 验证 Ruby 的 =begin/=end 块注释。
func TestRubyBeginEndComment(t *testing.T) {
	analyzer := &RubyAnalyzer{}
	content := "=begin\n" +
		"comment body\n" +
		"=end\n" +
		"puts \"ok\"\n"

	metrics := analyzeText(t, analyzer, content)

	if metrics.Total != 4 || metrics.Code != 1 || metrics.Comment != 3 || metrics.Blank != 0 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

// TestPythonStringAndComment 验证 Python 字符串中 # 与真实注释的区分。
func TestPythonStringAndComment(t *testing.T) {
	analyzer := &PythonAnalyzer{}
	content := "value = \"hello # world\"\n" +
		"# real comment\n"

	metrics := analyzeText(t, analyzer, content)

	if metrics.Total != 2 || metrics.Code != 1 || metrics.Comment != 1 || metrics.Blank != 0 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

// TestSQLNestedBlockComment 验证 SQL 嵌套块注释和行注释。
func TestSQLNestedBlockComment(t *testing.T) {
	analyzer := &SQLAnalyzer{}
	content := "SELECT 1; /* outer /* inner */ outer */\n" +
		"-- line comment\n"

	metrics := analyzeText(t, analyzer, content)

	if metrics.Total != 2 || metrics.Code != 1 || metrics.Comment != 2 || metrics.Blank != 0 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

// TestRegistryLanguages 确认注册中心包含用户要求的 9 种语言。
func TestRegistryLanguages(t *testing.T) {
	registry := NewRegistry()
	languages := registry.Languages()

	if len(languages) != 9 {
		t.Fatalf("unexpected language count: %d", len(languages))
	}

	requiredExtensions := []string{".go", ".js", ".ts", ".py", ".rs", ".rb", ".java", ".cpp", ".sql"}
	for _, extension := range requiredExtensions {
		if _, ok := registry.AnalyzerForFile("x" + extension); !ok {
			t.Fatalf("missing analyzer for extension %s", extension)
		}
	}
}
