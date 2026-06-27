package memory

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

var (
	preMarketIndexCodes = map[string]struct{}{
		"^DJI.US": {}, "^IXIC.US": {}, "000001.SH": {}, "399001.SZ": {}, "800000.HK": {},
	}
	preMarketNewsMarkets = []string{"US", "CN", "HK"}
)

// WorkingStore persists and applies working memory updates.
type WorkingStore struct {
	store *infra.StateStore
}

// NewWorkingStore creates a working memory store.
func NewWorkingStore(store *infra.StateStore) *WorkingStore {
	return &WorkingStore{store: store}
}

func (s *WorkingStore) key(sessionID string) string {
	return "working/" + sessionID
}

// Create initializes working memory for a session.
func (s *WorkingStore) Create(sessionID, skill string) (*PreMarketWorking, error) {
	w := NewPreMarketWorking(sessionID, skill)
	return w, s.Save(w)
}

// Load reads working memory.
func (s *WorkingStore) Load(sessionID string) (*PreMarketWorking, error) {
	data, err := s.store.Load(s.key(sessionID))
	if err != nil || data == nil {
		return nil, err
	}
	return decodeWorking(data)
}

// Save persists working memory.
func (s *WorkingStore) Save(w *PreMarketWorking) error {
	return s.store.Save(s.key(w.SessionID), encodeWorking(w))
}

// Apply updates working memory after a tool result.
func (s *WorkingStore) Apply(w *PreMarketWorking, toolName string, result tools.Result) (*PreMarketWorking, error) {
	updated := cloneWorking(w)
	stepKey := fmt.Sprintf("%s:%s", toolName, result.Status)
	if !contains(updated.StepsCompleted, stepKey) {
		updated.StepsCompleted = append(updated.StepsCompleted, stepKey)
	}
	data := result.Data
	if data == nil {
		data = map[string]any{}
	}

	switch toolName {
	case "check_trading_day":
		if v, ok := data["is_trading_day"].(bool); ok {
			updated.IsTradingDay = &v
			if v {
				updated.Phase = "phase_a"
			} else {
				updated.Phase = "done"
			}
		}
	case "get_report_bot_codes":
		if bots, ok := data["bots"].([]any); ok {
			updated.BotCodes = nil
			for _, item := range bots {
				if m, ok := item.(map[string]any); ok {
					bot := botFromMap(m)
					updated.BotCodes = append(updated.BotCodes, bot)
					if _, exists := updated.Stocks[bot.Code]; !exists {
						updated.Stocks[bot.Code] = StockWorkspace{
							Code: bot.Code, StockName: bot.StockName,
							BotID: bot.BotID, BotName: bot.BotName, BotType: bot.BotType,
							Status: "pending",
						}
					}
				}
			}
		}
	case "get_mcp_analysis":
		code, _ := data["code"].(string)
		period, _ := data["period"].(string)
		analysis, _ := data["analysis_result"].(string)
		if _, isIndex := preMarketIndexCodes[code]; isIndex {
			updated.MarketContext.IndexAnalysisRefs[code] = truncate(analysis, 2000)
			if !contains(updated.MarketContext.IndexCodesDone, code) {
				updated.MarketContext.IndexCodesDone = append(updated.MarketContext.IndexCodesDone, code)
			}
			if len(updated.MarketContext.IndexCodesDone) >= 5 {
				updated.MarketContext.IndicesDone = true
			}
		} else if ws, ok := updated.Stocks[code]; ok && period == "weekly" {
			ws.WeeklyAnalysisRef = truncate(analysis, 2000)
			updated.Stocks[code] = ws
		}
	case "fetch_stock_news":
		code, _ := data["code"].(string)
		text, _ := data["text"].(string)
		if ws, ok := updated.Stocks[code]; ok {
			ws.StockNewsSummary = truncate(text, 2000)
			updated.Stocks[code] = ws
		}
	case "get_capital_flow":
		code, _ := data["code"].(string)
		if ws, ok := updated.Stocks[code]; ok {
			if result.Status == tools.StatusSkip && isAShare(code) {
				ws.CapitalFlowSummary = ""
			} else if result.Status == tools.StatusSkip {
				reason, _ := data["skip_reason"].(string)
				ws.CapitalFlowSummary = truncate(reason, 2000)
			} else if latest, ok := data["latest"].(map[string]any); ok {
				ws.CapitalFlowSummary = fmt.Sprintf("main_in_flow=%v", latest["main_in_flow"])
			}
			updated.Stocks[code] = ws
		}
	case "get_capital_distribution":
		code, _ := data["code"].(string)
		if ws, ok := updated.Stocks[code]; ok {
			if result.Status == tools.StatusSkip && isAShare(code) {
				ws.CapitalDistributionSummary = ""
			} else {
				formatted, _ := data["formatted"].(string)
				ws.CapitalDistributionSummary = truncate(formatted, 2000)
				if ws.CapitalDistributionSummary == "" {
					ws.CapitalDistributionSummary = truncate(result.Summary, 2000)
				}
			}
			updated.Stocks[code] = ws
		}
	case "get_bot_yesterday_attitude":
		code, _ := data["code"].(string)
		if code == "" {
			code = updated.CurrentStock
		}
		if ws, ok := updated.Stocks[code]; ok {
			att, _ := data["attitude"].(string)
			ws.Attitude = att
			updated.Stocks[code] = ws
		}
	case "list_today_reports":
		code, _ := data["code"].(string)
		if reported, _ := data["already_reported"].(bool); reported {
			if ws, ok := updated.Stocks[code]; ok {
				ws.Status = "skipped"
				updated.Stocks[code] = ws
			}
		}
	case "save_local_report":
		code, _ := data["code"].(string)
		path, _ := data["path"].(string)
		if ws, ok := updated.Stocks[code]; ok && path != "" {
			ws.ReportRef = path
			updated.Stocks[code] = ws
		}
	case "create_pre_market_report":
		code, _ := data["code"].(string)
		if ws, ok := updated.Stocks[code]; ok {
			ws.Status = "reported"
			if id, _ := data["report_id"].(string); id != "" {
				ws.ReportID = id
			}
			updated.Stocks[code] = ws
		}
	case "fetch_market_news":
		market, _ := data["market"].(string)
		text, _ := data["text"].(string)
		if market != "" {
			updated.MarketContext.MarketNews[market] = truncate(text, 2000)
		}
		if hasAllMarkets(updated.MarketContext.MarketNews) {
			updated.MarketContext.MarketNewsDone = true
		}
	case "write_execution_log":
		if path, _ := data["path"].(string); path != "" {
			updated.Artifacts["execution_log"] = path
		}
	}

	if err := s.Save(updated); err != nil {
		return w, err
	}
	return updated, nil
}

func hasAllMarkets(news map[string]string) bool {
	for _, m := range preMarketNewsMarkets {
		if _, ok := news[m]; !ok {
			return false
		}
	}
	return true
}

func isAShare(code string) bool {
	return strings.HasSuffix(code, ".SH") || strings.HasSuffix(code, ".SZ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func botFromMap(m map[string]any) BotStock {
	return BotStock{
		Code:      str(m, "code"),
		StockName: str(m, "stock_name"),
		BotID:     str(m, "bot_id"),
		BotName:   str(m, "bot_name"),
		BotType:   str(m, "bot_type"),
	}
}

func str(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func cloneWorking(w *PreMarketWorking) *PreMarketWorking {
	c := *w
	c.BotCodes = append([]BotStock(nil), w.BotCodes...)
	c.StepsCompleted = append([]string(nil), w.StepsCompleted...)
	c.MarketContext.IndexCodesDone = append([]string(nil), w.MarketContext.IndexCodesDone...)
	c.MarketContext.IndexAnalysisRefs = map[string]string{}
	for k, v := range w.MarketContext.IndexAnalysisRefs {
		c.MarketContext.IndexAnalysisRefs[k] = v
	}
	c.MarketContext.MarketNews = map[string]string{}
	for k, v := range w.MarketContext.MarketNews {
		c.MarketContext.MarketNews[k] = v
	}
	c.Stocks = map[string]StockWorkspace{}
	for k, v := range w.Stocks {
		c.Stocks[k] = v
	}
	c.Artifacts = map[string]string{}
	for k, v := range w.Artifacts {
		c.Artifacts[k] = v
	}
	return &c
}

// encode/decode via JSON round-trip for simplicity
func encodeWorking(w *PreMarketWorking) map[string]any {
	m := map[string]any{
		"session_id": w.SessionID, "skill": w.Skill, "phase": w.Phase,
		"current_stock_code": w.CurrentStock,
		"steps_completed": w.StepsCompleted, "artifacts": w.Artifacts,
	}
	if w.IsTradingDay != nil {
		m["is_trading_day"] = *w.IsTradingDay
	}
	bots := make([]map[string]any, 0, len(w.BotCodes))
	for _, b := range w.BotCodes {
		bots = append(bots, map[string]any{
			"code": b.Code, "stock_name": b.StockName, "bot_id": b.BotID,
			"bot_name": b.BotName, "bot_type": b.BotType,
		})
	}
	m["bot_codes"] = bots
	m["market_context"] = map[string]any{
		"indices_done": w.MarketContext.IndicesDone, "market_news_done": w.MarketContext.MarketNewsDone,
		"index_analysis_refs": w.MarketContext.IndexAnalysisRefs,
		"index_codes_done": w.MarketContext.IndexCodesDone, "market_news": w.MarketContext.MarketNews,
	}
	stocks := map[string]any{}
	for k, v := range w.Stocks {
		stocks[k] = map[string]any{
			"code": v.Code, "stock_name": v.StockName, "bot_id": v.BotID,
			"bot_name": v.BotName, "bot_type": v.BotType, "status": v.Status,
			"weekly_analysis_ref": v.WeeklyAnalysisRef, "attitude": v.Attitude,
			"capital_flow_summary": v.CapitalFlowSummary,
			"capital_distribution_summary": v.CapitalDistributionSummary,
			"report_ref": v.ReportRef, "report_id": v.ReportID,
			"stock_news_summary": v.StockNewsSummary,
		}
	}
	m["stocks"] = stocks
	return m
}

func decodeWorking(data map[string]any) (*PreMarketWorking, error) {
	w := NewPreMarketWorking(stringField(data, "session_id"), stringField(data, "skill"))
	w.Phase = stringField(data, "phase")
	w.CurrentStock = stringField(data, "current_stock_code")
	if v, ok := data["is_trading_day"].(bool); ok {
		w.IsTradingDay = &v
	}
	if steps, ok := data["steps_completed"].([]any); ok {
		for _, s := range steps {
			if str, ok := s.(string); ok {
				w.StepsCompleted = append(w.StepsCompleted, str)
			}
		}
	}
	if arts, ok := data["artifacts"].(map[string]any); ok {
		for k, v := range arts {
			if s, ok := v.(string); ok {
				w.Artifacts[k] = s
			}
		}
	}
	if bots, ok := data["bot_codes"].([]any); ok {
		for _, item := range bots {
			if m, ok := item.(map[string]any); ok {
				w.BotCodes = append(w.BotCodes, botFromMap(m))
			}
		}
	}
	if mc, ok := data["market_context"].(map[string]any); ok {
		if v, ok := mc["indices_done"].(bool); ok {
			w.MarketContext.IndicesDone = v
		}
		if v, ok := mc["market_news_done"].(bool); ok {
			w.MarketContext.MarketNewsDone = v
		}
		if refs, ok := mc["index_analysis_refs"].(map[string]any); ok {
			for k, v := range refs {
				if s, ok := v.(string); ok {
					w.MarketContext.IndexAnalysisRefs[k] = s
				}
			}
		}
		if done, ok := mc["index_codes_done"].([]any); ok {
			for _, item := range done {
				if s, ok := item.(string); ok {
					w.MarketContext.IndexCodesDone = append(w.MarketContext.IndexCodesDone, s)
				}
			}
		}
		if news, ok := mc["market_news"].(map[string]any); ok {
			for k, v := range news {
				if s, ok := v.(string); ok {
					w.MarketContext.MarketNews[k] = s
				}
			}
		}
	}
	if stocks, ok := data["stocks"].(map[string]any); ok {
		for code, raw := range stocks {
			if m, ok := raw.(map[string]any); ok {
				w.Stocks[code] = StockWorkspace{
					Code: code, StockName: str(m, "stock_name"), BotID: str(m, "bot_id"),
					BotName: str(m, "bot_name"), BotType: str(m, "bot_type"),
					Status: strDefault(m, "status", "pending"),
					WeeklyAnalysisRef: str(m, "weekly_analysis_ref"), Attitude: str(m, "attitude"),
					CapitalFlowSummary: str(m, "capital_flow_summary"),
					CapitalDistributionSummary: str(m, "capital_distribution_summary"),
					ReportRef: str(m, "report_ref"), ReportID: str(m, "report_id"),
					StockNewsSummary: str(m, "stock_news_summary"),
				}
			}
		}
	}
	return w, nil
}

func stringField(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func strDefault(m map[string]any, k, def string) string {
	if v := str(m, k); v != "" {
		return v
	}
	return def
}
