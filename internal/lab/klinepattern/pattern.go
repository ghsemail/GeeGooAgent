package klinepattern

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// allowedKlinePatternNames mirrors GeeGooSignal talib_lib pattern_list (28 TA-Lib CDL names).
var allowedKlinePatternNames = map[string]string{
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

// FilteredKlinePatternsAtIndex returns whitelisted patterns for bar idx.
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
		if _, ok := allowedKlinePatternNames[h.name]; !ok {
			continue
		}
		labelCN := strings.TrimSpace(h.labelCN)
		if labelCN == "" {
			labelCN = allowedKlinePatternNames[h.name]
		}
		out = append(out, slots.KlinePatternHit{
			Name:      h.name,
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
			name := canonicalPatternName(patternHitName(row))
			if name == "" {
				continue
			}
			dir := normalizePatternDirection(name, patternDirectionValue(row))
			out = append(out, parsedPatternHit{
				name:      name,
				direction: dir,
				labelCN:   firstString(row, "name_chs", "label_cn"),
			})
		case string:
			name := canonicalPatternName(row)
			if name == "" {
				continue
			}
			out = append(out, parsedPatternHit{
				name:      name,
				direction: defaultDirectionForName(name),
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
	if s := firstString(row, "name"); s != "" && !looksLikeNumericDirection(s) {
		return s
	}
	return firstString(row, "index", "id")
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
	if alt, ok := allowedKlinePatternNames[raw]; ok && alt != "" {
		return raw
	}
	compact := strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || r == ' ' {
			return -1
		}
		return r
	}, raw)
	for name := range allowedKlinePatternNames {
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
	if _, isDoji := dojiPatternNames[name]; isDoji && v == 0 {
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
	if _, ok := dojiPatternNames[name]; ok {
		return 2
	}
	return 0
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
