package languages

import (
	"strings"

	"gocloc/internal/model"
)

// normalizeLine 用于去除每行末尾的换行符。
// 该函数适配 Windows 的 \r\n 与 Unix 的 \n。
func normalizeLine(line string) string {
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line
}

// applyLineClassification 根据 FSM 输出的分类结果更新统计值。
//
// 约束说明：
// - 每次调用都默认是“处理完一整行”，因此 Total 固定 +1
// - 同一行可以同时具备 code/comment，两者独立累计
// - 空白行判定要求：去掉空白字符后为空，且没有 code/comment 标记
func applyLineClassification(metrics *model.LineMetrics, line string, hasCode bool, hasComment bool) {
	metrics.Total++

	if strings.TrimSpace(line) == "" && !hasCode && !hasComment {
		metrics.Blank++
		return
	}

	if hasCode {
		metrics.Code++
	}

	if hasComment {
		metrics.Comment++
	}

	if !hasCode && !hasComment {
		metrics.Blank++
	}
}
