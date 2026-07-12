package chattui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// ProgressMsg is sent from Agent progress callbacks into the Bubble Tea loop.
type ProgressMsg struct {
	Event string
	Data  map[string]any
}

// ApplyProgress mutates the model transcript from a progress event.
func (m *Model) ApplyProgress(event string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	switch event {
	case "turn_start":
		m.busy = true
		m.status = "thinking…"
		m.liveThinkingID = ""
		m.liveToolsID = ""
		m.liveReplyID = ""
	case "stream_delta":
		content, _ := data["content"].(string)
		if strings.TrimSpace(content) == "" {
			return
		}
		m.ensureLiveReply()
		idx := m.blockIndex(m.liveReplyID)
		if idx >= 0 {
			m.blocks[idx].Body += content
			m.blocks[idx].Live = true
		}
	case "llm_plan":
		reasoning, _ := data["reasoning"].(string)
		if strings.TrimSpace(reasoning) == "" {
			return
		}
		m.finalizeLiveThinking()
		id := fmt.Sprintf("think-%d", m.seq)
		m.seq++
		m.blocks = append(m.blocks, Block{
			ID: id, Kind: KindThinking, Title: "💭 思考",
			Body: reasoning, Live: true,
		})
		m.liveThinkingID = id
		m.focus = len(m.blocks) - 1
	case "llm_tools", "tool_start":
		name, _ := data["name"].(string)
		if name == "" {
			if names, ok := data["tool_names"].([]string); ok && len(names) > 0 {
				name = strings.Join(names, ", ")
			}
		}
		if name == "" {
			return
		}
		m.ensureLiveTools()
		idx := m.blockIndex(m.liveToolsID)
		if idx >= 0 {
			line := "→ " + name
			if m.blocks[idx].Body != "" {
				m.blocks[idx].Body += "\n" + line
			} else {
				m.blocks[idx].Body = line
			}
			m.blocks[idx].Live = true
			m.blocks[idx].Title = "🔧 工具"
		}
		m.status = "running…"
	case "tool_done":
		name, _ := data["name"].(string)
		status, _ := data["status"].(string)
		summary, _ := data["summary"].(string)
		m.ensureLiveTools()
		idx := m.blockIndex(m.liveToolsID)
		if idx >= 0 {
			line := fmt.Sprintf("✓ %s [%s] %s", name, status, TruncateRunes(summary, 120))
			if m.blocks[idx].Body != "" {
				m.blocks[idx].Body += "\n" + line
			} else {
				m.blocks[idx].Body = line
			}
		}
	case "error":
		msg, _ := data["message"].(string)
		m.info = msg
		m.busy = false
		m.status = "error"
	}
}

func (m *Model) ensureLiveReply() {
	if m.liveReplyID != "" {
		return
	}
	id := fmt.Sprintf("reply-%d", m.seq)
	m.seq++
	m.blocks = append(m.blocks, Block{ID: id, Kind: KindReply, Title: "助手", Live: true})
	m.liveReplyID = id
}

func (m *Model) ensureLiveTools() {
	if m.liveToolsID != "" {
		return
	}
	id := fmt.Sprintf("tools-%d", m.seq)
	m.seq++
	m.blocks = append(m.blocks, Block{ID: id, Kind: KindTools, Title: "🔧 工具", Live: true})
	m.liveToolsID = id
	m.focus = len(m.blocks) - 1
}

func (m *Model) finalizeLiveThinking() {
	if m.liveThinkingID == "" {
		return
	}
	if idx := m.blockIndex(m.liveThinkingID); idx >= 0 {
		m.blocks[idx].Live = false
	}
	m.liveThinkingID = ""
}

func (m *Model) finalizeLiveSections() {
	for _, id := range []string{m.liveThinkingID, m.liveToolsID, m.liveReplyID} {
		if idx := m.blockIndex(id); idx >= 0 {
			m.blocks[idx].Live = false
		}
	}
	m.liveThinkingID, m.liveToolsID, m.liveReplyID = "", "", ""
	m.busy = false
	m.status = "ready"
	m.turnEnded = time.Now()
}

func (m *Model) blockIndex(id string) int {
	if id == "" {
		return -1
	}
	for i := range m.blocks {
		if m.blocks[i].ID == id {
			return i
		}
	}
	return -1
}

func (m *Model) expandLastDetails() {
	for i := len(m.blocks) - 1; i >= 0; i-- {
		k := m.blocks[i].Kind
		if k == KindThinking || k == KindTools {
			yes := true
			m.blocks[i].UserExpanded = &yes
			m.blocks[i].Live = false
			m.focus = i
			if k == KindThinking {
				break
			}
		}
	}
}

func headerLabel(b Block, cfg config.DisplayConfig) string {
	n := b.LineCount()
	extra := ""
	if n > 0 {
		extra = fmt.Sprintf(" · %d 行", n)
	}
	if b.DurationSec > 0 {
		extra += fmt.Sprintf(" · %.1fs", b.DurationSec)
	}
	return fmt.Sprintf("%s %s%s", b.Chevron(cfg), b.Title, extra)
}
