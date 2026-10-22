package llm

import (
	"regexp"
	"strings"
)

// Fixed diag_compose section titles (must match system/user prompt).
var ComposeSectionTitles = []string{
	"一、信号触发情况",
	"二、probe 与命中率",
	"三、Episode 与风险",
	"四、综合评价",
}

var composeSectionGlueRes []*regexp.Regexp

func init() {
	for _, title := range ComposeSectionTitles {
		esc := regexp.QuoteMeta(title)
		// ### 一、信号触发情况在腾讯... → break before body 在
		composeSectionGlueRes = append(composeSectionGlueRes,
			regexp.MustCompile(`(?m)(###\s*`+esc+`)([\p{Han}])`),
		)
	}
}

// NormalizeComposeSections inserts newlines after fixed section headings when the model glues body text.
func NormalizeComposeSections(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = NormalizeComposeMarkdown(s)
	for _, re := range composeSectionGlueRes {
		s = re.ReplaceAllString(s, "$1\n\n$2")
	}
	// Generic: ### 一、xxx情况在 / 下 / 本 → common body openers after 情况|评价|风险
	s = regexp.MustCompile(`(?m)(###\s*[一二三四]、[^\n]{2,12}(?:情况|评价|风险))([在对于对本据根])`).ReplaceAllString(s, "$1\n\n$2")
	return strings.TrimSpace(s)
}
