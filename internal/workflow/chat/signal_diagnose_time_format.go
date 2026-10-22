package chat

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func normalizeDiagnoseDetailForCard(raw map[string]any) map[string]any {
	if raw == nil {
		return nil
	}
	b, err := jsonCloneMap(raw)
	if err != nil {
		return raw
	}
	for _, key := range []string{"buy_rules", "sell_rules"} {
		rules, ok := b[key].([]any)
		if !ok {
			continue
		}
		for i, item := range rules {
			row, ok := item.(map[string]any)
			if !ok {
				continue
			}
			normalizeDiagnoseRuleTimes(row)
			rules[i] = row
		}
		b[key] = rules
	}
	if rng, ok := b["range"].(map[string]any); ok {
		normalizeDiagnoseRangeTimes(rng)
		b["range"] = rng
	}
	return b
}

func normalizeDiagnoseRangeTimes(rng map[string]any) {
	for _, k := range []string{"start", "end", "from", "to"} {
		if v, ok := rng[k]; ok {
			rng[k] = formatDiagnoseTimeValue(v)
		}
	}
}

func normalizeDiagnoseRuleTimes(row map[string]any) {
	if lb, ok := row["last_bar"].(map[string]any); ok {
		if t, ok := lb["time"]; ok {
			lb["time"] = formatDiagnoseTimeValue(t)
		}
		row["last_bar"] = lb
	}
	if samples, ok := row["sample_reasons"].([]any); ok {
		for i, s := range samples {
			m, ok := s.(map[string]any)
			if !ok {
				continue
			}
			if t, ok := m["time"]; ok {
				m["time"] = formatDiagnoseTimeValue(t)
			}
			samples[i] = m
		}
		row["sample_reasons"] = samples
	}
}

func formatDiagnoseTimeValue(v any) any {
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" || s == "<nil>" {
		return v
	}
	if t, ok := parseDiagnoseTimeString(s); ok {
		return t.Format("2006-01-02 15:04")
	}
	return v
}

func parseDiagnoseTimeString(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, true
		}
	}
	compact := strings.Fields(s)[0]
	if len(compact) >= 12 {
		if t, err := time.ParseInLocation("200601021504", compact[:12], loc); err == nil {
			return t, true
		}
	}
	if len(compact) == 8 {
		if t, err := time.ParseInLocation("20060102", compact, loc); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func jsonCloneMap(raw map[string]any) (map[string]any, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}
