package languages

import (
	"bufio"
	"errors"
	"io"
	"unicode"

	"gocloc/internal/model"
)

// JavaScriptAnalyzer 是 JavaScript 专用 FSM 分析器。
type JavaScriptAnalyzer struct{}

// Name 返回语言名称。
func (a *JavaScriptAnalyzer) Name() string {
	return "JavaScript"
}

// Extensions 返回 JavaScript 常见后缀。
func (a *JavaScriptAnalyzer) Extensions() []string {
	return []string{".js", ".mjs", ".cjs"}
}

// Analyze 使用 JavaScript 独立状态机进行流式分析。
func (a *JavaScriptAnalyzer) Analyze(reader io.Reader) (model.LineMetrics, error) {
	engine := &javaScriptFSMEngine{}
	return engine.analyze(reader)
}

// javaScriptFSMEngine 持有 JavaScript 语法解析状态。
type javaScriptFSMEngine struct {
	inBlockComment    bool
	inSingleQuotedStr bool
	inDoubleQuotedStr bool
	inTemplateLiteral bool
}

// analyze 执行逐行读取与统计。
func (e *javaScriptFSMEngine) analyze(reader io.Reader) (model.LineMetrics, error) {
	var metrics model.LineMetrics

	// JavaScript 分析同样使用流式逐行读取：
	// 这样既能控制内存，又能保持“每行独立计数 + 状态跨行延续”的语义。
	bufferedReader := bufio.NewReader(reader)

	for {
		line, err := bufferedReader.ReadString('\n')
		// 没有任何剩余数据时，说明读取结束。
		if errors.Is(err, io.EOF) && len(line) == 0 {
			break
		}
		// 除 EOF 之外的读取错误直接返回。
		if err != nil && !errors.Is(err, io.EOF) {
			return metrics, err
		}

		// processLine 会根据当前 FSM 状态判断本行是否包含 code/comment。
		currentLine := normalizeLine(line)
		hasCode, hasComment := e.processLine(currentLine)
		applyLineClassification(&metrics, currentLine, hasCode, hasComment)

		// 最后一行可能没有 \n，处理后再退出。
		if errors.Is(err, io.EOF) {
			break
		}
	}

	return metrics, nil
}

// processLine 解析一行 JavaScript 代码。
func (e *javaScriptFSMEngine) processLine(line string) (bool, bool) {
	hasCode := false
	hasComment := false
	runes := []rune(line)

	// 继承跨行状态：块注释和字符串/模板字符串都可能延续到下一行。
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
			// JS 的 /* */ 注释不支持嵌套，这里只寻找当前层结束符。
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
			// 转义字符会消费下一个 rune，避免误判字符串结束。
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
			// 双引号字符串的转义逻辑与单引号一致。
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

		// 模板字符串中保留注释符号文本，不计为注释。
		if e.inTemplateLiteral {
			hasCode = true
			// 模板字符串允许换行，也保留 //、/* 等文本，不应计入注释。
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
			// 空白字符不会直接贡献分类，继续扫描后续字符。
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
