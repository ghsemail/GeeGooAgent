package strategydev

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// MacroPoolSymbolInput is one frozen symbol tier row for pool JEV synthesis.
type MacroPoolSymbolInput struct {
	Symbol       string
	ResultID     string
	JEVDirection string // long | short | neutral (ledger)
	JEVReason    string
	Confidence   float64
}

// MacroPoolJEV is the pool-level JEV outcome (assemble-only; no HTTP PUT).
type MacroPoolJEV struct {
	OK                bool
	Model             string
	RecommendedChoice string
	Direction         LabJEVUnifiedDirection
	Confidence        float64
	Conflict          bool
	Choices           []PriceAnalysisJEVChoice
	RecommendedReason string
	Error             string
}

func ledgerDirectionToChoice(dir string) string {
	switch strings.TrimSpace(strings.ToLower(dir)) {
	case "long":
		return "看多"
	case "short":
		return "看空"
	default:
		return "不明确"
	}
}

func choiceToLedgerDirection(choice string) LabJEVUnifiedDirection {
	switch strings.TrimSpace(choice) {
	case "看多":
		return LabJEVUnifiedBull
	case "看空":
		return LabJEVUnifiedBear
	default:
		return LabJEVUnifiedUnclear
	}
}

func ledgerDirectionToUnified(dir string) LabJEVUnifiedDirection {
	return choiceToLedgerDirection(ledgerDirectionToChoice(dir))
}

const maxPoolSymbolReasonInState = 480

// BuildMacroPoolJEVState builds Decision Choice state from symbol summaries.
func BuildMacroPoolJEVState(title string, symbols []MacroPoolSymbolInput) string {
	var b strings.Builder
	if t := strings.TrimSpace(title); t != "" {
		fmt.Fprintf(&b, "监控池：%s\n", t)
	}
	for _, sym := range symbols {
		code := strings.TrimSpace(sym.Symbol)
		if code == "" {
			continue
		}
		choice := ledgerDirectionToChoice(sym.JEVDirection)
		reason := strings.TrimSpace(sym.JEVReason)
		if len(reason) > maxPoolSymbolReasonInState {
			reason = reason[:maxPoolSymbolReasonInState] + "…"
		}
		fmt.Fprintf(&b, "\n【标的 %s】\n", code)
		fmt.Fprintf(&b, "标的 JEV 方向：%s\n", choice)
		if sym.Confidence > 0 {
			fmt.Fprintf(&b, "置信：%.3f\n", clampUnitInterval(sym.Confidence))
		}
		if reason != "" {
			fmt.Fprintf(&b, "依据摘要：%s\n", reason)
		}
	}
	return strings.TrimSpace(b.String())
}

// macroPoolSymbolDirectionConflict is true when symbol tier has both long and short (ledger) inputs.
// Neutral-only disagreement does not set conflict; pool Choice may still be 不明确.
func macroPoolSymbolDirectionConflict(symbols []MacroPoolSymbolInput) bool {
	hasLong, hasShort := false, false
	for _, sym := range symbols {
		switch strings.TrimSpace(strings.ToLower(sym.JEVDirection)) {
		case "long":
			hasLong = true
		case "short":
			hasShort = true
		}
	}
	return hasLong && hasShort
}

func macroPoolReasonExcerpt(reason string) string {
	r := strings.TrimSpace(reason)
	if len(r) <= 240 {
		return r
	}
	return r[:239] + "…"
}

// ResolveMacroPoolJEV runs GeeGooData Choice on symbol summaries, then Chinese pool reason.
func ResolveMacroPoolJEV(
	ctx context.Context,
	jev *JevClient,
	poolWorkflow, title string,
	symbols []MacroPoolSymbolInput,
) MacroPoolJEV {
	model := config.DefaultDecisionModel
	if jev != nil && strings.TrimSpace(jev.cfg.Model) != "" {
		model = strings.TrimSpace(jev.cfg.Model)
	}
	base := MacroPoolJEV{Model: model, Choices: priceAnalysisJEVChoicesFromProbs(nil)}
	state := BuildMacroPoolJEVState(title, symbols)
	if state == "" {
		base.Error = "pool decision state empty"
		return base
	}
	if jev == nil || strings.TrimSpace(jev.cfg.APIKey) == "" {
		base.Error = "jev not configured"
		return base
	}
	instructions := "基于 state 中各标的 JEV 方向、置信与各条「依据摘要」全文，综合判断该监控池当日整体方向。" +
		"禁止对标的简单计数、固定权重或机械多数票；须阅读摘要中的事实与理由后权衡。" +
		"只能从给定选项中选一项。规则：①摘要与方向一致且互相印证时可倾向该方向；②多空摘要冲突时须权衡后选看多或看空；" +
		"③仅部分标的有明确倾向时跟随有信息的一方；④均不足时才选不明确。"
	choice, probs, confidence, respModel, err := jev.Choice(ctx, state, "direction", instructions, labJEVUnifiedCriteria)
	if err != nil {
		base.Error = err.Error()
		return base
	}
	if respModel != "" {
		base.Model = respModel
	}
	idx, ok := indexPriceAnalysisJEVLabel(choice)
	if !ok {
		base.Error = fmt.Sprintf("unexpected jev choice: %q", choice)
		return base
	}
	base.Choices = priceAnalysisJEVChoicesFromProbs(probs)
	base.RecommendedChoice = choice
	conf := 0.0
	if idx >= 0 && idx < len(priceAnalysisJEVLabels) {
		if p, ok := probs[priceAnalysisJEVLabels[idx]]; ok {
			conf = clampUnitInterval(p)
		}
	}
	if confidence > 0 {
		conf = clampUnitInterval(confidence)
	}
	conflict := macroPoolSymbolDirectionConflict(symbols)
	if conflict && conf > labJEVUnifiedConflictConfidenceCap {
		conf = labJEVUnifiedConflictConfidenceCap
	}
	var reasonSymbols []macroPoolReasonSymbol
	for _, sym := range symbols {
		reasonSymbols = append(reasonSymbols, macroPoolReasonSymbol{
			Symbol:        strings.TrimSpace(sym.Symbol),
			Direction:     strings.TrimSpace(strings.ToLower(sym.JEVDirection)),
			Confidence:    sym.Confidence,
			ReasonExcerpt: macroPoolReasonExcerpt(sym.JEVReason),
		})
	}
	reason, rerr := jev.SummarizeMacroPoolJEVReason(ctx, poolWorkflow, title, choice, idx, base.Choices, conf, conflict, reasonSymbols)
	if rerr != nil {
		base.Error = rerr.Error()
		return base
	}
	base.OK = true
	base.Error = ""
	base.Direction = choiceToLedgerDirection(choice)
	base.Confidence = conf
	base.Conflict = conflict
	base.RecommendedReason = reason
	return base
}