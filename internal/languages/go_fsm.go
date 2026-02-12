package languages

import (
	"bufio"
	"errors"
	"io"
	"unicode"

	"gocloc/internal/model"
)

// GoAnalyzer 是 Go 语言专用分析器。
// 该实现只处理 Go 语法相关状态，不与其他语言复用 FSM 类型。
type GoAnalyzer struct{}

// Name 返回语言名称。
func (a *GoAnalyzer) Name() string {
	return "Go"
}

// Extensions 返回 Go 文件后缀。
func (a *GoAnalyzer) Extensions() []string {
	return []string{".go"}
}

// Analyze 使用 Go 专用 FSM 对输入流逐行扫描。
func (a *GoAnalyzer) Analyze(reader io.Reader) (model.LineMetrics, error) {
	engine := &goFSMEngine{}
	return engine.analyze(reader)
}

// goFSMEngine 维护 Go 语言分析时的状态集合。
type goFSMEngine struct {
	inBlockComment     bool
	inDoubleQuotedStr  bool
	inSingleQuotedRune bool
	inRawStringLiteral bool
}

// analyze 采用流式读取逐行解析，避免一次性加载大文件。
func (e *goFSMEngine) analyze(reader io.Reader) (model.LineMetrics, error) {
	var metrics model.LineMetrics

	// 这里使用 ReadString('\n') 做“按行流式”读取：
	// 1) 不会把整个文件一次性载入内存；
	// 2) 便于和行级统计模型（code/comment/blank）天然对齐。
	bufferedReader := bufio.NewReader(reader)
	for {
		line, err := bufferedReader.ReadString('\n')
		// EOF 且没有任何剩余字符时，说明已经没有可处理行，直接退出。
		if errors.Is(err, io.EOF) && len(line) == 0 {
			break
		}
		// 非 EOF 错误需要立即返回，避免输出不完整统计结果。
		if err != nil && !errors.Is(err, io.EOF) {
			return metrics, err
		}

		// 逐行交给 processLine，让状态机在“当前行+历史状态”基础上判断。
		currentLine := normalizeLine(line)
		hasCode, hasComment := e.processLine(currentLine)
		applyLineClassification(&metrics, currentLine, hasCode, hasComment)

		// EOF 但 line 非空代表“最后一行没有换行符”，这行已经处理完，随后退出。
		if errors.Is(err, io.EOF) {
			break
		}
	}

	return metrics, nil
}

// processLine 扫描单行并更新 FSM 状态，返回该行是否包含 code/comment。
func (e *goFSMEngine) processLine(line string) (bool, bool) {
	hasCode := false
	hasComment := false
	runes := []rune(line)

	// 先根据“跨行状态”做初始赋值：
	// - 如果上一个行尾还处于块注释中，本行天然包含 comment；
	// - 如果上一个行尾还在字符串/字符字面量中，本行天然包含 code。
	if e.inBlockComment {
		hasComment = true
	}
	if e.inDoubleQuotedStr || e.inSingleQuotedRune || e.inRawStringLiteral {
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
			// Go 块注释不支持嵌套，所以只识别最近的 */ 结束当前注释状态。
			if current == '*' && hasNext && next == '/' {
				e.inBlockComment = false
				idx += 2
				continue
			}
			idx++
			continue
		}

		if e.inRawStringLiteral {
			hasCode = true
			// 原始字符串仅由反引号闭合，不处理转义。
			if current == '`' {
				e.inRawStringLiteral = false
			}
			idx++
			continue
		}

		if e.inDoubleQuotedStr {
			hasCode = true
			// 普通字符串里反斜杠会吞掉下一个字符，避免误把 \" 当结束引号。
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

		if e.inSingleQuotedRune {
			hasCode = true
			// 字符字面量同样处理转义，避免 '\'' 等场景误判。
			if current == '\\' && hasNext {
				idx += 2
				continue
			}
			if current == '\'' {
				e.inSingleQuotedRune = false
			}
			idx++
			continue
		}

		if unicode.IsSpace(current) {
			// 空白字符不决定 code/comment，仅推进游标。
			idx++
			continue
		}

		// 行注释：遇到 // 后剩余部分都属于注释。
		if current == '/' && hasNext && next == '/' {
			hasComment = true
			return hasCode, hasComment
		}

		// 块注释：Go 不支持嵌套，进入后直到 */。
		if current == '/' && hasNext && next == '*' {
			hasComment = true
			e.inBlockComment = true
			idx += 2
			continue
		}

		if current == '"' {
			hasCode = true
			e.inDoubleQuotedStr = true
			idx++
			continue
		}

		if current == '\'' {
			hasCode = true
			e.inSingleQuotedRune = true
			idx++
			continue
		}

		if current == '`' {
			hasCode = true
			e.inRawStringLiteral = true
			idx++
			continue
		}

		hasCode = true
		idx++
	}

	return hasCode, hasComment
}
