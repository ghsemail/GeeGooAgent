package chattui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// ProgressMsg is sent from Agent progress callbacks into the Bubble Tea loop.
type ProgressMsg struct {
	Slot  int
	Event string
	Data  map[string]any
}

// ApplyProgress mutates a live slot transcript from a progress event.
func (s *LiveSlot) ApplyProgress(event string, data map[string]any) {
	if s == nil {
		return
	}
	if data == nil {
		data = map[string]any{}
	}
	switch event {
	case "turn_start":
		s.Busy = true
		s.Status = "thinking…"
		s.TurnStartedAt = time.Now()
		s.TurnEndedAt = time.Time{}
		s.LiveThinkingID = ""
		s.LiveToolsID = ""
		s.LiveReplyID = ""
	case "stream_delta":
		content, _ := data["content"].(string)
		if strings.TrimSpace(content) == "" {
			return
		}
		s.ensureLiveReply()
		idx := s.blockIndex(s.LiveReplyID)
		if idx >= 0 {
			s.Blocks[idx].Body += content
			s.Blocks[idx].Live = true
		}
	case "llm_plan":
		reasoning, _ := data["reasoning"].(string)
		if strings.TrimSpace(reasoning) == "" {
			return
		}
		s.finalizeLiveThinking()
		id := fmt.Sprintf("think-%d", s.Seq)
		s.Seq++
		s.Blocks = append(s.Blocks, Block{
			ID: id, Kind: KindThinking, Title: "💭 思考",
			Body: reasoning, Live: true,
		})
		s.LiveThinkingID = id
		s.Focus = len(s.Blocks) - 1
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
		s.ensureLiveTools()
		idx := s.blockIndex(s.LiveToolsID)
		if idx >= 0 {
			line := "→ " + name
			if s.Blocks[idx].Body != "" {
				s.Blocks[idx].Body += "\n" + line
			} else {
				s.Blocks[idx].Body = line
			}
			s.Blocks[idx].Live = true
			s.Blocks[idx].Title = "🔧 工具"
		}
		s.Status = "running…"
	case "tool_done":
		name, _ := data["name"].(string)
		status, _ := data["status"].(string)
		summary, _ := data["summary"].(string)
		s.ensureLiveTools()
		idx := s.blockIndex(s.LiveToolsID)
		if idx >= 0 {
			line := fmt.Sprintf("✓ %s [%s] %s", name, status, TruncateRunes(summary, 120))
			if s.Blocks[idx].Body != "" {
				s.Blocks[idx].Body += "\n" + line
			} else {
				s.Blocks[idx].Body = line
			}
		}
	case "error":
		msg, _ := data["message"].(string)
		s.Err = msg
		s.Busy = false
		s.Status = "error"
	}
}

func (s *LiveSlot) ensureLiveReply() {
	if s.LiveReplyID != "" {
		return
	}
	id := fmt.Sprintf("reply-%d", s.Seq)
	s.Seq++
	s.Blocks = append(s.Blocks, Block{ID: id, Kind: KindReply, Title: "助手", Live: true})
	s.LiveReplyID = id
}

func (s *LiveSlot) ensureLiveTools() {
	if s.LiveToolsID != "" {
		return
	}
	id := fmt.Sprintf("tools-%d", s.Seq)
	s.Seq++
	s.Blocks = append(s.Blocks, Block{ID: id, Kind: KindTools, Title: "🔧 工具", Live: true})
	s.LiveToolsID = id
	s.Focus = len(s.Blocks) - 1
}

func (s *LiveSlot) finalizeLiveThinking() {
	if s.LiveThinkingID == "" {
		return
	}
	if idx := s.blockIndex(s.LiveThinkingID); idx >= 0 {
		s.Blocks[idx].Live = false
	}
	s.LiveThinkingID = ""
}

func (s *LiveSlot) finalizeLiveSections() {
	for _, id := range []string{s.LiveThinkingID, s.LiveToolsID, s.LiveReplyID} {
		if idx := s.blockIndex(id); idx >= 0 {
			s.Blocks[idx].Live = false
		}
	}
	s.LiveThinkingID, s.LiveToolsID, s.LiveReplyID = "", "", ""
	s.Busy = false
	s.Status = "ready"
	if !s.TurnStartedAt.IsZero() {
		s.TurnEndedAt = time.Now()
	}
}

func (s *LiveSlot) blockIndex(id string) int {
	if id == "" {
		return -1
	}
	for i := range s.Blocks {
		if s.Blocks[i].ID == id {
			return i
		}
	}
	return -1
}

func (s *LiveSlot) expandLastDetails() {
	for i := len(s.Blocks) - 1; i >= 0; i-- {
		k := s.Blocks[i].Kind
		if k == KindThinking || k == KindTools {
			yes := true
			s.Blocks[i].UserExpanded = &yes
			s.Blocks[i].Live = false
			s.Focus = i
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

// keep time import used for turn metadata elsewhere
var _ = time.Time{}
