package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

// NotifyStockFeishuForUsers sends stock digests to all report users for the market.
// Returns the number of users notified successfully.
func (a *App) NotifyStockFeishuForUsers(ctx context.Context, skill, market string, result workflow.RunResult) (int, error) {
	if a == nil || a.MCP == nil {
		return 0, fmt.Errorf("mcp client not configured")
	}
	token := ""
	if a.Config != nil {
		token = strings.TrimSpace(a.Config.MCPToken())
	}
	if token == "" {
		return 0, fmt.Errorf("mcp_token required to list report users")
	}
	users, err := a.MCP.ListReportUsers(ctx, token, market)
	if err != nil {
		return 0, err
	}
	if len(users) == 0 {
		return 0, fmt.Errorf("no report users for market %s", market)
	}
	sent := 0
	for _, user := range users {
		userID := strings.TrimSpace(user.UserID)
		if userID == "" {
			continue
		}
		if a.NotifyUserStockFeishu(ctx, userID, skill, market, result) {
			sent++
		}
	}
	return sent, nil
}
