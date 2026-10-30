package llm

import (
	"regexp"
	"strings"
)

var composeSectionMarkers = []string{"[[S1]]", "[[S2]]", "[[S3]]", "[[S4]]"}

var composeFixedHeadings = []string{
	"一、信号触发情况",
	"二、probe 与命中率",
	"三、Episode 与风险",
	"四、综合评价",
}

// ComposeSectionHeadings returns the fixed ### titles for signal diagnose compose cards.
func ComposeSectionHeadings() []string {
	out := make([]string, len(composeFixedHeadings))
	copy(out, composeFixedHeadings)
	return out
}

var composeDelimiterRE = regexp.MustCompile(`\[\[S([1-4])\]\]`)

// ParseComposeSectionBodies splits model output by [[S1]]..[[S4]] markers.
// Without markers, all text goes to section 1.
func ParseComposeSectionBodies(raw string) [4]string {
	var bodies [4]string
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return bodies
	}
	idxs := composeDelimiterRE.FindAllStringSubmatchIndex(raw, -1)
	if len(idxs) == 0 {
		bodies[0] = normalizeComposeParagraphs(raw)
		return bodies
	}
	for i, loc := range idxs {
		num := raw[loc[2]:loc[3]]
		sec := int(num[0] - '1')
		if sec < 0 || sec > 3 {
			continue
		}
		contentStart := loc[1]
		contentEnd := len(raw)
		if i+1 < len(idxs) {
			contentEnd = idxs[i+1][0]
		}
		body := strings.TrimSpace(raw[contentStart:contentEnd])
		bodies[sec] = normalizeComposeParagraphs(body)
	}
	return bodies
}

func normalizeComposeParagraphs(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	var paras []string
	var cur strings.Builder
	flush := func() {
		if t := strings.TrimSpace(cur.String()); t != "" {
			paras = append(paras, t)
		}
		cur.Reset()
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		if cur.Len() > 0 {
			cur.WriteByte('\n')
		}
		cur.WriteString(strings.TrimSpace(line))
	}
	flush()
	return strings.Join(paras, "\n\n")
}

// RenderComposeSectionDocument renders fixed ### headings + bodies (no delimiters).
func RenderComposeSectionDocument(bodies [4]string) string {
	var b strings.Builder
	for i, heading := range composeFixedHeadings {
		body := strings.TrimSpace(bodies[i])
		if body == "" && i > 0 {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("### ")
		b.WriteString(heading)
		b.WriteString("\n\n")
		if body != "" {
			b.WriteString(body)
		}
	}
	return strings.TrimSpace(b.String())
}

// ComposeSectionRenderer incrementally renders streamed model body text.
type ComposeSectionRenderer struct {
	raw          strings.Builder
	renderedPrev string
	OnRendered   func(chunk string)
}

func (r *ComposeSectionRenderer) PushModelText(delta string) {
	if r == nil || delta == "" {
		return
	}
	r.raw.WriteString(delta)
	rendered := RenderComposeSectionDocument(ParseComposeSectionBodies(r.raw.String()))
	if rendered == r.renderedPrev {
		return
	}
	if strings.HasPrefix(rendered, r.renderedPrev) {
		chunk := rendered[len(r.renderedPrev):]
		r.renderedPrev = rendered
		if chunk != "" && r.OnRendered != nil {
			r.OnRendered(chunk)
		}
		return
	}
	r.renderedPrev = rendered
	if rendered != "" && r.OnRendered != nil {
		r.OnRendered(rendered)
	}
}

func (r *ComposeSectionRenderer) Document() string {
	return RenderComposeSectionDocument(ParseComposeSectionBodies(r.raw.String()))
}

// ParseRenderedComposeSectionDocument splits a rendered ### heading document into section bodies.
func ParseRenderedComposeSectionDocument(doc string) ([4]string, bool) {
	doc = strings.TrimSpace(doc)
	if doc == "" {
		return [4]string{}, false
	}
	if strings.HasPrefix(doc, "### Agent 评价") {
		doc = strings.TrimSpace(strings.TrimPrefix(doc, "### Agent 评价"))
	}
	var bodies [4]string
	found := false
	for i, heading := range composeFixedHeadings {
		marker := "### " + heading
		idx := strings.Index(doc, marker)
		if idx < 0 {
			continue
		}
		found = true
		start := idx + len(marker)
		rest := strings.TrimSpace(doc[start:])
		end := len(rest)
		for j := i + 1; j < len(composeFixedHeadings); j++ {
			next := "### " + composeFixedHeadings[j]
			if pos := strings.Index(rest, next); pos >= 0 {
				end = pos
				break
			}
		}
		bodies[i] = strings.TrimSpace(rest[:end])
	}
	return bodies, found
}

// RenderComposeSectionDocumentAll renders all four card headings; empty bodies stay blank under the title.
func RenderComposeSectionDocumentAll(bodies [4]string) string {
	var b strings.Builder
	for i, heading := range composeFixedHeadings {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("### ")
		b.WriteString(heading)
		b.WriteString("\n\n")
		if body := strings.TrimSpace(bodies[i]); body != "" {
			b.WriteString(body)
		}
	}
	return strings.TrimSpace(b.String())
}
