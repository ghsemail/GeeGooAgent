package strategydev

import "testing"

func TestSanitizeLabJEVReasonSummaryStripsThinkAndEnglish(t *testing.T) {
	raw := `<think>We need answer Chinese 1-2 sentences, no markdown/list/JSON, concise explain basis without changing/reinferring direction. Need cite probability/confidence. "System One 给出看空，置信度0.85，选项概率看空0.9，显著高于看多0.06和不明确0.04。信号分布高度集中于看空，因此主要依据是模型对空头方向的概率优势非常明显。" 2 sen...</think>`
	got := sanitizeLabJEVReasonSummary(raw)
	want := "JEV 判定为看空，置信度0.85，选项概率看空0.9，显著高于看多0.06和不明确0.04。信号分布高度集中于看空，因此主要依据是模型对空头方向的概率优势非常明显。"
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestSanitizeLabJEVReasonSummaryPlainChinese(t *testing.T) {
	raw := "多周期结构与锚点形态同向，短线偏多。"
	if got := sanitizeLabJEVReasonSummary(raw); got != raw {
		t.Fatalf("got=%q", got)
	}
}

func TestLabJEVReasonReplaceProductNames(t *testing.T) {
	got := labJEVReasonReplaceProductNames(`System One 给出「看多」`)
	if got != `JEV 判定为「看多」` {
		t.Fatalf("got=%q", got)
	}
}

func TestSanitizeLabJEVReasonSummaryVisibleAfterThink(t *testing.T) {
	raw := `<think>plan</think>最终可见的中文理由一句。`
	if got := sanitizeLabJEVReasonSummary(raw); got != "最终可见的中文理由一句。" {
		t.Fatalf("got=%q", got)
	}
}

func TestSanitizeLabJEVReasonSummaryTAForceIndexMixedLine(t *testing.T) {
	raw := `<think>We need 1-2 sentences without changing direction. JEV 决策看空，Force Index 为蓝，显示下跌动能尚未扭转，模型置信度 0.89.</think>`
	got := sanitizeLabJEVReasonSummary(raw)
	want := "JEV 决策看空，Force Index 为蓝，显示下跌动能尚未扭转，模型置信度 0.89."
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestSanitizeLabJEVReasonSummaryTARSIRejectedQuotedFragment(t *testing.T) {
	full := "JEV 决策看空，Stochastic（28.00）与 RSI（21.69）已超卖，模型仍以日线方向为主给出高置信度看空。"
	raw := `<think>We need 1-2 sentences. "8.00）与 RSI（21.69）已超卖，模型仍以日线方向为主给出高置信度看空。" ` + full + `</think>`
	got := sanitizeLabJEVReasonSummary(raw)
	if got != full {
		t.Fatalf("got=%q want=%q", got, full)
	}
}

func TestSanitizeLabJEVReasonSummaryTAMultiIndicatorUnicodeMinus(t *testing.T) {
	want := "JEV 决策看空，Force Index（−113544869.847）与 Stochastic（28.00）与 RSI（21.69）已超卖，模型仍以日线方向为主给出高置信度看空。"
	raw := `<think>We need 1-2 sentences without changing direction. ` + want + `</think>`
	if got := sanitizeLabJEVReasonSummary(raw); got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestSanitizeLabJEVReasonSummaryTASell328Style(t *testing.T) {
	want := "JEV 决策看空，Stochastic（60.00）与 RSI（45.05、49.58）均处中性区，尚无明确反转信号，模型置信度为 0.74。"
	frag := "60.00）与 RSI（45.05、49.58）均处中性区，尚无明确反转信号，模型置信度为 0.74。"
	raw := `<think>planning text. "` + frag + `" ` + want + `</think>`
	if got := sanitizeLabJEVReasonSummary(raw); got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}
