package strategydev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// JevConfig holds System One credentials (token from catalog, never logged).
type JevConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

// JevClient calls TypeSafe System One (/v1/systemone).
type JevClient struct {
	cfg    JevConfig
	client *http.Client
}

func NewJevClient(cfg JevConfig) *JevClient {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = config.DefaultDecisionBaseURL
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = config.DefaultDecisionModel
	}
	return &JevClient{
		cfg: JevConfig{BaseURL: base, APIKey: strings.TrimSpace(cfg.APIKey), Model: model},
		client: &http.Client{Timeout: 45 * time.Second},
	}
}

type jevPickRequest struct {
	Task     string      `json:"task"`
	Regime   string      `json:"regime"`
	Side     string      `json:"side"`
	Freq     string      `json:"frequency"`
	Buy      []Candidate `json:"buy_candidates"`
	Sell     []Candidate `json:"sell_candidates"`
}

type jevPickResponse struct {
	BuySignalID  string `json:"buy_signal_id"`
	SellSignalID string `json:"sell_signal_id"`
	Reason       string `json:"reason"`
}

type jevVerdictRequest struct {
	Task           string          `json:"task"`
	Diagnose       DiagnoseSummary `json:"diagnose"`
	SelectedBuy    string          `json:"buy_name"`
	SelectedSell   string          `json:"sell_name"`
	HardBlockPass  bool            `json:"hard_block_pass"`
}

type jevVerdictResponse struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}

func (j *JevClient) PickStrategies(ctx context.Context, regime, side, freq string, buy, sell []Candidate) (buyID, sellID, reason string, err error) {
	if j == nil || j.cfg.APIKey == "" {
		return heuristicPick(regime, buy, sell)
	}
	payload, _ := json.Marshal(jevPickRequest{
		Task: "strategy_dev_pick", Regime: regime, Side: side, Freq: freq,
		Buy: slimCandidates(buy), Sell: slimCandidates(sell),
	})
	sys := `你是 GeeGoo 策略开发编排器（System-1）。只在给定候选中各选一个买入和一个卖出策略。
输出 JSON：{"buy_signal_id":"...","sell_signal_id":"...","reason":"一句中文"}
禁止选择 id 为空或 nosignal 的买入；下跌行情买入只能用于平空。`
	content, err := j.systemOne(ctx, sys, string(payload))
	if err != nil {
		return heuristicPick(regime, buy, sell)
	}
	var parsed jevPickResponse
	if err := json.Unmarshal([]byte(extractJSON(content)), &parsed); err != nil {
		return heuristicPick(regime, buy, sell)
	}
	return strings.TrimSpace(parsed.BuySignalID), strings.TrimSpace(parsed.SellSignalID), parsed.Reason, nil
}

func (j *JevClient) Verdict(ctx context.Context, in jevVerdictRequest) (string, string, error) {
	hardPass := ApplyHardVerdictRules(in.Diagnose) == VerdictPass
	in.HardBlockPass = !hardPass
	if j == nil || j.cfg.APIKey == "" {
		return ApplyHardVerdictRules(in.Diagnose), "规则判定", nil
	}
	payload, _ := json.Marshal(in)
	sys := `你是 GeeGoo 策略开发编排器（System-1）。根据诊断数字给出第六步结论。
只能输出 JSON：{"verdict":"通过"|"换一对"|"拉长窗口","reason":"一句中文"}
硬约束：complete_segments<3 或 avg_direction_return<=0 时 verdict 不能是「通过」。`
	content, err := j.systemOne(ctx, sys, string(payload))
	if err != nil {
		return ApplyHardVerdictRules(in.Diagnose), "Jev 不可用，使用规则判定", nil
	}
	var parsed jevVerdictResponse
	if err := json.Unmarshal([]byte(extractJSON(content)), &parsed); err != nil {
		return ApplyHardVerdictRules(in.Diagnose), "Jev 响应解析失败", nil
	}
	verdict := normalizeVerdict(parsed.Verdict)
	if verdict == VerdictPass && !hardPass {
		if in.Diagnose.CompleteSegs < 3 {
			return VerdictLongWindow, "完成段不足 3", nil
		}
		return VerdictSwapPair, "平均方向收益不为正", nil
	}
	return verdict, parsed.Reason, nil
}

type jevChoiceAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    float64            `json:"confidence"`
}

// Choice calls System One choice primitive (state + typed questions).
func (j *JevClient) Choice(
	ctx context.Context,
	state string,
	questionID, instructions string,
	criteria map[string]string,
) (choice string, probabilities map[string]float64, confidence float64, model string, err error) {
	if j == nil || strings.TrimSpace(j.cfg.APIKey) == "" {
		return "", nil, 0, "", fmt.Errorf("jev not configured")
	}
	qid := strings.TrimSpace(questionID)
	if qid == "" {
		qid = "direction"
	}
	body := map[string]any{
		"model": j.cfg.Model,
		"state": strings.TrimSpace(state),
		"questions": map[string]any{
			qid: map[string]any{
				"type":         "choice",
				"instructions": strings.TrimSpace(instructions),
				"criteria":     criteria,
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", nil, 0, "", err
	}
	url := j.cfg.BaseURL + "/systemone"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", nil, 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+j.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := j.client.Do(req)
	if err != nil {
		return "", nil, 0, "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, 0, "", err
	}
	if resp.StatusCode >= 400 {
		return "", nil, 0, "", fmt.Errorf("systemone http %d", resp.StatusCode)
	}
	var envelope struct {
		Model   string                     `json:"model"`
		Answers map[string]jevChoiceAnswer `json:"answers"`
	}
	if err := json.Unmarshal(b, &envelope); err != nil {
		return "", nil, 0, "", err
	}
	ans, ok := envelope.Answers[qid]
	if !ok {
		return "", nil, 0, "", fmt.Errorf("systemone missing answer %q", qid)
	}
	model = strings.TrimSpace(envelope.Model)
	if model == "" {
		model = j.cfg.Model
	}
	return strings.TrimSpace(ans.Choice), ans.Probabilities, ans.Confidence, model, nil
}

func (j *JevClient) systemOne(ctx context.Context, system, user string) (string, error) {
	url := j.cfg.BaseURL + "/systemone"
	body := map[string]any{
		"model": j.cfg.Model,
		"messages": []map[string]any{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.05,
		"max_tokens":  400,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+j.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := j.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("systemone http %d", resp.StatusCode)
	}
	modelHeader := resp.Header.Get("X-Model")
	if modelHeader != "" && !strings.Contains(modelHeader, "jev") {
		// response should be signed jev-1.13.0; non-fatal if header absent
	}
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(b, &envelope); err != nil {
		return "", err
	}
	if len(envelope.Choices) == 0 {
		return "", fmt.Errorf("empty systemone response")
	}
	return envelope.Choices[0].Message.Content, nil
}

func slimCandidates(in []Candidate) []Candidate {
	out := make([]Candidate, 0, len(in))
	for _, c := range in {
		out = append(out, Candidate{SignalID: c.SignalID, Name: c.Name, Kind: c.Kind})
	}
	return out
}

func heuristicPick(regime string, buy, sell []Candidate) (string, string, string, error) {
	if len(sell) == 0 {
		return "", "", "", fmt.Errorf("无卖出候选")
	}
	sellID := sell[0].SignalID
	var buyID string
	for _, c := range buy {
		if !isNoSignalRules(c.Buy) {
			buyID = c.SignalID
			break
		}
	}
	if buyID == "" && (regime == RegimeDown || regime == RegimeWeakDown) {
		return "", sellID, "", fmt.Errorf("下跌行情缺少合法平空买入")
	}
	return buyID, sellID, "启发式选择", nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func normalizeVerdict(v string) string {
	v = strings.TrimSpace(v)
	switch v {
	case VerdictPass, VerdictSwapPair, VerdictLongWindow:
		return v
	case "pass", "PASS":
		return VerdictPass
	case "swap", "换策略":
		return VerdictSwapPair
	case "extend", "extend_window":
		return VerdictLongWindow
	default:
		return ApplyHardVerdictRules(DiagnoseSummary{})
	}
}
