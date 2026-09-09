package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/opslog"
	"github.com/ghsemail/GeeGooAgent/internal/stockdigest"
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
	var deliveryFailures []string
	for _, user := range users {
		userToken := strings.TrimSpace(user.MCPToken)
		if userToken == "" {
			continue
		}
		userOpts := baseOpts
		userOpts.MCPToken = userToken
		result, runErr := a.runSkillWithSteps(ctx, skill, phaseA, perStock, userOpts)
		last = result
		userID := strings.TrimSpace(user.UserID)
		reportDate := opts.ReportDate
		if result.Working != nil && strings.TrimSpace(result.Working.ReportDate) != "" {
			reportDate = strings.TrimSpace(result.Working.ReportDate)
		}
		rec := opslog.NewRunRecorder(skill, market, userID, result.SessionID, reportDate)

		feishuSent := false
		feishuSkipReason := ""
		if runErr != nil {
			feishuSkipReason = "workflow_error"
		} else if opts.NotifyFeishu && userID != "" {
			feishuSkipReason = stockdigest.NotifySkipReason(skill, market, result)
			if feishuSkipReason == "" {
				feishuSent = a.maybeNotifyUserStockFeishu(ctx, userID, skill, market, result)
				if !feishuSent {
					feishuSkipReason = "feishu_send_failed"
				}
			}
		} else if opts.NotifyFeishu {
			feishuSkipReason = "missing_user_id"
		} else {
			feishuSkipReason = "notify_disabled"
		}
		a.persistReportGenerationLog(ctx, rec, result, feishuSent, feishuSkipReason)

		ok, reason := userStockDeliveryOK(skill, market, result, runErr, opts.NotifyFeishu, feishuSent, feishuSkipReason)
		if !ok {
			if userID == "" {
				userID = "unknown"
			}
			deliveryFailures = append(deliveryFailures, fmt.Sprintf("%s:%s", userID, reason))
		}
	}
	if len(deliveryFailures) > 0 {
		return last, fmt.Errorf("report delivery failed for %d user(s): %s", len(deliveryFailures), strings.Join(deliveryFailures, "; "))
	}
	return last, nil
}
