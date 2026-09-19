package diagram

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

// Renderer turns a Document into self-contained HTML.
type Renderer interface {
	Name() string
	Render(ctx context.Context, doc Document) (Artifact, error)
}

// BuiltinRenderer draws a dark, self-contained SVG viewer from Archify IR.
// Used when Archify/Node is unavailable and as the committed-artifact fallback.
type BuiltinRenderer struct{}

func (BuiltinRenderer) Name() string { return "builtin" }

func (BuiltinRenderer) Render(_ context.Context, doc Document) (Artifact, error) {
	if len(doc.IR) == 0 {
		return Artifact{}, fmt.Errorf("diagram: empty IR for %s", doc.ID)
	}
	body, err := renderBuiltinBody(doc)
	if err != nil {
		return Artifact{}, err
	}
	page := builtinPage(doc.Title, body)
	sum := sha256.Sum256(page)
	return Artifact{
		HTML:        page,
		SHA256:      hex.EncodeToString(sum[:]),
		ContentType: "text/html; charset=utf-8",
		Source:      "builtin",
	}, nil
}

func renderBuiltinBody(doc Document) (string, error) {
	switch doc.Kind {
	case KindArchitecture:
		return renderArchitectureSVG(doc.IR)
	default:
		return renderWorkflowSVG(doc.IR)
	}
}

func builtinPage(title, body string) []byte {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"/>")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"/>")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</title><style>")
	b.WriteString(`html,body{margin:0;height:100%;background:#020617;color:#e7eef7;font:13px/1.45 ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}
.wrap{min-height:100%;padding:18px 20px 28px;box-sizing:border-box}
h1{margin:0 0 6px;font-size:18px;letter-spacing:-.02em}
.sub{color:#94a3b8;margin:0 0 16px;font-size:12px}
.canvas{background:#0f172a;border:1px solid #1e293b;border-radius:16px;padding:16px;overflow:auto}
svg{display:block;max-width:100%;height:auto}
.node{rx:8}
.lbl{fill:#f8fafc;font-size:12px;font-weight:700}
.sublbl{fill:#94a3b8;font-size:10px}
.lane{fill:#0b1220;stroke:#1e293b}
.lanelbl{fill:#64748b;font-size:10px;letter-spacing:.12em}
.edge{fill:none;stroke:#64748b;stroke-width:1.4}
.edge-em{stroke:#22d3ee}
.card{display:inline-block;vertical-align:top;width:280px;margin:14px 12px 0 0;background:#0f172a;border:1px solid #1e293b;border-radius:12px;padding:12px}
.card h2{margin:0 0 8px;font-size:12px;color:#94a3b8;text-transform:uppercase;letter-spacing:.06em}
.card li{margin:0 0 4px}
`)
	b.WriteString("</style></head><body><div class=\"wrap\">")
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1><p class=\"sub\">GeeGoo diagram · 确定性 IR 渲染</p>")
	b.WriteString(body)
	b.WriteString("</div></body></html>")
	return []byte(b.String())
}

func renderWorkflowSVG(raw json.RawMessage) (string, error) {
	var ir workflowIR
	if err := json.Unmarshal(raw, &ir); err != nil {
		return "", err
	}
	laneIndex := map[string]int{}
	for i, lane := range ir.Lanes {
		laneIndex[lane.ID] = i
	}
	const (
		laneH = 120.0
		colW  = 170.0
		x0    = 36.0
		y0    = 36.0
		nw    = 140.0
		nh    = 56.0
	)
	width := 80 + colW*6
	height := 80 + laneH*float64(max(1, len(ir.Lanes)))
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<div class="canvas"><svg viewBox="0 0 %.0f %.0f" xmlns="http://www.w3.org/2000/svg">`, width, height))
	for i, lane := range ir.Lanes {
		y := y0 + float64(i)*laneH
		b.WriteString(fmt.Sprintf(`<rect class="lane" x="16" y="%.1f" width="%.1f" height="%.1f" rx="12"/>`, y-10, width-32, laneH-16))
		b.WriteString(fmt.Sprintf(`<text class="lanelbl" x="28" y="%.1f">%s</text>`, y+8, html.EscapeString(strings.ToUpper(lane.Label))))
	}
	pos := map[string][2]float64{}
	for _, n := range ir.Nodes {
		li := laneIndex[n.Lane]
		x := x0 + float64(n.Col)*colW
		y := y0 + float64(li)*laneH + 28
		pos[n.ID] = [2]float64{x + nw/2, y + nh/2}
		fill, stroke := semanticFill(n.Type)
		b.WriteString(fmt.Sprintf(`<rect class="node" x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s"/>`, x, y, nw, nh, fill, stroke))
		b.WriteString(fmt.Sprintf(`<text class="lbl" x="%.1f" y="%.1f">%s</text>`, x+10, y+24, html.EscapeString(truncate(n.Label, 18))))
		if n.Sublabel != "" {
			b.WriteString(fmt.Sprintf(`<text class="sublbl" x="%.1f" y="%.1f">%s</text>`, x+10, y+42, html.EscapeString(truncate(n.Sublabel, 22))))
		}
	}
	for _, e := range ir.Edges {
		a, okA := pos[e.From]
		c, okB := pos[e.To]
		if !okA || !okB {
			continue
		}
		cls := "edge"
		if e.Variant == "emphasis" {
			cls = "edge edge-em"
		}
		b.WriteString(fmt.Sprintf(`<path class="%s" d="M %.1f %.1f C %.1f %.1f, %.1f %.1f, %.1f %.1f"/>`,
			cls, a[0]+nw/2-8, a[1], a[0]+60, a[1], c[0]-60, c[1], c[0]-nw/2+8, c[1]))
	}
	b.WriteString("</svg></div>")
	b.WriteString(renderCards(ir.Cards))
	return b.String(), nil
}

func renderArchitectureSVG(raw json.RawMessage) (string, error) {
	var ir architectureIR
	if err := json.Unmarshal(raw, &ir); err != nil {
		return "", err
	}
	width, height := 1100.0, 520.0
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<div class="canvas"><svg viewBox="0 0 %.0f %.0f" xmlns="http://www.w3.org/2000/svg">`, width, height))
	pos := map[string][4]float64{}
	for _, n := range ir.Components {
		if len(n.Pos) < 2 || len(n.Size) < 2 {
			continue
		}
		x, y := float64(n.Pos[0]), float64(n.Pos[1])
		w, h := float64(n.Size[0]), float64(n.Size[1])
		pos[n.ID] = [4]float64{x, y, w, h}
		fill, stroke := semanticFill(n.Type)
		b.WriteString(fmt.Sprintf(`<rect class="node" x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s"/>`, x, y, w, h, fill, stroke))
		b.WriteString(fmt.Sprintf(`<text class="lbl" x="%.1f" y="%.1f">%s</text>`, x+10, y+26, html.EscapeString(n.Label)))
		if n.Sublabel != "" {
			b.WriteString(fmt.Sprintf(`<text class="sublbl" x="%.1f" y="%.1f">%s</text>`, x+10, y+44, html.EscapeString(n.Sublabel)))
		}
	}
	for _, e := range ir.Connections {
		a, okA := pos[e.From]
		c, okB := pos[e.To]
		if !okA || !okB {
			continue
		}
		x1, y1 := a[0]+a[2], a[1]+a[3]/2
		x2, y2 := c[0], c[1]+c[3]/2
		cls := "edge"
		if e.Variant == "emphasis" {
			cls = "edge edge-em"
		}
		b.WriteString(fmt.Sprintf(`<path class="%s" d="M %.1f %.1f L %.1f %.1f"/>`, cls, x1, y1, x2, y2))
	}
	b.WriteString("</svg></div>")
	b.WriteString(renderCards(ir.Cards))
	return b.String(), nil
}

func renderCards(cards []workflowCard) string {
	if len(cards) == 0 {
		return ""
	}
	var b strings.Builder
	for _, c := range cards {
		b.WriteString(`<div class="card"><h2>`)
		b.WriteString(html.EscapeString(c.Title))
		b.WriteString("</h2><ul>")
		for _, item := range c.Items {
			if strings.TrimSpace(item) == "" {
				continue
			}
			b.WriteString("<li>")
			b.WriteString(html.EscapeString(item))
			b.WriteString("</li>")
		}
		b.WriteString("</ul></div>")
	}
	return b.String()
}

func semanticFill(kind string) (fill, stroke string) {
	switch kind {
	case "frontend":
		return "#083344", "#22d3ee"
	case "backend":
		return "#052e2b", "#34d399"
	case "database":
		return "#2e1065", "#a78bfa"
	case "cloud":
		return "#422006", "#fbbf24"
	case "security":
		return "#4c0519", "#fb7185"
	case "messagebus":
		return "#431407", "#fb923c"
	default:
		return "#1e293b", "#94a3b8"
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
