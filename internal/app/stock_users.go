package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func (a *App) runStockForMarketUsers(
	ctx context.Context,
	skill, market string,
	opts SkillRunOptions,
	phaseA, perStock []workflow.Step,
) (workflow.RunResult, error) {
	if a == nil || a.MCP == nil {
		return workflow.RunResult{}, fmt.Errorf("mcp client not configured")
	}
	token := strings.TrimSpace(opts.MCPToken)
	if token == "" && a.Config != nil {
		token = strings.TrimSpace(a.Config.MCPToken())
	}
	if token == "" {
		return workflow.RunResult{}, fmt.Errorf("mcp_token required to list report users")
	}
	baseOpts := SkillRunOptions{Market: market, MCPToken: token, ReportDate: opts.ReportDate, NotifyFeishu: opts.NotifyFeishu}
	users, err := a.MCP.ListReportUsers(ctx, token, market)
	if err != nil || len(users) == 0 {
		return a.runSkillWithSteps(ctx, skill, phaseA, perStock, baseOpts)
	}
	var last workflow.RunResult
	var lastErr error
	for _, user := range users {
		userToken := strings.TrimSpace(user.MCPToken)
		if userToken == "" {
			continue
		}
		userOpts := baseOpts
		userOpts.MCPToken = userToken
		result, runErr := a.runSkillWithSteps(ctx, skill, phaseA, perStock, userOpts)
		last = result
		lastErr = runErr
		if runErr != nil {
			continue
		}
		if opts.NotifyFeishu {
			userID := strings.TrimSpace(user.UserID)
			if userID != "" {
				a.maybeNotifyUserStockFeishu(ctx, userID, skill, market, result)
			}
		}
	}
	return last, lastErr
}
