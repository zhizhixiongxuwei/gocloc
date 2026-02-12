package languages

import (
	"bufio"
	"errors"
	"io"
	"unicode"

	"gocloc/internal/model"
)

// CCPPAnalyzer 是 C/C++ 专用 FSM 分析器。
type CCPPAnalyzer struct{}

// Name 返回语言名称。
func (a *CCPPAnalyzer) Name() string {
	return "C/C++"
}

// Extensions 返回 C/C++ 典型后缀集合。
func (a *CCPPAnalyzer) Extensions() []string {
	return []string{".c", ".cc", ".cpp", ".cxx", ".h", ".hh", ".hpp", ".hxx"}
}

// Analyze 使用 C/C++ 独立 FSM 对内容进行流式扫描。
func (a *CCPPAnalyzer) Analyze(reader io.Reader) (model.LineMetrics, error) {
	engine := &cCppFSMEngine{}
	return engine.analyze(reader)
}

// cCppFSMEngine 维护 C/C++ 注释和字符串状态。
type cCppFSMEngine struct {
	inBlockComment bool
	inDoubleQuoted bool
	inSingleQuoted bool
}

// analyze 逐行处理输入流。
func (e *cCppFSMEngine) analyze(reader io.Reader) (model.LineMetrics, error) {
	var metrics model.LineMetrics

	// C/C++ 使用按行流式读取，避免大文件造成内存压力。
	// 块注释和字符串状态由 engine 持久化，保证跨行解析正确。
	bufferedReader := bufio.NewReader(reader)

	for {
		line, err := bufferedReader.ReadString('\n')
		// 没有残留字符的 EOF 说明读取完成。
		if errors.Is(err, io.EOF) && len(line) == 0 {
			break
		}
		// 真正读取错误直接返回。
		if err != nil && !errors.Is(err, io.EOF) {
			return metrics, err
		}

		// 把当前行交给 processLine，根据 FSM 状态做精确分类。
		currentLine := normalizeLine(line)
		hasCode, hasComment := e.processLine(currentLine)
		applyLineClassification(&metrics, currentLine, hasCode, hasComment)

		// 最后一行即使没有换行，也已完成统计。
		if errors.Is(err, io.EOF) {
			break
		}
	}

	return metrics, nil
}

// processLine 解析单行 C/C++ 内容。
func (e *cCppFSMEngine) processLine(line string) (bool, bool) {
	hasCode := false
	hasComment := false
	runes := []rune(line)

	// 初始化当前行分类标记，先继承跨行状态。
	if e.inBlockComment {
		hasComment = true
	}
	if e.inDoubleQuoted || e.inSingleQuoted {
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
			// C/C++ 块注释非嵌套，遇到 */ 即离开。
			if current == '*' && hasNext && next == '/' {
				e.inBlockComment = false
				idx += 2
				continue
			}
			idx++
			continue
		}

		if e.inDoubleQuoted {
			hasCode = true
			// 字符串里的转义字符优先消费，避免误识别结束引号。
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
			// 字符字面量同样需要跳过转义。
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
			// 空白字符不参与注释/代码判断。
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
