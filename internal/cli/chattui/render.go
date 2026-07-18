package chattui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

var (
	styleFocus = lipgloss.NewStyle().Foreground(lipgloss.Color(chatui.ColorGold))
)

type transcriptSegment int

const (
	segmentNone transcriptSegment = iota
	segmentUser
	segmentProcess // thinking / tools / activity
	segmentReply
)

func (m *Model) rebuildBanner() {
	if m.width <= 0 {
		m.width = 80
	}
	m.banner = chatui.RenderBanner(m.bannerOpts, m.width, false)
}

func (m *Model) statusBarOpts() chatui.StatusBarOptions {
	s := m.activeSlot()
	opts := chatui.StatusBarOptions{
		ContextWindow: llm.DefaultContextWindow,
	}
	if s == nil {
		return opts
	}
	opts.Busy = s.Busy
	opts.Steps = len(s.Repl.StepLog)
	if s.Host != nil && s.Host.Repl != nil {
		r := s.Host.Repl
		cfg := r.App.Config
		model := s.Host.ModelLine()
		opts.Model = model
		opts.PromptTokens = r.Session.LastPromptTokens
		opts.ContextWindow = llm.ResolveContextWindow(model, cfg.Compression.ContextLength)
	}
	if s.Busy && !s.TurnStartedAt.IsZero() {
		opts.Elapsed = time.Since(s.TurnStartedAt)
	} else if !s.TurnEndedAt.IsZero() && !s.TurnStartedAt.IsZero() {
		opts.Elapsed = s.TurnEndedAt.Sub(s.TurnStartedAt)
	}
	return opts
}

func (m *Model) renderTranscript() string {
	width := m.width
	if width <= 0 {
		width = 80
	}
	var b strings.Builder
	if m.banner != "" {
		b.WriteString(m.banner)
	}
	s := m.activeSlot()
	if s == nil {
		return b.String()
	}

	hasSegment := m.banner != ""
	var prev transcriptSegment = segmentNone
	if m.banner != "" {
		prev = segmentProcess // banner counts as prior content for first user turn rule
	}
	for i, block := range s.Blocks {
		if !block.IsVisible(m.display) {
			continue
		}
		if block.Kind == KindReply {
			body := strings.TrimRight(block.Body, "\n")
			if body == "" && !block.Live {
				continue
			}
		}
		cur := blockSegment(block.Kind)
		if hasSegment {
			m.writeSegmentDivider(&b, width, prev, cur)
		}
		switch block.Kind {
		case KindUser:
			b.WriteString(chatui.RenderUserLine(block.Body))
			b.WriteByte('\n')
			if s.Busy && i == len(s.Blocks)-1 {
				m.writeSegmentDivider(&b, width, segmentUser, segmentProcess)
				b.WriteString(chatui.RenderInitializing())
				b.WriteByte('\n')
			}
		case KindReply:
			body := strings.TrimRight(block.Body, "\n")
			if block.Live {
				b.WriteString(chatui.RenderAssistantBoxLive(body, width))
			} else {
				b.WriteString(chatui.RenderAssistantBox(body, width))
			}
			b.WriteByte('\n')
		default:
			prefix := "  "
			if i == s.Focus {
				prefix = styleFocus.Render("› ")
			}
			b.WriteString(prefix + renderSectionHeader(block, m.display))
			b.WriteByte('\n')
			if block.IsExpanded(m.display) {
				body := strings.TrimRight(block.Body, "\n")
				for _, line := range strings.Split(body, "\n") {
					if block.Kind == KindThinking {
						b.WriteString(chatui.RenderThinkingLine(line))
					} else {
						b.WriteString(chatui.RenderDetailLine(line))
					}
					b.WriteByte('\n')
				}
			} else if block.ShowThinkingPreview(m.display) {
				for _, line := range block.LastBodyLines(thinkingPreviewLines) {
					b.WriteString(chatui.RenderThinkingLine(TruncateRunes(line, width-4)))
					b.WriteByte('\n')
				}
			} else if block.ShowLivePreview(m.display) {
				line := TruncateRunes(block.LastBodyLine(), width-4)
				b.WriteString(chatui.RenderDetailLine(line))
				b.WriteByte('\n')
			}
		}
		prev = cur
		hasSegment = true
	}

	if s.Err != "" {
		if hasSegment {
			b.WriteString(chatui.RenderSoftDivider(width))
			b.WriteByte('\n')
		}
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(chatui.ColorErr)).Bold(true).Render("✗ " + s.Err))
		b.WriteByte('\n')
	}

	_ = EstimateTranscriptHeight(s.Blocks, m.display)
	return b.String()
}

func blockSegment(kind SectionKind) transcriptSegment {
	switch kind {
	case KindUser:
		return segmentUser
	case KindReply:
		return segmentReply
	default:
		return segmentProcess
	}
}

func (m *Model) writeSegmentDivider(b *strings.Builder, width int, prev, cur transcriptSegment) {
	if prev == segmentNone || cur == segmentNone {
		return
	}
	if cur == segmentUser {
		b.WriteString(chatui.RenderRule(width))
		b.WriteByte('\n')
		return
	}
	if prev == cur && cur == segmentProcess {
		b.WriteString(chatui.RenderSoftDivider(width))
		b.WriteByte('\n')
		return
	}
	switch {
	case prev == segmentUser && cur == segmentProcess,
		prev == segmentUser && cur == segmentReply,
		prev == segmentProcess && cur == segmentReply,
		prev == segmentReply && cur == segmentProcess:
		b.WriteString(chatui.RenderSoftDivider(width))
		b.WriteByte('\n')
	}
}

func renderSectionHeader(b Block, cfg config.DisplayConfig) string {
	n := b.LineCount()
	extra := ""
	if n > 0 {
		extra = fmt.Sprintf(" · %d 行", n)
	}
	if b.DurationSec > 0 {
		extra += fmt.Sprintf(" · %.1fs", b.DurationSec)
	}
	title := b.Title
	if title == "" {
		title = string(b.Kind)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(chatui.ColorAmber)).Bold(true).Render(
		fmt.Sprintf("%s %s%s", b.Chevron(cfg), title, extra),
	)
}

func renderInputLine(ti textinput.Model) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(chatui.ColorGold)).Bold(true).Render("❯ ") + ti.View()
}
