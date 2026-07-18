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

	wroteContent := m.banner != ""
	for i, block := range s.Blocks {
		if !block.IsVisible(m.display) {
			continue
		}
		switch block.Kind {
		case KindUser:
			if wroteContent {
				b.WriteString(chatui.RenderRule(width))
				b.WriteByte('\n')
			}
			b.WriteString(chatui.RenderUserLine(block.Body))
			b.WriteByte('\n')
			wroteContent = true
			if s.Busy && i == len(s.Blocks)-1 {
				b.WriteString(chatui.RenderRule(width))
				b.WriteByte('\n')
				b.WriteString(chatui.RenderInitializing())
				b.WriteByte('\n')
			}
		case KindReply:
			body := strings.TrimRight(block.Body, "\n")
			if body == "" && !block.Live {
				continue
			}
			if wroteContent {
				b.WriteString(chatui.RenderRule(width))
				b.WriteByte('\n')
			}
			if block.Live {
				b.WriteString(chatui.RenderAssistantBoxLive(body, width))
			} else {
				b.WriteString(chatui.RenderAssistantBox(body, width))
			}
			b.WriteByte('\n')
			wroteContent = true
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
			} else if block.ShowLivePreview(m.display) {
				line := TruncateRunes(block.LastBodyLine(), width-4)
				if block.Kind == KindThinking {
					b.WriteString(chatui.RenderThinkingLine(line))
				} else {
					b.WriteString(chatui.RenderDetailLine(line))
				}
				b.WriteByte('\n')
			}
			wroteContent = true
		}
	}

	if s.Err != "" {
		b.WriteString(chatui.RenderRule(width))
		b.WriteByte('\n')
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(chatui.ColorErr)).Bold(true).Render("✗ " + s.Err))
		b.WriteByte('\n')
	}

	_ = EstimateTranscriptHeight(s.Blocks, m.display)
	return b.String()
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
