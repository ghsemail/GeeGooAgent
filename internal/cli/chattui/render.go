package chattui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

var (
	styleFocus = lipgloss.NewStyle().Foreground(lipgloss.Color(chatui.ColorAmber)).Bold(true)
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
	showBanner := m.banner != ""
	s := m.activeSlot()
	if s != nil {
		for _, block := range s.Blocks {
			if block.Kind == KindUser {
				showBanner = false
				break
			}
		}
	}
	if showBanner {
		b.WriteString(m.banner)
	}
	if s == nil {
		return b.String()
	}

	hasSegment := showBanner && m.banner != ""
	prev := segmentNone
	if showBanner && m.banner != "" {
		prev = segmentProcess
	}
	for i := 0; i < len(s.Blocks); i++ {
		block := s.Blocks[i]
		if !block.IsVisible(m.display) {
			continue
		}
		if block.Kind == KindReply {
			body := strings.TrimRight(block.Body, "\n")
			if body == "" && !block.Live {
				continue
			}
		}

		if IsProcessKind(block.Kind) {
			cur := segmentProcess
			if hasSegment {
				m.writeSegmentDivider(&b, width, prev, cur)
			}
			var panel strings.Builder
			m.appendProcessBlock(&panel, block, i, s.Focus, width)
			b.WriteString(panel.String())
			prev = cur
			hasSegment = true
			continue
		}

		cur := blockSegment(block.Kind)
		if hasSegment {
			m.writeSegmentDivider(&b, width, prev, cur)
		}
		switch block.Kind {
		case KindUser:
			b.WriteString(chatui.RenderUserPromptBox(block.Body, width))
			b.WriteByte('\n')
			if s.Busy && i == len(s.Blocks)-1 {
				b.WriteString(chatui.RenderWorkingLine())
				b.WriteByte('\n')
			}
		case KindReply:
			body := strings.TrimRight(block.Body, "\n")
			if block.Live && !m.display.StreamReplyEnabled() {
				continue
			}
			opts := chatui.AssistantRenderOptions{Markdown: m.display.ReplyMarkdownEnabled(), Live: block.Live}
			b.WriteString(chatui.RenderAssistantBoxWith(body, width, opts))
			b.WriteByte('\n')
			if !block.Live && block.DurationSec > 0 {
				b.WriteString(chatui.RenderTurnFooter(time.Duration(block.DurationSec * float64(time.Second))))
				b.WriteByte('\n')
			}
		}
		prev = cur
		hasSegment = true
	}

	if s.Err != "" {
		if hasSegment {
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
	if prev == segmentProcess && cur == segmentReply {
		b.WriteString(chatui.RenderAgentHeader(width))
		b.WriteByte('\n')
		return
	}
	switch {
	case prev == segmentUser && cur == segmentProcess,
		prev == segmentUser && cur == segmentReply:
		b.WriteString(chatui.RenderSoftDivider(width))
		b.WriteByte('\n')
	}
}

func renderInputLine(ti textinput.Model, model string, width int) string {
	return chatui.RenderInputChrome(ti.View(), model, width)
}
