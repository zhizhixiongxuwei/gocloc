package languages

import (
	"bufio"
	"errors"
	"io"
	"unicode"

	"gocloc/internal/model"
)

// TypeScriptAnalyzer 是 TypeScript 专用 FSM 分析器。
// 尽管语法与 JavaScript 相近，也保持独立文件与独立引擎实现。
type TypeScriptAnalyzer struct{}

// Name 返回语言名称。
func (a *TypeScriptAnalyzer) Name() string {
	return "TypeScript"
}

// Extensions 返回 TypeScript 文件后缀。
func (a *TypeScriptAnalyzer) Extensions() []string {
	return []string{".ts", ".tsx"}
}

// Analyze 逐行调用 TypeScript 独立状态机。
func (a *TypeScriptAnalyzer) Analyze(reader io.Reader) (model.LineMetrics, error) {
	engine := &typeScriptFSMEngine{}
	return engine.analyze(reader)
}

// typeScriptFSMEngine 维护 TypeScript 状态机状态。
type typeScriptFSMEngine struct {
	inBlockComment    bool
	inSingleQuotedStr bool
	inDoubleQuotedStr bool
	inTemplateLiteral bool
}

// analyze 执行流式读取与行级统计。
func (e *typeScriptFSMEngine) analyze(reader io.Reader) (model.LineMetrics, error) {
	var metrics model.LineMetrics

	// 逐行流式读取可以兼顾性能和准确性：
	// - 性能：不需要把文件整体读入内存；
	// - 准确性：行级计数天然贴合 total/code/comment/blank 的定义。
	bufferedReader := bufio.NewReader(reader)

	for {
		line, err := bufferedReader.ReadString('\n')
		// EOF 且无剩余文本时结束。
		if errors.Is(err, io.EOF) && len(line) == 0 {
			break
		}
		// 只要是非 EOF 的异常都应该中断。
		if err != nil && !errors.Is(err, io.EOF) {
			return metrics, err
		}

		// 单行解析由 processLine 负责，内部会处理状态迁移。
		currentLine := normalizeLine(line)
		hasCode, hasComment := e.processLine(currentLine)
		applyLineClassification(&metrics, currentLine, hasCode, hasComment)

		// 处理完最后一行（无换行符）后退出。
		if errors.Is(err, io.EOF) {
			break
		}
	}

	return metrics, nil
}

// processLine 解析一行 TypeScript 内容。
func (e *typeScriptFSMEngine) processLine(line string) (bool, bool) {
	hasCode := false
	hasComment := false
	runes := []rune(line)

	// 把上一行遗留状态带入本行，避免跨行字符串/注释统计丢失。
	if e.inBlockComment {
		hasComment = true
	}
	if e.inSingleQuotedStr || e.inDoubleQuotedStr || e.inTemplateLiteral {
		hasCode = true
	}

	for idx := 0; idx < len(runes); {
		current := runes[idx]
		hasNext := idx+1 < len(runes)
		next := rune(0)
		if hasNext {
			next = runes[idx+1]
		}

		if e.inBlockComment {
			hasComment = true
			// TS 的块注释也按非嵌套规则处理。
			if current == '*' && hasNext && next == '/' {
				e.inBlockComment = false
				idx += 2
				continue
			}
			idx++
			continue
		}

		if e.inSingleQuotedStr {
			hasCode = true
			// 在字符串态里，转义字符优先级高于结束引号。
			if current == '\\' && hasNext {
				idx += 2
				continue
			}
			if current == '\'' {
				e.inSingleQuotedStr = false
			}
			idx++
			continue
		}

		if e.inDoubleQuotedStr {
			hasCode = true
			// 双引号字符串用同样的转义策略。
			if current == '\\' && hasNext {
				idx += 2
				continue
			}
			if current == '"' {
				e.inDoubleQuotedStr = false
			}
			idx++
			continue
		}

		if e.inTemplateLiteral {
			hasCode = true
			// 模板字符串支持跨行，直到反引号闭合才退出该状态。
			if current == '\\' && hasNext {
				idx += 2
				continue
			}
			if current == '`' {
				e.inTemplateLiteral = false
			}
			idx++
			continue
		}

		if unicode.IsSpace(current) {
			// 空白字符不直接决定行分类。
			idx++
			continue
		}

		if current == '/' && hasNext && next == '/' {
			hasComment = true
			return hasCode, hasComment
		}

		if current == '/' && hasNext && next == '*' {
			hasComment = true
			e.inBlockComment = true
			idx += 2
			continue
		}

		if current == '\'' {
			hasCode = true
			e.inSingleQuotedStr = true
			idx++
			continue
		}

		if current == '"' {
			hasCode = true
			e.inDoubleQuotedStr = true
			idx++
			continue
		}

		if current == '`' {
			hasCode = true
			e.inTemplateLiteral = true
			idx++
			continue
		}

		hasCode = true
		idx++
	}

	return hasCode, hasComment
}
