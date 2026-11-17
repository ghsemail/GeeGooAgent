package strategydev

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
)

// MCPMacroTradingDayChecker uses GeeGooBot POST /checkTradingDay with calendar date.
type MCPMacroTradingDayChecker struct {
	Client *mcp.Client
	Token  string
}

func NewMCPMacroTradingDayChecker(client *mcp.Client, token string) *MCPMacroTradingDayChecker {
	return &MCPMacroTradingDayChecker{Client: client, Token: strings.TrimSpace(token)}
}

func (c *MCPMacroTradingDayChecker) IsExchangeTradingDay(ctx context.Context, code, calendarDay string) (bool, error) {
	if c == nil || c.Client == nil || strings.TrimSpace(c.Token) == "" {
		return false, fmt.Errorf("mcp trading day checker not configured")
	}
	calendarDay = strings.TrimSpace(calendarDay)
	if calendarDay == "" {
		return false, fmt.Errorf("empty calendar day")
	}
	if !mcp.ValidCheckTradingDayDate(calendarDay) {
		return false, fmt.Errorf("invalid calendar day %q", calendarDay)
	}
	td, err := c.Client.CheckTradingDayOnDate(ctx, c.Token, strings.TrimSpace(code), calendarDay)
	if err != nil {
		return false, err
	}
	if td == nil {
		return false, fmt.Errorf("checkTradingDay empty response")
	}
	q := mcp.CheckTradingDayQuery{Code: strings.TrimSpace(code), Date: calendarDay}
	if err := mcp.ValidateCheckTradingDayData(q, td); err != nil {
		return false, err
	}
	return td.IsTradingDay, nil
}

// failClosedMacroTradingDayChecker always errors (HTTP macro path when MCP missing).
type failClosedMacroTradingDayChecker struct {
	reason string
}

func FailClosedMacroTradingDayChecker(reason string) MacroTradingDayChecker {
	return failClosedMacroTradingDayChecker{reason: strings.TrimSpace(reason)}
}

func (f failClosedMacroTradingDayChecker) IsExchangeTradingDay(context.Context, string, string) (bool, error) {
	if f.reason == "" {
		f.reason = "trading day checker unavailable"
	}
	return false, fmt.Errorf("%s", f.reason)
}