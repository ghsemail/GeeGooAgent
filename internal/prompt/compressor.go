package prompt

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const clearedToolPlaceholder = "[Old tool output cleared to save context space]"

func clearOldToolResults(msgs []llm.Message, protectFrom, minChars int) []llm.Message {
	out := make([]llm.Message, len(msgs))
	copy(out, msgs)
	for i := 0; i < protectFrom && i < len(out); i++ {
		if out[i].Role == llm.RoleTool && len(out[i].Content) > minChars {
			out[i].Content = clearedToolPlaceholder
		}
	}
	return out
}

func alignBoundaryBackward(msgs []llm.Message, cut int) int {
	if cut <= 0 || cut >= len(msgs) {
		return cut
	}
	i := cut
	for i > 0 && msgs[i].Role == llm.RoleTool {
		i--
	}
	if i >= 0 && i < len(msgs) && msgs[i].Role == llm.RoleAssistant && len(msgs[i].ToolCalls) > 0 {
		return i
	}
	return cut
}

func sanitizeToolPairs(msgs []llm.Message) []llm.Message {
	need := map[string]bool{}
	have := map[string]bool{}
	for _, m := range msgs {
		for _, tc := range m.ToolCalls {
			if tc.ID != "" {
				need[tc.ID] = true
			}
		}
		if m.Role == llm.RoleTool && m.ToolCallID != "" {
			have[m.ToolCallID] = true
		}
	}
	var out []llm.Message
	for _, m := range msgs {
		if m.Role == llm.RoleTool {
			if m.ToolCallID == "" || !need[m.ToolCallID] {
				continue
			}
			out = append(out, m)
			continue
		}
		out = append(out, m)
		if m.Role == llm.RoleAssistant {
			for _, tc := range m.ToolCalls {
				if tc.ID != "" && !have[tc.ID] {
					out = append(out, llm.Message{
						Role: llm.RoleTool, ToolCallID: tc.ID,
						Content: `{"status":"skipped","summary":"[tool result omitted during compaction]"}`,
					})
					have[tc.ID] = true
				}
			}
		}
	}
	return out
}

func determineCut(msgs []llm.Message, cfg config.ResolvedCompression) (headEnd, cut int) {
	headEnd = cfg.ProtectFirstN
	if headEnd > len(msgs) {
		headEnd = len(msgs)
	}
	thresholdTokens := int(float64(cfg.ContextLength) * cfg.Threshold)
	budget := int(float64(thresholdTokens) * cfg.TargetRatio)
	acc := 0
	cut = len(msgs)
	for i := len(msgs) - 1; i >= headEnd; i-- {
		acc += EstimateTokens([]llm.Message{msgs[i]})
		cut = i
		if acc >= budget {
			break
		}
	}
	if len(msgs)-cut < cfg.ProtectLastN {
		cut = len(msgs) - cfg.ProtectLastN
		if cut < headEnd {
			cut = headEnd
		}
	}
	cut = alignBoundaryBackward(msgs, cut)
	if cut < headEnd {
		cut = headEnd
	}
	return headEnd, cut
}

type Compressor struct {
	cfg        config.ResolvedCompression
	summarizer Summarizer
}

func NewCompressor(cfg config.ResolvedCompression, sum Summarizer) *Compressor {
	return &Compressor{cfg: cfg, summarizer: sum}
}

func (c *Compressor) ShouldCompress(tokenEstimate, messageCount int) bool {
	if c == nil || !c.cfg.Enabled {
		return false
	}
	minMsgs := c.cfg.ProtectFirstN + c.cfg.ProtectLastN + 1
	if messageCount < minMsgs {
		return false
	}
	thresholdTokens := int(float64(c.cfg.ContextLength) * c.cfg.Threshold)
	return tokenEstimate >= thresholdTokens
}

// Compress returns (messages, didCompress, newSummary, err).
// On summarizer failure: returns original messages, did=false, err=nil.
func (c *Compressor) Compress(ctx context.Context, messages []llm.Message, previousSummary string, tokenEstimate int) ([]llm.Message, bool, string, error) {
	if c == nil || !c.ShouldCompress(tokenEstimate, len(messages)) {
		return messages, false, previousSummary, nil
	}
	headEnd, cut := determineCut(messages, c.cfg)
	if cut <= headEnd {
		return messages, false, previousSummary, nil
	}
	working := clearOldToolResults(messages, cut, c.cfg.ClearToolMinChars)
	head := working[:headEnd]
	middle := working[headEnd:cut]
	tail := working[cut:]

	contentTokens := EstimateTokens(middle)
	maxTok := contentTokens * 20 / 100
	if maxTok < 2000 {
		maxTok = 2000
	}
	capTok := c.cfg.ContextLength * 5 / 100
	if capTok > 12000 {
		capTok = 12000
	}
	if maxTok > capTok {
		maxTok = capTok
	}

	if c.summarizer == nil {
		return messages, false, previousSummary, nil
	}
	summary, err := c.summarizer.Summarize(ctx, middle, previousSummary, maxTok)
	if err != nil {
		return messages, false, previousSummary, nil
	}

	summaryMsg := llm.Message{
		Role:    llm.RoleUser,
		Content: "[CONTEXT COMPACTION] Earlier turns were compacted into the following summary:\n" + summary,
	}
	if len(head) > 0 && head[len(head)-1].Role == llm.RoleUser {
		summaryMsg.Role = llm.RoleAssistant
	}

	out := make([]llm.Message, 0, len(head)+1+len(tail))
	out = append(out, head...)
	out = append(out, summaryMsg)
	out = append(out, tail...)
	out = sanitizeToolPairs(out)
	return out, true, summary, nil
}
