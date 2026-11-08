package klinepattern

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// klinePatternLabelCN maps canonical pattern id → 中文展示（仅当 Signal 未给 name_chs 时回退用，不做过滤）。
var klinePatternLabelCN = map[string]string{
	"Doji":               "十字线",
	"DragonflyDoji":      "蜻蜓十字线",
	"GravestoneDoji":     "墓碑十字线",
	"DojiStar":           "十字星",
	"Hammer":             "锤子线",
	"HangingMan":         "上吊线",
	"InvertedHammer":     "倒锤线",
	"HighWave":           "风高浪大线",
	"Marubozu":           "烛台线",
	"ShootingStar":       "流星线",
	"Belthold":           "捉腰带线",
	"Engulfing":          "吞没模式",
	"Piercing":           "刺透模式",
	"Harami":             "母子线",
	"HaramiCross":        "十字孕线",
	"DarkCloudCover":     "乌云压顶",
	"HomingPigeon":       "家鸽",
	"MatchingLow":        "相同低价",
	"ThreeInside":        "三内部上涨和下跌",
	"ThreeOutside":       "三外部上涨和下跌",
	"EveningDojiStar":    "十字暮星",
	"EveningStar":        "暮星",
	"MorningStar":        "晨星",
	"MorningDojiStar":    "十字晨星",
	"Tristar":            "三星",
	"ThreeCrows":         "三只乌鸦",
	"ThreeWhiteSoldiers": "三个白兵",
	"TwoCrows":           "两只乌鸦",
}

var dojiPatternNames = map[string]struct{}{
	"Doji":            {},
	"DragonflyDoji":   {},
	"GravestoneDoji":  {},
	"DojiStar":        {},
	"HaramiCross":     {},
	"MorningDojiStar": {},
	"EveningDojiStar": {},
	"Tristar":         {},
}

// PatternHitsSeriesFromProbe reads pattern_hits aligned with bars from probe/diagnose payload.
func PatternHitsSeriesFromProbe(data map[string]any) []any {
	if data == nil {
		return nil
	}
	raw, ok := data["pattern_hits"].([]any)
	if !ok || len(raw) == 0 {
		return nil
	}
	return raw
}

// FilteredKlinePatternsAtIndex returns probe pattern hits for one bar (Signal is source of truth; no name whitelist).
func FilteredKlinePatternsAtIndex(series []any, idx int) []slots.KlinePatternHit {
	if len(series) == 0 || idx < 0 || idx >= len(series) {
		return nil
	}
	rawHits := parseBarPatternHits(series[idx])
	if len(rawHits) == 0 {
		return nil
	}
	out := make([]slots.KlinePatternHit, 0, len(rawHits))
	for _, h := range rawHits {
		name := strings.TrimSpace(h.name)
		labelCN := strings.TrimSpace(h.labelCN)
		if name == "" && labelCN == "" {
			continue
		}
		if labelCN == "" {
			if cn, ok := klinePatternLabelCN[canonicalPatternName(name)]; ok {
				labelCN = cn
			} else if cn, ok := klinePatternLabelCN[name]; ok {
				labelCN = cn
			} else {
				labelCN = name
			}
		}
		if name == "" {
			name = labelCN
		}
		out = append(out, slots.KlinePatternHit{
			Name:      name,
			Direction: h.direction,
			LabelCN:   labelCN,
		})
	}
	return out
}

type parsedPatternHit struct {
	name      string
	direction int
	labelCN   string
}

func parseBarPatternHits(raw any) []parsedPatternHit {
	switch v := raw.(type) {
	case nil:
		return nil
	case []any:
		return parsePatternHitList(v)
	case map[string]any:
		if inner, ok := v["hits"].([]any); ok {
			return parsePatternHitList(inner)
		}
		if inner, ok := v["patterns"].([]any); ok {
			return parsePatternHitList(inner)
		}
		return parsePatternHitMap(v)
	default:
		return nil
	}
}

func parsePatternHitList(items []any) []parsedPatternHit {
	out := make([]parsedPatternHit, 0, len(items))
	for _, item := range items {
		switch row := item.(type) {
		case map[string]any:
			name, labelCN, ok := patternIdentityFromProbeRow(row)
			if !ok {
				continue
			}
			dir := normalizePatternDirection(name, patternDirectionValue(row))
			out = append(out, parsedPatternHit{
				name:      name,
				direction: dir,
				labelCN:   labelCN,
			})
		case string:
			name, labelCN, ok := patternIdentityFromProbeString(row)
			if !ok {
				continue
			}
			out = append(out, parsedPatternHit{
				name:      name,
				direction: defaultDirectionForName(name),
				labelCN:   labelCN,
			})
		}
	}
	return out
}

func parsePatternHitMap(m map[string]any) []parsedPatternHit {
	out := make([]parsedPatternHit, 0, len(m))
	for key, val := range m {
		if isMetaPatternKey(key) {
			continue
		}
		name := canonicalPatternName(key)
		if name == "" {
			name = strings.TrimSpace(key)
		}
		if name == "" {
			continue
		}
		dir := normalizePatternDirection(name, slots.FloatAny(val))
		out = append(out, parsedPatternHit{name: name, direction: dir})
	}
	return out
}

func isMetaPatternKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "hits", "patterns", "count", "note":
		return true
	default:
		return false
	}
}

func patternDirectionValue(row map[string]any) float64 {
	for _, key := range []string{"direction", "dir", "sign", "pattern", "value", "signal", "side"} {
		if v, ok := row[key]; ok {
			if key == "pattern" {
				if _, isStr := v.(string); isStr {
					continue
				}
			}
			return slots.FloatAny(v)
		}
	}
	return 0
}

// patternHitName reads English TA-Lib name from GeeGooSignal pattern_hits entries.
func patternHitName(row map[string]any) string {
	if s := firstString(row, "name", "Name"); s != "" && !looksLikeNumericDirection(s) {
		return s
	}
	return firstString(row, "index", "id")
}

func patternIdentityFromProbeRow(row map[string]any) (name, labelCN string, ok bool) {
	rawName := strings.TrimSpace(patternHitName(row))
	labelCN = firstString(row, "name_chs", "label_cn", "NameCHS")
	name = canonicalPatternName(rawName)
	if name == "" {
		name = canonicalPatternNameFromLabelCN(labelCN)
	}
	if name == "" && rawName != "" {
		name = rawName
	}
	if name == "" && labelCN != "" {
		name = labelCN
	}
	if name == "" && labelCN == "" {
		return "", "", false
	}
	return name, labelCN, true
}

func patternIdentityFromProbeString(raw string) (name, labelCN string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}
	name = canonicalPatternName(raw)
	if name == "" {
		name = raw
	}
	return name, "", true
}

func canonicalPatternNameFromLabelCN(labelCN string) string {
	labelCN = strings.TrimSpace(labelCN)
	if labelCN == "" {
		return ""
	}
	for en, cn := range klinePatternLabelCN {
		if cn == labelCN {
			return en
		}
	}
	return ""
}

func looksLikeNumericDirection(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	switch s {
	case "1", "-1", "2", "-2", "100", "-100":
		return true
	}
	var f float64
	_, err := fmt.Sscan(s, &f)
	return err == nil && strings.ContainsAny(s, "0123456789")
}

func firstString(row map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := row[k]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func canonicalPatternName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.TrimPrefix(raw, "CDL")
	if _, ok := klinePatternLabelCN[raw]; ok {
		return raw
	}
	compact := strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || r == ' ' {
			return -1
		}
		return r
	}, raw)
	for name := range klinePatternLabelCN {
		key := strings.Map(func(r rune) rune {
			if r == '_' || r == '-' || r == ' ' {
				return -1
			}
			return r
		}, name)
		if strings.EqualFold(compact, key) {
			return name
		}
	}
	return ""
}

func normalizePatternDirection(name string, v float64) int {
	if v == 2 || v == -2 {
		return 2
	}
	if int(v) == 2 {
		return 2
	}
	if _, isDoji := dojiPatternNames[patternCanonicalKey(name)]; isDoji && v == 0 {
		return 2
	}
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return defaultDirectionForName(name)
}

func defaultDirectionForName(name string) int {
	if _, ok := dojiPatternNames[patternCanonicalKey(name)]; ok {
		return 2
	}
	return 0
}

func patternCanonicalKey(name string) string {
	if c := canonicalPatternName(name); c != "" {
		return c
	}
	return strings.TrimSpace(name)
}

// EnrichSignalEpisodeEvalPatterns attaches trigger-bar patterns to episode start indices.
func EnrichSignalEpisodeEvalPatterns(eval *slots.SignalEpisodeEval, series []any) {
	if eval == nil || len(series) == 0 {
		return
	}
	for i := range eval.BuyDetails {
		AttachPatternsToEpisode(&eval.BuyDetails[i], series)
	}
	for i := range eval.SellDetails {
		AttachPatternsToEpisode(&eval.SellDetails[i], series)
	}
}

// AttachPatternsToEpisode fills pattern hits on one episode detail row.
func AttachPatternsToEpisode(ep *slots.SignalEpisodeDetail, series []any) {
	if ep == nil {
		return
	}
	hits := FilteredKlinePatternsAtIndex(series, ep.StartIdx)
	if len(hits) == 0 {
		ep.KlinePatterns = nil
		ep.KlinePatternSummary = ""
		return
	}
	ep.KlinePatterns = hits
	ep.KlinePatternSummary = slots.KlinePatternSummary(hits)
}
