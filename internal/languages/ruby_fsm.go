package languages

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"unicode"

	"gocloc/internal/model"
)

// RubyAnalyzer 是 Ruby 专用 FSM 分析器。
type RubyAnalyzer struct{}

// Name 返回语言名称。
func (a *RubyAnalyzer) Name() string {
	return "Ruby"
}

// Extensions 返回 Ruby 后缀。
func (a *RubyAnalyzer) Extensions() []string {
	return []string{".rb"}
}

// Analyze 使用 Ruby 独立 FSM 执行扫描。
func (a *RubyAnalyzer) Analyze(reader io.Reader) (model.LineMetrics, error) {
	engine := &rubyFSMEngine{}
	return engine.analyze(reader)
}

// rubyFSMEngine 保存 Ruby 状态机状态。
// Ruby 支持 =begin / =end 块注释，这里用独立状态处理。
type rubyFSMEngine struct {
	inBeginEndComment bool
	inSingleQuotedStr bool
	inDoubleQuotedStr bool
}

// analyze 逐行流式读取并统计。
func (e *rubyFSMEngine) analyze(reader io.Reader) (model.LineMetrics, error) {
	var metrics model.LineMetrics

	// Ruby 同样按行流式处理：
	// - 保证大文件可控；
	// - 让 =begin/=end 与字符串状态能在行之间连续传播。
	bufferedReader := bufio.NewReader(reader)

	for {
		line, err := bufferedReader.ReadString('\n')
		// 完整读取结束时退出。
		if errors.Is(err, io.EOF) && len(line) == 0 {
			break
		}
		// 真正的读取错误要立即上抛。
		if err != nil && !errors.Is(err, io.EOF) {
			return metrics, err
		}

		// 把当前行交给 FSM 决策，然后统一写入统计模型。
		currentLine := normalizeLine(line)
		hasCode, hasComment := e.processLine(currentLine)
		applyLineClassification(&metrics, currentLine, hasCode, hasComment)

		// 最后一行可能没有 \n，处理后再跳出循环。
		if errors.Is(err, io.EOF) {
			break
		}
	}

	return metrics, nil
}

// processLine 处理单行 Ruby 内容。
func (e *rubyFSMEngine) processLine(line string) (bool, bool) {
	hasCode := false
	hasComment := false

	// begin/end 注释块优先级高于其他词法结构：
	// 只要处于该状态，整行都按 comment 处理，直到遇到 =end。
	// 若已处于 begin/end 注释块中，整行视为注释，直到遇到 =end。
	if e.inBeginEndComment {
		hasComment = true
		if isRubyBeginEndDirective(line, "=end") {
			e.inBeginEndComment = false
		}
		return false, hasComment
	}

	// 进入 begin/end 注释块，当前行本身也计为注释。
	if isRubyBeginEndDirective(line, "=begin") {
		e.inBeginEndComment = true
		return false, true
	}

	runes := []rune(line)
	if e.inSingleQuotedStr || e.inDoubleQuotedStr {
		hasCode = true
	}

	for idx := 0; idx < len(runes); {
		current := runes[idx]
		hasNext := idx+1 < len(runes)

		if e.inSingleQuotedStr {
			hasCode = true
			// Ruby 字符串支持反斜杠转义，需要先跳过被转义字符。
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
			// 双引号字符串的转义处理与单引号一致。
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
			// 空白字符不做分类决策，继续扫描后续 token。
			idx++
			continue
		}

		// Ruby 行注释标识：#
		if current == '#' {
			hasComment = true
			return hasCode, hasComment
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

		hasCode = true
		idx++
	}

	return hasCode, hasComment
}

// isRubyBeginEndDirective 判断当前行是否是 =begin 或 =end 指令。
// 实际 Ruby 规范要求它们位于行首，这里允许前导空白，兼容更多代码风格。
func isRubyBeginEndDirective(line string, directive string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, directive) {
		return false
	}
	if len(trimmed) == len(directive) {
		return true
	}
	return unicode.IsSpace(rune(trimmed[len(directive)]))
}
