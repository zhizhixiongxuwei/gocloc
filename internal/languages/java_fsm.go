package languages

import (
	"bufio"
	"errors"
	"io"
	"unicode"

	"gocloc/internal/model"
)

// JavaAnalyzer 是 Java 专用 FSM 分析器。
type JavaAnalyzer struct{}

// Name 返回语言名称。
func (a *JavaAnalyzer) Name() string {
	return "Java"
}

// Extensions 返回 Java 后缀。
func (a *JavaAnalyzer) Extensions() []string {
	return []string{".java"}
}

// Analyze 调用 Java 独立 FSM。
func (a *JavaAnalyzer) Analyze(reader io.Reader) (model.LineMetrics, error) {
	engine := &javaFSMEngine{}
	return engine.analyze(reader)
}

// javaFSMEngine 维护 Java 词法级状态。
// 包含注释、普通字符串、字符字面量、文本块（"""）等状态。
type javaFSMEngine struct {
	inBlockComment bool
	inDoubleQuoted bool
	inSingleQuoted bool
	inTextBlockStr bool
}

// analyze 逐行读取并统计。
func (e *javaFSMEngine) analyze(reader io.Reader) (model.LineMetrics, error) {
	var metrics model.LineMetrics

	// Java 文件按行流式读取，避免一次性占用大内存。
	// 文本块字符串（"""）和块注释状态通过 engine 字段跨行延续。
	bufferedReader := bufio.NewReader(reader)

	for {
		line, err := bufferedReader.ReadString('\n')
		// EOF 且无文本时表示读取结束。
		if errors.Is(err, io.EOF) && len(line) == 0 {
			break
		}
		// 任何非 EOF 读取异常都应立即失败。
		if err != nil && !errors.Is(err, io.EOF) {
			return metrics, err
		}

		// 逐行交给 FSM 判定当前行的 code/comment 属性。
		currentLine := normalizeLine(line)
		hasCode, hasComment := e.processLine(currentLine)
		applyLineClassification(&metrics, currentLine, hasCode, hasComment)

		// 最后一行已处理后退出。
		if errors.Is(err, io.EOF) {
			break
		}
	}

	return metrics, nil
}

// processLine 处理一行 Java 文本。
func (e *javaFSMEngine) processLine(line string) (bool, bool) {
	hasCode := false
	hasComment := false
	runes := []rune(line)

	// 先注入跨行状态，确保多行注释/字符串不会漏算。
	if e.inBlockComment {
		hasComment = true
	}
	if e.inDoubleQuoted || e.inSingleQuoted || e.inTextBlockStr {
		hasCode = true
	}

	for idx := 0; idx < len(runes); {
		current := runes[idx]
		hasNext := idx+1 < len(runes)
		hasNextTwo := idx+2 < len(runes)
		next := rune(0)
		nextTwo := rune(0)
		if hasNext {
			next = runes[idx+1]
		}
		if hasNextTwo {
			nextTwo = runes[idx+2]
		}

		if e.inBlockComment {
			hasComment = true
			// Java 的 /* */ 注释不支持嵌套，找到 */ 即可离开。
			if current == '*' && hasNext && next == '/' {
				e.inBlockComment = false
				idx += 2
				continue
			}
			idx++
			continue
		}

		if e.inTextBlockStr {
			hasCode = true
			// 文本块字符串以 """ 闭合，内部可跨行包含注释符号文本。
			if current == '"' && hasNext && hasNextTwo && next == '"' && nextTwo == '"' {
				e.inTextBlockStr = false
				idx += 3
				continue
			}
			idx++
			continue
		}

		if e.inDoubleQuoted {
			hasCode = true
			// 处理转义字符，避免 \" 导致提早退出字符串态。
			if current == '\\' && hasNext {
				idx += 2
				continue
			}
			if current == '"' {
				e.inDoubleQuoted = false
			}
			idx++
			continue
		}

		if e.inSingleQuoted {
			hasCode = true
			// 字符字面量同样要处理转义。
			if current == '\\' && hasNext {
				idx += 2
				continue
			}
			if current == '\'' {
				e.inSingleQuoted = false
			}
			idx++
			continue
		}

		if unicode.IsSpace(current) {
			// 空白字符不改变分类结果。
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

		if current == '"' && hasNext && hasNextTwo && next == '"' && nextTwo == '"' {
			hasCode = true
			e.inTextBlockStr = true
			idx += 3
			continue
		}

		if current == '"' {
			hasCode = true
			e.inDoubleQuoted = true
			idx++
			continue
		}

		if current == '\'' {
			hasCode = true
			e.inSingleQuoted = true
			idx++
			continue
		}

		hasCode = true
		idx++
	}

	return hasCode, hasComment
}
