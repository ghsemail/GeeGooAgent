package stockfmt

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// KeyLevels holds extracted support/resistance from weekly analysis text.
type KeyLevels struct {
	Support    *float64
	Resistance *float64
	Valid      bool
	Warnings   []string
}

var (
	priceRangeRE = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*[-~至到—]\s*(\d+(?:\.\d+)?)`)
	singlePriceRE = regexp.MustCompile(`(\d+(?:\.\d+)?)`)
	supportLabelREs = []*regexp.Regexp{
		regexp.MustCompile(`(?i)第一支撑位`),
		regexp.MustCompile(`(?i)第二支撑位`),
		regexp.MustCompile(`(?i)短期支撑位`),
		regexp.MustCompile(`(?i)主要支撑位`),
		regexp.MustCompile(`(?i)关键支撑位`),
		regexp.MustCompile(`(?i)近期支撑位`),
		regexp.MustCompile(`(?i)支撑位`),
		regexp.MustCompile(`(?i)\bS1\b`),
	}
	resistanceLabelREs = []*regexp.Regexp{
		regexp.MustCompile(`(?i)第一阻力位`),
		regexp.MustCompile(`(?i)第二阻力位`),
		regexp.MustCompile(`(?i)短期阻力位`),
		regexp.MustCompile(`(?i)主要阻力位`),
		regexp.MustCompile(`(?i)关键阻力位`),
		regexp.MustCompile(`(?i)近期阻力位`),
		regexp.MustCompile(`(?i)阻力位`),
		regexp.MustCompile(`(?i)压力位`),
		regexp.MustCompile(`(?i)上方压力`),
		regexp.MustCompile(`(?i)\bR1\b`),
	}
)

// ExtractWeeklyKeyLevels parses support/resistance from weekly analysis markdown.
// Support takes the lower bound of ranges; resistance takes the upper bound.
func ExtractWeeklyKeyLevels(text, code string) KeyLevels {
	text = strings.TrimSpace(text)
	if text == "" {
		return KeyLevels{}
	}
	support, _ := findLevelByLabels(text, supportLabelREs, false)
	if support == nil {
		support, _ = findLevelByKeyword(text, "支撑", false)
	}
	resistance, _ := findLevelByLabels(text, resistanceLabelREs, true)
	if resistance == nil {
		resistance, _ = findLevelByKeyword(text, "阻力", true)
	}
	if resistance == nil {
		resistance, _ = findLevelByKeyword(text, "压力", true)
	}

	warnings := make([]string, 0, 2)
	if support != nil && isCodeArtifact(*support, code) {
		warnings = append(warnings, "support looks like code digits")
		support = nil
	}
	if resistance != nil && isCodeArtifact(*resistance, code) {
		warnings = append(warnings, "resistance looks like code digits")
		resistance = nil
	}
	if support != nil && resistance != nil {
		if math.Abs(*support-*resistance) < 1e-6 {
			warnings = append(warnings, "support equals resistance")
			support, resistance = nil, nil
		} else if *support > *resistance {
			warnings = append(warnings, "support > resistance, swapped")
			support, resistance = resistance, support
		}
	}
	return KeyLevels{
		Support:    support,
		Resistance: resistance,
		Valid:      support != nil && resistance != nil,
		Warnings:   warnings,
	}
}

// ApplyPriceSanity discards levels outside [5%, 300%] of current price.
func ApplyPriceSanity(levels KeyLevels, currentPrice float64) KeyLevels {
	if currentPrice <= 0 || !levels.Valid {
		return levels
	}
	lo, hi := currentPrice*0.05, currentPrice*3.0
	bad := func(v float64) bool { return v < lo || v > hi }
	if levels.Support != nil && bad(*levels.Support) {
		levels.Warnings = append(levels.Warnings, "support out of sane price range")
		levels.Support = nil
	}
	if levels.Resistance != nil && bad(*levels.Resistance) {
		levels.Warnings = append(levels.Warnings, "resistance out of sane price range")
		levels.Resistance = nil
	}
	levels.Valid = levels.Support != nil && levels.Resistance != nil
	return levels
}

// CapitalFlowDivergent reports mixed retail/main-force signals in capital summaries.
func CapitalFlowDivergent(capitalSummary string) bool {
	s := strings.TrimSpace(capitalSummary)
	if s == "" {
		return false
	}
	retailIn := strings.Contains(s, "小单净流入")
	mainWeak := strings.Contains(s, "主力偏弱") || strings.Contains(s, "主力净流出")
	bigOutSmallIn := strings.Contains(s, "大单净流出") && strings.Contains(s, "小单净流入")
	superOutBigIn := strings.Contains(s, "超大单净流出") && strings.Contains(s, "大单净流入")
	return (retailIn && mainWeak) || bigOutSmallIn || superOutBigIn
}

func findLevelByLabels(text string, labels []*regexp.Regexp, useHigh bool) (*float64, []float64) {
	for _, re := range labels {
		idxs := re.FindAllStringIndex(text, -1)
		for _, loc := range idxs {
			if len(loc) < 2 {
				continue
			}
			segment := text[loc[1]:]
			if len(segment) > 160 {
				segment = segment[:160]
			}
			if price, raw := parsePriceSegment(segment, useHigh); price != nil {
				return price, raw
			}
		}
	}
	return nil, nil
}

func findLevelByKeyword(text, keyword string, useHigh bool) (*float64, []float64) {
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, keyword) {
			continue
		}
		if price, raw := parsePriceSegment(line, useHigh); price != nil {
			return price, raw
		}
		if prices := allPricesInLine(line); len(prices) > 0 {
			pick := prices[0]
			if useHigh {
				for _, p := range prices[1:] {
					if p > pick {
						pick = p
					}
				}
			} else {
				for _, p := range prices[1:] {
					if p < pick {
						pick = p
					}
				}
			}
			return &pick, prices
		}
	}
	return nil, nil
}

func allPricesInLine(line string) []float64 {
	matches := singlePriceRE.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]float64, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			continue
		}
		out = append(out, roundPrice(v))
	}
	return out
}

func parsePriceSegment(segment string, useHigh bool) (*float64, []float64) {
	if m := priceRangeRE.FindStringSubmatch(segment); len(m) == 3 {
		lo, err1 := strconv.ParseFloat(m[1], 64)
		hi, err2 := strconv.ParseFloat(m[2], 64)
		if err1 == nil && err2 == nil {
			if lo > hi {
				lo, hi = hi, lo
			}
			raw := []float64{roundPrice(lo), roundPrice(hi)}
			pick := raw[0]
			if useHigh {
				pick = raw[1]
			}
			return &pick, raw
		}
	}
	if m := singlePriceRE.FindStringSubmatch(segment); len(m) == 2 {
		v, err := strconv.ParseFloat(m[1], 64)
		if err == nil {
			p := roundPrice(v)
			return &p, nil
		}
	}
	return nil, nil
}

func roundPrice(v float64) float64 {
	return math.Round(v*100) / 100
}

func isCodeArtifact(value float64, code string) bool {
	prefix := strings.SplitN(code, ".", 2)[0]
	digits := strings.TrimLeft(prefix, "0")
	if digits == "" {
		digits = "0"
	}
	cn, err := strconv.ParseFloat(digits, 64)
	if err != nil {
		return false
	}
	return math.Abs(value-cn) < math.Max(1.0, cn*0.001)
}
