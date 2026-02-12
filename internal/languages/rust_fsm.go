package languages

import (
	"bufio"
	"errors"
	"io"
	"unicode"

	"gocloc/internal/model"
)

// RustAnalyzer 是 Rust 语言专用 FSM 分析器。
type RustAnalyzer struct{}

// Name 返回语言名称。
func (a *RustAnalyzer) Name() string {
	return "Rust"
}

// Extensions 返回 Rust 后缀。
func (a *RustAnalyzer) Extensions() []string {
	return []string{".rs"}
}

// Analyze 使用 Rust 独立 FSM 流式读取并统计。
func (a *RustAnalyzer) Analyze(reader io.Reader) (model.LineMetrics, error) {
	engine := &rustFSMEngine{}
	return engine.analyze(reader)
}

// rustFSMEngine 记录 Rust 语法解析状态。
// Rust 的块注释支持嵌套，因此采用 depth 计数。
type rustFSMEngine struct {
	blockCommentDepth int
	inDoubleQuotedStr bool
	inSingleQuotedChr bool
	inRawString       bool
	rawStringHashCnt  int
}

// analyze 逐行执行解析，适配大文件流式处理。
func (e *rustFSMEngine) analyze(reader io.Reader) (model.LineMetrics, error) {
	var metrics model.LineMetrics

	// Rust 文件可能很大，采用逐行流式读取来控制内存占用。
	// 同时借助 engine 的成员字段保持跨行状态（嵌套注释、原始字符串等）。
	bufferedReader := bufio.NewReader(reader)

	for {
		line, err := bufferedReader.ReadString('\n')
		// 没有任何剩余字符时说明已经读完。
		if errors.Is(err, io.EOF) && len(line) == 0 {
			break
		}
		// 读取失败且不是 EOF 时，直接返回错误。
		if err != nil && !errors.Is(err, io.EOF) {
			return metrics, err
		}

		// 把当前行交给状态机，得到该行 code/comment 标记后再统一计数。
		currentLine := normalizeLine(line)
		hasCode, hasComment := e.processLine(currentLine)
		applyLineClassification(&metrics, currentLine, hasCode, hasComment)

		// EOF 且本行已被处理，退出主循环。
		if errors.Is(err, io.EOF) {
			break
		}
	}

	return metrics, nil
}

// processLine 分析一行 Rust 代码。
func (e *rustFSMEngine) processLine(line string) (bool, bool) {
	hasCode := false
	hasComment := false
	runes := []rune(line)

	// Rust 支持嵌套块注释，所以用 depth 计数器，而不是单一布尔值。
	// 只要 depth > 0，本行至少包含 comment。
	if e.blockCommentDepth > 0 {
		hasComment = true
	}
	if e.inDoubleQuotedStr || e.inSingleQuotedChr || e.inRawString {
		hasCode = true
	}

	for idx := 0; idx < len(runes); {
		current := runes[idx]
		hasNext := idx+1 < len(runes)
		next := rune(0)
		if hasNext {
			next = runes[idx+1]
		}

		if e.blockCommentDepth > 0 {
			hasComment = true

			// 在注释内部继续遇到 /* 时深度 +1，实现嵌套注释。
			if current == '/' && hasNext && next == '*' {
				e.blockCommentDepth++
				idx += 2
				continue
			}
			// 遇到 */ 时深度 -1，直到回到 0 才算完全离开注释态。
			if current == '*' && hasNext && next == '/' {
				e.blockCommentDepth--
				idx += 2
				continue
			}
			idx++
			continue
		}

		if e.inRawString {
			hasCode = true
			// 原始字符串结束符是 "####... 的组合，# 数量必须与开头一致。
			if current == '"' && e.matchRawStringTerminator(runes, idx) {
				e.inRawString = false
				idx += 1 + e.rawStringHashCnt
				continue
			}
			idx++
			continue
		}

		if e.inDoubleQuotedStr {
			hasCode = true
			// 标准字符串中反斜杠优先，避免把 \" 误判成闭合。
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

		if e.inSingleQuotedChr {
			hasCode = true
			// 字符字面量同样处理转义，如 '\n'、'\''。
			if current == '\\' && hasNext {
				idx += 2
				continue
			}
			if current == '\'' {
				e.inSingleQuotedChr = false
			}
			idx++
			continue
		}

		if unicode.IsSpace(current) {
			// 空白字符不参与分类，仅推进扫描。
			idx++
			continue
		}

		if current == '/' && hasNext && next == '/' {
			hasComment = true
			return hasCode, hasComment
		}

		if current == '/' && hasNext && next == '*' {
			hasComment = true
			// 新进入注释时深度从 1 开始。
			e.blockCommentDepth = 1
			idx += 2
			continue
		}

		// Rust 原始字符串格式：r"...", r#"..."#, br"..." 等。
		if consumed, started := e.tryStartRawString(runes, idx); started {
			hasCode = true
			idx = consumed
			continue
		}

		if current == '"' {
			hasCode = true
			e.inDoubleQuotedStr = true
			idx++
			continue
		}

		if current == '\'' && rustLooksLikeCharLiteral(runes, idx) {
			hasCode = true
			e.inSingleQuotedChr = true
			idx++
			continue
		}

		hasCode = true
		idx++
	}

	return hasCode, hasComment
}

// tryStartRawString 检测并进入 Rust 原始字符串状态。
// 返回值 consumed 是“已消费到的新索引位置”。
func (e *rustFSMEngine) tryStartRawString(runes []rune, idx int) (consumed int, started bool) {
	// 允许前缀是 r 或 br。
	start := idx
	if runes[idx] == 'b' {
		if idx+1 >= len(runes) || runes[idx+1] != 'r' {
			return idx + 1, false
		}
		start = idx + 1
	}

	if runes[start] != 'r' {
		return idx + 1, false
	}

	cursor := start + 1
	hashCount := 0
	for cursor < len(runes) && runes[cursor] == '#' {
		hashCount++
		cursor++
	}

	if cursor >= len(runes) || runes[cursor] != '"' {
		return idx + 1, false
	}

	e.inRawString = true
	e.rawStringHashCnt = hashCount
	return cursor + 1, true
}

// matchRawStringTerminator 判断当前位置是否命中原始字符串结束符。
func (e *rustFSMEngine) matchRawStringTerminator(runes []rune, idx int) bool {
	for i := 0; i < e.rawStringHashCnt; i++ {
		nextIndex := idx + 1 + i
		if nextIndex >= len(runes) || runes[nextIndex] != '#' {
			return false
		}
	}
	return true
}

// rustLooksLikeCharLiteral 用于区分字符字面量和生命周期标识（如 'a）。
func rustLooksLikeCharLiteral(runes []rune, idx int) bool {
	if idx+2 >= len(runes) {
		return false
	}

	// 普通字符：'a'
	if runes[idx+1] != '\\' && runes[idx+2] == '\'' {
		return true
	}

	// 转义字符：'\n'
	if runes[idx+1] == '\\' && idx+3 < len(runes) && runes[idx+3] == '\'' {
		return true
	}

	return false
}
