// Package model 定义 gocloc 的核心数据模型。
// 这些结构会被扫描器、输出层和命令层共同使用。
package model

// LineMetrics 表示一组行级统计值。
//
// 注意：
// - Total 表示总行数（每行计 1）
// - Code/Comment 可以在同一行同时 +1（例如: x := 1 // note）
// - Blank 仅用于既不是代码也不是注释的空白行
type LineMetrics struct {
	Total   int64 `json:"total"`
	Code    int64 `json:"code"`
	Comment int64 `json:"comment"`
	Blank   int64 `json:"blank"`
}

// Add 将另一个统计结果叠加到当前对象。
func (m *LineMetrics) Add(other LineMetrics) {
	m.Total += other.Total
	m.Code += other.Code
	m.Comment += other.Comment
	m.Blank += other.Blank
}

// FileMetrics 表示单文件扫描结果。
type FileMetrics struct {
	Path     string      `json:"path"`
	Language string      `json:"language"`
	Metrics  LineMetrics `json:"metrics"`
}

// LanguageMetrics 表示某个语言的聚合结果。
type LanguageMetrics struct {
	Language   string      `json:"language"`
	Extensions []string    `json:"extensions"`
	Files      int64       `json:"files"`
	Metrics    LineMetrics `json:"metrics"`
}

// ScanError 记录单文件扫描失败信息。
// 设计为“错误不阻断全量扫描”，便于大仓库分析时容错。
type ScanError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// TotalMetrics 表示项目级总计信息。
// 在 LineMetrics 基础上额外增加 Files 字段，
// 用于表达“本次扫描统计到了多少个有效源码文件”。
type TotalMetrics struct {
	Files int64 `json:"files"`
	LineMetrics
}

// AddFileMetrics 累加一个文件的统计值到项目总计中。
func (m *TotalMetrics) AddFileMetrics(other LineMetrics) {
	m.Files++
	m.LineMetrics.Add(other)
}

// ScanResult 是 scan 命令的完整输出模型。
// 包含文件级明细、语言级汇总、全局总计和错误列表。
type ScanResult struct {
	ScannedPath string            `json:"scanned_path"`
	Files       []FileMetrics     `json:"files"`
	Languages   []LanguageMetrics `json:"languages"`
	Total       TotalMetrics      `json:"total"`
	Errors      []ScanError       `json:"errors"`
}
