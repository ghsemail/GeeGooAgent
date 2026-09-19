package stockfmt

import (
	"fmt"
	"math"
	"strings"
)

// KeyLevelProvenance describes where support/resistance came from.
type KeyLevelProvenance struct {
	EngineUsed   bool
	TextUsed     bool
	Merged       bool
	SupportZone  string
	ResistZone   string
	SupportSrc   []string
	ResistSrc    []string
}

// MergeKeyLevels combines engine zones (primary) with weekly MCP text extraction (secondary / cross-check).
func MergeKeyLevels(engine, text KeyLevels, currentPrice float64) KeyLevels {
	out := KeyLevels{
		Support:    engine.Support,
		Resistance: engine.Resistance,
		Valid:      engine.Valid,
		Warnings:   append([]string(nil), engine.Warnings...),
		Provenance: engine.Provenance,
	}
	if out.Provenance.EngineUsed {
		out.Provenance.TextUsed = text.Valid
	}
	if !engine.Valid && text.Valid {
		out = text
		out.Provenance = KeyLevelProvenance{TextUsed: true}
		return ApplyPriceSanity(out, currentPrice)
	}
	if !engine.Valid {
		return ApplyPriceSanity(text, currentPrice)
	}
	if !text.Valid {
		return ApplyPriceSanity(out, currentPrice)
	}

	// Cross-check: if text deviates >15% from engine center, note warning but keep engine.
	if engine.Support != nil && text.Support != nil {
		if relDiff(*engine.Support, *text.Support) > 0.15 {
			out.Warnings = append(out.Warnings, fmt.Sprintf("weekly text support %.2f differs from engine %.2f", *text.Support, *engine.Support))
		}
	}
	if engine.Resistance != nil && text.Resistance != nil {
		if relDiff(*engine.Resistance, *text.Resistance) > 0.15 {
			out.Warnings = append(out.Warnings, fmt.Sprintf("weekly text resistance %.2f differs from engine %.2f", *text.Resistance, *engine.Resistance))
		}
	}
	out.Provenance.Merged = true
	out.Provenance.TextUsed = true
	return ApplyPriceSanity(out, currentPrice)
}

func relDiff(a, b float64) float64 {
	if a == 0 && b == 0 {
		return 0
	}
	d := math.Abs(a - b)
	m := math.Max(math.Abs(a), math.Abs(b))
	if m <= 0 {
		return d
	}
	return d / m
}

// FormatKeyLevelLine renders a human-readable S/R line for reports.
func FormatKeyLevelLine(levels KeyLevels) string {
	if !levels.Valid {
		return ""
	}
	parts := []string{
		fmt.Sprintf("关键支撑约 %.2f、阻力约 %.2f", *levels.Support, *levels.Resistance),
	}
	if levels.Provenance.SupportZone != "" {
		parts = append(parts, "支撑带 "+levels.Provenance.SupportZone)
	}
	if levels.Provenance.ResistZone != "" {
		parts = append(parts, "阻力带 "+levels.Provenance.ResistZone)
	}
	if len(levels.Provenance.SupportSrc) > 0 {
		parts = append(parts, "支撑证据："+strings.Join(levels.Provenance.SupportSrc, ", "))
	}
	return strings.Join(parts, "；")
}
