package languages

import (
	"bufio"
	"errors"
	"io"
	"unicode"

	"gocloc/internal/model"
)

// SQLAnalyzer 是 SQL 专用 FSM 分析器。
type SQLAnalyzer struct{}

// Name 返回语言名称。
func (a *SQLAnalyzer) Name() string {
	return "SQL"
}

// Extensions 返回 SQL 常见后缀。
func (a *SQLAnalyzer) Extensions() []string {
	return []string{".sql"}
}

// Analyze 使用 SQL 独立 FSM 进行分析。
func (a *SQLAnalyzer) Analyze(reader io.Reader) (model.LineMetrics, error) {
	engine := &sqlFSMEngine{}
	return engine.analyze(reader)
}

// sqlFSMEngine 维护 SQL 解析状态。
// 此实现支持 /* */ 嵌套块注释。
type sqlFSMEngine struct {
	blockCommentDepth int
	inSingleQuotedStr bool
	inDoubleQuotedStr bool
}

// analyze 逐行读取并累计统计值。
func (e *sqlFSMEngine) analyze(reader io.Reader) (model.LineMetrics, error) {
	var metrics model.LineMetrics

	// SQL 逐行流式读取，避免加载整文件。
	// 嵌套注释深度与字符串状态跨行保留，确保复杂 SQL 脚本统计准确。
	bufferedReader := bufio.NewReader(reader)

	for {
		line, err := bufferedReader.ReadString('\n')
		// 没有剩余内容时结束读取循环。
		if errors.Is(err, io.EOF) && len(line) == 0 {
			break
		}
		// 非 EOF 错误直接返回。
		if err != nil && !errors.Is(err, io.EOF) {
			return metrics, err
		}

		// processLine 返回本行的 code/comment 标志，再统一累加。
		currentLine := normalizeLine(line)
		hasCode, hasComment := e.processLine(currentLine)
		applyLineClassification(&metrics, currentLine, hasCode, hasComment)

		// 最后一行无 \n 的情况已处理，退出循环。
		if errors.Is(err, io.EOF) {
			break
		}
	}

	return metrics, nil
}

// processLine 分析单行 SQL 文本。
func (e *sqlFSMEngine) processLine(line string) (bool, bool) {
	hasCode := false
	hasComment := false
	runes := []rune(line)

	// SQL 的块注释支持嵌套，因此采用 depth 而非布尔状态。
	if e.blockCommentDepth > 0 {
		hasComment = true
	}
	if e.inSingleQuotedStr || e.inDoubleQuotedStr {
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

			// 嵌套注释开始，深度 +1。
			if current == '/' && hasNext && next == '*' {
				e.blockCommentDepth++
				idx += 2
				continue
			}
			// 注释结束，深度 -1，回到 0 表示离开注释态。
			if current == '*' && hasNext && next == '/' {
				e.blockCommentDepth--
				idx += 2
				continue
			}
			idx++
			continue
		}

		if e.inSingleQuotedStr {
			hasCode = true
			// SQL 单引号字符串使用 '' 作为转义，这里显式跳过。
			if current == '\'' {
				// SQL 单引号转义方式：''。
				if hasNext && next == '\'' {
					idx += 2
					continue
				}
				e.inSingleQuotedStr = false
			}
			idx++
			continue
		}

		if e.inDoubleQuotedStr {
			hasCode = true
			// SQL 双引号标识符/字符串中，"" 表示转义双引号。
			if current == '"' {
				// SQL 双引号转义方式：""。
				if hasNext && next == '"' {
					idx += 2
					continue
				}
				e.inDoubleQuotedStr = false
			}
			idx++
			continue
		}

		if unicode.IsSpace(current) {
			// 空白字符不直接决定分类。
			idx++
			continue
		}

		if current == '-' && hasNext && next == '-' {
			hasComment = true
			return hasCode, hasComment
		}

		if current == '/' && hasNext && next == '*' {
			hasComment = true
			// 首次进入块注释，深度初始化为 1。
			e.blockCommentDepth = 1
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

		hasCode = true
		idx++
	}

	return hasCode, hasComment
}
