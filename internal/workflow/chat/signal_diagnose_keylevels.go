package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func floatSliceField(raw any, key string) []float64 {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	items, ok := m[key].([]any)
	if !ok {
		return nil
	}
	out := make([]float64, len(items))
	for i, v := range items {
		out[i] = slots.FloatAny(v)
	}
	return out
}

func keyLevelBarSeriesFromProbeData(raw any) *KeyLevelBarSeries {
	m, ok := raw.(map[string]any)
	if !ok || len(m) == 0 {
		return nil
	}
	series := &KeyLevelBarSeries{
		Align:         strings.TrimSpace(fmt.Sprint(m["align"])),
		SupportLow:    floatSliceField(m, "support_low"),
		SupportHigh:   floatSliceField(m, "support_high"),
		SupportCenter: floatSliceField(m, "support_center"),
		ResistLow:     floatSliceField(m, "resist_low"),
		ResistHigh:    floatSliceField(m, "resist_high"),
		ResistCenter:  floatSliceField(m, "resist_center"),
	}
	if len(series.SupportLow) == 0 && len(series.ResistHigh) == 0 {
		return nil
	}
	return series
}

func keyLevelSnapshotFromProbeData(raw any) KeyLevelSnapshot {
	m, ok := raw.(map[string]any)
	if !ok || len(m) == 0 {
		return KeyLevelSnapshot{}
	}
	if _, hasJudgment := m["judgment"]; hasJudgment {
		return keyLevelSnapshotFromToolData(m)
	}
	return KeyLevelSnapshot{
		SupportLow:    floatMap(m, "support_low"),
		SupportHigh:   floatMap(m, "support_high"),
		SupportCenter: floatMap(m, "support_center"),
		ResistLow:     floatMap(m, "resist_low"),
		ResistHigh:    floatMap(m, "resist_high"),
		ResistCenter:  floatMap(m, "resist_center"),
		Summary:       strings.TrimSpace(fmt.Sprint(m["summary"])),
	}
}

func keyLevelSnapshotUsableForEpisodeStop(kl KeyLevelSnapshot) bool {
	return kl.SupportLow > 0 || kl.ResistHigh > 0
}

func keyLevelSnapshotFromToolData(data map[string]any) KeyLevelSnapshot {
	if len(data) == 0 {
		return KeyLevelSnapshot{}
	}
	out := KeyLevelSnapshot{Summary: strings.TrimSpace(fmt.Sprint(data["summary"]))}
	judgment, _ := data["judgment"].(map[string]any)
	if judgment == nil {
		return out
	}
	if s := strings.TrimSpace(fmt.Sprint(judgment["summary"])); s != "" && out.Summary == "" {
		out.Summary = s
	}
	if sup, _ := judgment["support"].(map[string]any); sup != nil {
		out.SupportLow = floatMap(sup, "low")
		out.SupportHigh = floatMap(sup, "high")
		out.SupportCenter = floatMap(sup, "center")
	}
	if res, _ := judgment["resistance"].(map[string]any); res != nil {
		out.ResistLow = floatMap(res, "low")
		out.ResistHigh = floatMap(res, "high")
		out.ResistCenter = floatMap(res, "center")
	}
	return out
}

func floatMap(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	return slots.FloatAny(m[key])
}

func (kl KeyLevelSnapshot) episodeStop(mode, buyRef, sellRef string, series *KeyLevelBarSeries) *slots.KeyLevelEpisodeStop {
	if kl.SupportLow <= 0 && kl.ResistHigh <= 0 && series == nil {
		return nil
	}
	stop := &slots.KeyLevelEpisodeStop{
		SupportLow:    kl.SupportLow,
		SupportHigh:   kl.SupportHigh,
		SupportCenter: kl.SupportCenter,
		ResistLow:     kl.ResistLow,
		ResistHigh:    kl.ResistHigh,
		ResistCenter:  kl.ResistCenter,
		Mode:          normalizeKeyBreakMode(mode),
		BuyBreakRef:  normalizeKeyBreakRef(buyRef, defaultBuyBreakRef(mode)),
		SellBreakRef: normalizeKeyBreakRef(sellRef, defaultSellBreakRef(mode)),
	}
	if series != nil {
		stop.SupportLowSeries = series.SupportLow
		stop.SupportHighSeries = series.SupportHigh
		stop.SupportCenterSeries = series.SupportCenter
		stop.ResistLowSeries = series.ResistLow
		stop.ResistHighSeries = series.ResistHigh
		stop.ResistCenterSeries = series.ResistCenter
	}
	if !stop.HasLevels() {
		return nil
	}
	return stop
}

func (r *Runner) phaseSignalDiagnoseFetchKeyLevels(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	if flow.StockCode == "" {
		flow.Phase = PhaseEvaluateAccuracy
		flow.touch()
		return nil
	}
	if keyLevelSnapshotUsableForEpisodeStop(flow.KeyLevels) {
		flow.Phase = PhaseEvaluateAccuracy
		flow.touch()
		return nil
	}
	res := r.runTool(ctx, toolCtx, "get_key_levels", map[string]any{
		"code": flow.StockCode,
	}, recordTool)
	if res.Status == tools.StatusOK {
		flow.KeyLevels = keyLevelSnapshotFromToolData(res.Data)
		if flow.KeyLevels.Summary == "" {
			flow.KeyLevels.Summary = strings.TrimSpace(res.Summary)
		}
	} else {
		// Non-fatal: fall back to until_opposite only.
		flow.KeyLevels = KeyLevelSnapshot{Summary: "关键价位未获取：" + strings.TrimSpace(res.Summary)}
	}
	flow.Phase = PhaseEvaluateAccuracy
	flow.touch()
	return nil
}
