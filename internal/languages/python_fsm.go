package languages

import (
	"bufio"
	"errors"
	"io"
	"unicode"

	"gocloc/internal/model"
)

// PythonAnalyzer 是 Python 语言专用 FSM 分析器。
type PythonAnalyzer struct{}

// Name 返回语言名称。
func (a *PythonAnalyzer) Name() string {
	return "Python"
}

// Extensions 返回 Python 后缀。
func (a *PythonAnalyzer) Extensions() []string {
	return []string{".py"}
}

// Analyze 使用 Python 独立 FSM 执行流式统计。
func (a *PythonAnalyzer) Analyze(reader io.Reader) (model.LineMetrics, error) {
	engine := &pythonFSMEngine{}
	return engine.analyze(reader)
}

// pythonFSMEngine 保存 Python 解析状态。
type pythonFSMEngine struct {
	inSingleQuotedStr bool
	inDoubleQuotedStr bool
	inTripleSingleStr bool
	inTripleDoubleStr bool
}

// analyze 流式读取并逐行统计。
func (e *pythonFSMEngine) analyze(reader io.Reader) (model.LineMetrics, error) {
	var metrics model.LineMetrics

	// Python 引擎按行读取并保持状态机跨行延续：
	// 三引号字符串经常跨行，必须在流式处理中持续保留状态。
	bufferedReader := bufio.NewReader(reader)

	for {
		line, err := bufferedReader.ReadString('\n')
		// 完整 EOF（无残余字符）直接结束。
		if errors.Is(err, io.EOF) && len(line) == 0 {
			break
		}
		// 读取过程中出现非 EOF 错误时，返回已知错误以便上层感知。
		if err != nil && !errors.Is(err, io.EOF) {
			return metrics, err
		}

		// 逐行归一化并交给 processLine 做 FSM 判定。
		currentLine := normalizeLine(line)
		hasCode, hasComment := e.processLine(currentLine)
		applyLineClassification(&metrics, currentLine, hasCode, hasComment)

		// EOF 但仍有本行内容时，需要在本轮统计后再退出。
		if errors.Is(err, io.EOF) {
			break
		}
	}

	return metrics, nil
}

// processLine 处理单行 Python 文本。
func (e *pythonFSMEngine) processLine(line string) (bool, bool) {
	hasCode := false
	hasComment := false
	runes := []rune(line)

	// 三引号或普通引号字符串如果跨行未闭合，当前行默认属于 code。
	if e.inSingleQuotedStr || e.inDoubleQuotedStr || e.inTripleSingleStr || e.inTripleDoubleStr {
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

		if e.inTripleSingleStr {
			hasCode = true
			// 三单引号字符串只有遇到 ''' 才会退出。
			if current == '\'' && hasNext && hasNextTwo && next == '\'' && nextTwo == '\'' {
				e.inTripleSingleStr = false
				idx += 3
				continue
			}
			idx++
			continue
		}

		if e.inTripleDoubleStr {
			hasCode = true
			// 三双引号字符串只有遇到 """ 才会退出。
			if current == '"' && hasNext && hasNextTwo && next == '"' && nextTwo == '"' {
				e.inTripleDoubleStr = false
				idx += 3
				continue
			}
			idx++
			continue
		}

		if e.inSingleQuotedStr {
			hasCode = true
			// 普通字符串里反斜杠会转义下一个字符。
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
			// 双引号字符串同样处理转义。
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

		if unicode.IsSpace(current) {
			// 空白字符继续跳过，等待第一个有效 token 决定分类。
			idx++
			continue
		}

		// Python 的行注释标识为 #，字符串内 # 由字符串状态吞掉。
		if current == '#' {
			hasComment = true
			return hasCode, hasComment
		}

		if current == '\'' {
			hasCode = true
			if hasNext && hasNextTwo && next == '\'' && nextTwo == '\'' {
				e.inTripleSingleStr = true
				idx += 3
				continue
			}
			e.inSingleQuotedStr = true
			idx++
			continue
		}

		if current == '"' {
			hasCode = true
			if hasNext && hasNextTwo && next == '"' && nextTwo == '"' {
				e.inTripleDoubleStr = true
				idx += 3
				continue
			}
			e.inDoubleQuotedStr = true
			idx++
			continue
		}

		hasCode = true
		idx++
	}

	return hasCode, hasComment
}
