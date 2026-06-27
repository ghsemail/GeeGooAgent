package chatsession

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

var priceInSummary = regexp.MustCompile(`price=([0-9.]+)`)

var (
	priceTools  = map[string]struct{}{"get_current_price": {}, "get_ticker": {}}
	searchTools = map[string]struct{}{"search_code": {}}
)

// StockEvent is one price/search event extracted from a chat session.
type StockEvent struct {
	Code    string
	Query   string
	Price   *float64
	Tool    string
	Summary string
}

// SessionRecallHit is one matching past session.
type SessionRecallHit struct {
	SessionID   string
	UpdatedAt   string
	Score       int
	UserQueries []string
	StockEvents []StockEvent
	Snippet     string
}

// ExtractStockEvents pulls price/search events from persisted messages.
func ExtractStockEvents(session *ChatSession) []StockEvent {
	var events []StockEvent
	pending := map[string]struct {
		name string
		args map[string]any
	}{}

	for _, msg := range session.Messages {
		switch msg.Role {
		case llm.RoleAssistant:
			for _, call := range msg.ToolCalls {
				if _, ok := priceTools[call.Name]; ok {
					pending[call.ID] = struct {
						name string
						args map[string]any
					}{name: call.Name, args: call.Arguments}
				}
				if _, ok := searchTools[call.Name]; ok {
					pending[call.ID] = struct {
						name string
						args map[string]any
					}{name: call.Name, args: call.Arguments}
				}
			}
		case llm.RoleTool:
			item, ok := pending[msg.ToolCallID]
			if !ok {
				continue
			}
			delete(pending, msg.ToolCallID)
			payload := parseToolContent(msg.Content)
			data, _ := payload["data"].(map[string]any)
			if data == nil {
				data = payload
			}
			summary := fmt.Sprint(payload["summary"])
			var code, query string
			var price *float64
			if data != nil {
				if v, ok := data["code"].(string); ok && v != "" {
					code = v
				}
				if code == "" {
					if v, ok := item.args["code"].(string); ok {
						code = v
					}
				}
				if p := floatFromAny(data["price"]); p != nil {
					price = p
				}
			}
			if item.name == "search_code" {
				query = strFromAny(item.args["regex"])
				if query == "" {
					query = strFromAny(item.args["query"])
				}
			}
			if price == nil {
				if m := priceInSummary.FindStringSubmatch(summary); len(m) == 2 {
					if p := parseFloat(m[1]); p != nil {
						price = p
					}
				}
			}
			events = append(events, StockEvent{
				Code: code, Query: query, Price: price, Tool: item.name, Summary: truncateRunes(summary, 200),
			})
		}
	}
	return events
}

// SearchPastSessions finds recent chat sessions with stock/price activity.
func SearchPastSessions(store *ChatSessionStore, query, excludeSessionID string, limit, scanLimit int) ([]SessionRecallHit, error) {
	if store == nil {
		return nil, nil
	}
	ids, err := store.ListSessionIDs()
	if err != nil {
		return nil, err
	}
	var sessions []*ChatSession
	for _, sessionID := range ids {
		if sessionID == excludeSessionID {
			continue
		}
		loaded, err := store.Load(sessionID)
		if err != nil || loaded == nil || len(loaded.Messages) <= 1 {
			continue
		}
		sessions = append(sessions, loaded)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})
	if scanLimit > 0 && len(sessions) > scanLimit {
		sessions = sessions[:scanLimit]
	}

	var hits []SessionRecallHit
	for _, session := range sessions {
		events := ExtractStockEvents(session)
		queries := userQueries(session)
		corpus := sessionCorpus(session, events, queries)
		if len(events) == 0 && len(queries) == 0 {
			continue
		}
		score := scoreQuery(corpus, query)
		if strings.TrimSpace(query) != "" {
			if score <= 0 {
				continue
			}
		} else if len(events) == 0 {
			continue
		} else if score <= 0 {
			score = 1
		}
		hits = append(hits, SessionRecallHit{
			SessionID: session.ID, UpdatedAt: session.UpdatedAt.Format(time.RFC3339),
			Score: score, UserQueries: tailStrings(queries, 5),
			StockEvents: events, Snippet: buildSnippet(session, events, queries),
		})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].UpdatedAt > hits[j].UpdatedAt
	})
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

// HitsToData converts recall hits to tool result payload.
func HitsToData(hits []SessionRecallHit) map[string]any {
	matches := make([]map[string]any, 0, len(hits))
	for _, hit := range hits {
		events := make([]map[string]any, 0, len(hit.StockEvents))
		for _, e := range hit.StockEvents {
			item := map[string]any{"code": e.Code, "query": e.Query, "tool": e.Tool, "summary": e.Summary}
			if e.Price != nil {
				item["price"] = *e.Price
			}
			events = append(events, item)
		}
		matches = append(matches, map[string]any{
			"session_id": hit.SessionID, "updated_at": hit.UpdatedAt, "score": hit.Score,
			"snippet": hit.Snippet, "user_queries": hit.UserQueries, "stock_events": events,
		})
	}
	return map[string]any{"count": len(hits), "matches": matches}
}

func parseToolContent(content string) map[string]any {
	if strings.TrimSpace(content) == "" {
		return map[string]any{}
	}
	var payload map[string]any
	if json.Unmarshal([]byte(content), &payload) != nil {
		return map[string]any{}
	}
	return payload
}

func userQueries(session *ChatSession) []string {
	var queries []string
	for _, msg := range session.Messages {
		if msg.Role != llm.RoleUser || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		text := strings.TrimSpace(msg.Content)
		if strings.HasPrefix(text, "/") {
			continue
		}
		queries = append(queries, text)
	}
	return queries
}

func sessionCorpus(session *ChatSession, events []StockEvent, queries []string) string {
	parts := append([]string{}, queries...)
	for _, event := range events {
		if event.Code != "" {
			parts = append(parts, event.Code)
		}
		if event.Query != "" {
			parts = append(parts, event.Query)
		}
		if event.Summary != "" {
			parts = append(parts, event.Summary)
		}
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func buildSnippet(session *ChatSession, events []StockEvent, queries []string) string {
	var priced []StockEvent
	for _, e := range events {
		if e.Code != "" {
			if _, ok := priceTools[e.Tool]; ok {
				priced = append(priced, e)
			}
		}
	}
	if len(priced) > 0 {
		last := priced[len(priced)-1]
		pricePart := ""
		if last.Price != nil {
			pricePart = fmt.Sprintf(" price=%v", *last.Price)
		}
		return fmt.Sprintf("查价 %s%s", last.Code, pricePart)
	}
	if len(queries) > 0 {
		return truncateRunes(queries[len(queries)-1], 120)
	}
	if len(events) > 0 && events[len(events)-1].Query != "" {
		return "搜索 " + events[len(events)-1].Query
	}
	return "(no stock activity)"
}

func scoreQuery(corpus, query string) int {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		if strings.TrimSpace(corpus) == "" {
			return 0
		}
		return 1
	}
	score := 0
	if strings.Contains(corpus, q) {
		score += 3
	}
	for _, token := range strings.Fields(q) {
		if len(token) >= 2 && strings.Contains(corpus, token) {
			score++
		}
	}
	for _, hint := range []string{"股价", "价格", "查", "股票", "腾讯", "茅台", "price"} {
		if strings.Contains(q, hint) && strings.Contains(corpus, hint) {
			score++
		}
	}
	return score
}

func tailStrings(items []string, n int) []string {
	if len(items) <= n {
		return items
	}
	return items[len(items)-n:]
}

func truncateRunes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func strFromAny(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func floatFromAny(v any) *float64 {
	switch t := v.(type) {
	case float64:
		return &t
	case int:
		f := float64(t)
		return &f
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return &f
		}
	}
	return nil
}

func parseFloat(s string) *float64 {
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err != nil {
		return nil
	}
	return &f
}
