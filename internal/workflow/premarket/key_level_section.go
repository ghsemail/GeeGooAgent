package premarket

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

func keyLevelEngineSection(ws memory.StockWorkspace, levels stockfmt.KeyLevels) string {
	if !levels.Provenance.EngineUsed && !ws.KeyLevelsEngineOK {
		return "暂无 Key Level Engine 快照（可依赖周线文本中的支撑/阻力描述）。"
	}
	lines := make([]string, 0, 6)
	if levels.Provenance.SupportZone != "" {
		line := fmt.Sprintf("- **支撑带** %s", levels.Provenance.SupportZone)
		if ws.KeyLevelSupportState != "" {
			line += fmt.Sprintf("（状态：%s）", localizeZoneState(ws.KeyLevelSupportState))
		}
		lines = append(lines, line)
	} else if levels.Support != nil {
		lines = append(lines, fmt.Sprintf("- **支撑** 约 %.2f", *levels.Support))
	}
	if levels.Provenance.ResistZone != "" {
		line := fmt.Sprintf("- **阻力带** %s", levels.Provenance.ResistZone)
		if ws.KeyLevelResistState != "" {
			line += fmt.Sprintf("（状态：%s）", localizeZoneState(ws.KeyLevelResistState))
		}
		lines = append(lines, line)
	} else if levels.Resistance != nil {
		lines = append(lines, fmt.Sprintf("- **阻力** 约 %.2f", *levels.Resistance))
	}
	if len(levels.Provenance.SupportSrc) > 0 {
		lines = append(lines, "- 支撑证据："+strings.Join(levels.Provenance.SupportSrc, "、"))
	}
	if len(levels.Provenance.ResistSrc) > 0 {
		lines = append(lines, "- 阻力证据："+strings.Join(levels.Provenance.ResistSrc, "、"))
	}
	if len(lines) == 0 {
		return "引擎未返回有效价格带。"
	}
	return strings.Join(lines, "\n")
}

func localizeZoneState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "untested":
		return "未测试"
	case "tested":
		return "已测试"
	case "rejected":
		return "拒绝"
	case "broken":
		return "已突破"
	case "retested":
		return "突破后回踩"
	case "invalidated":
		return "已失效"
	case "possible_breakout":
		return "疑似突破"
	case "confirmed_breakout":
		return "突破确认"
	case "both":
		return "支撑/阻力翻转"
	case "neutral":
		return "中性"
	default:
		return state
	}
}
