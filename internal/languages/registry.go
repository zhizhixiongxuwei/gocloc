package languages

import (
	"io"
	"path/filepath"
	"sort"
	"strings"

	"gocloc/internal/model"
)

// Analyzer 定义单语言 FSM 分析器接口。
// 每种语言必须有独立实现文件，且独立维护自己的状态机状态。
type Analyzer interface {
	// Name 返回语言名称（例如 Go、JavaScript）。
	Name() string
	// Extensions 返回该语言支持的后缀列表（包含点号，如 .go）。
	Extensions() []string
	// Analyze 执行流式扫描并输出统计结果。
	Analyze(reader io.Reader) (model.LineMetrics, error)
}

// LanguageDescriptor 用于对外展示语言及后缀信息。
type LanguageDescriptor struct {
	Name       string
	Extensions []string
}

// Registry 管理语言分析器注册与后缀映射。
type Registry struct {
	analyzers     []Analyzer
	analyzerByExt map[string]Analyzer
}

// NewRegistry 创建并注册所有内置语言分析器。
func NewRegistry() *Registry {
	analyzers := []Analyzer{
		&GoAnalyzer{},
		&JavaScriptAnalyzer{},
		&TypeScriptAnalyzer{},
		&PythonAnalyzer{},
		&RustAnalyzer{},
		&RubyAnalyzer{},
		&JavaAnalyzer{},
		&CCPPAnalyzer{},
		&SQLAnalyzer{},
	}

	registry := &Registry{
		analyzers:     analyzers,
		analyzerByExt: make(map[string]Analyzer),
	}

	for _, analyzer := range analyzers {
		for _, ext := range analyzer.Extensions() {
			registry.analyzerByExt[strings.ToLower(ext)] = analyzer
		}
	}

	return registry
}

// AnalyzerForFile 根据文件后缀查找分析器。
func (r *Registry) AnalyzerForFile(path string) (Analyzer, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	analyzer, ok := r.analyzerByExt[ext]
	return analyzer, ok
}

// Languages 返回已注册语言清单。
func (r *Registry) Languages() []LanguageDescriptor {
	result := make([]LanguageDescriptor, 0, len(r.analyzers))
	for _, analyzer := range r.analyzers {
		extensions := append([]string(nil), analyzer.Extensions()...)
		sort.Strings(extensions)
		result = append(result, LanguageDescriptor{
			Name:       analyzer.Name(),
			Extensions: extensions,
		})
	}

	sort.Slice(result, func(i int, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// ExtensionsForLanguage 返回指定语言对应的全部后缀。
func (r *Registry) ExtensionsForLanguage(language string) []string {
	for _, analyzer := range r.analyzers {
		if analyzer.Name() == language {
			extensions := append([]string(nil), analyzer.Extensions()...)
			sort.Strings(extensions)
			return extensions
		}
	}
	return nil
}
