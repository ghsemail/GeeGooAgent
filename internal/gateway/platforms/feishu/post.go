package feishu

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	markdownFenceOpenRe  = regexp.MustCompile("^```[^`]*$")
	markdownFenceCloseRe = regexp.MustCompile("^```$")
)

// BuildMarkdownPostPayload builds Feishu post JSON with md elements (Hermes-aligned).
func BuildMarkdownPostPayload(content string) string {
	rows := BuildMarkdownPostRows(content)
	b, _ := json.Marshal(map[string]any{
		"zh_cn": map[string]any{"content": rows},
	})
	return string(b)
}

// BuildMarkdownPostRows splits fenced code blocks into separate md rows so Feishu
// does not swallow trailing prose after a code fence.
func BuildMarkdownPostRows(content string) [][]map[string]string {
	if content == "" {
		return [][]map[string]string{{{"tag": "md", "text": ""}}}
	}
	if !strings.Contains(content, "```") {
		return [][]map[string]string{{{"tag": "md", "text": content}}}
	}

	var rows [][]map[string]string
	var current []string
	inCode := false

	flush := func() {
		if len(current) == 0 {
			return
		}
		segment := strings.Join(current, "\n")
		if strings.TrimSpace(segment) != "" {
			rows = append(rows, []map[string]string{{"tag": "md", "text": segment}})
		}
		current = nil
	}

	for _, rawLine := range strings.Split(content, "\n") {
		stripped := strings.TrimSpace(rawLine)
		isFence := false
		if inCode {
			isFence = markdownFenceCloseRe.MatchString(stripped)
		} else {
			isFence = markdownFenceOpenRe.MatchString(stripped)
		}
		if isFence {
			if !inCode {
				flush()
			}
			current = append(current, rawLine)
			inCode = !inCode
			if !inCode {
				flush()
			}
			continue
		}
		current = append(current, rawLine)
	}
	flush()
	if len(rows) == 0 {
		return [][]map[string]string{{{"tag": "md", "text": content}}}
	}
	return rows
}

// PlainTextFallback strips light markdown for text msg_type fallback.
func PlainTextFallback(content string) string {
	s := content
	s = regexp.MustCompile("(?m)^#{1,6}\\s*").ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "`", "")
	return strings.TrimSpace(s)
}
