package tools

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
)

func TestCapitalDistributionHasData(t *testing.T) {
	if capitalDistributionHasData(nil) {
		t.Fatal("nil should be empty")
	}
	if !capitalDistributionHasData(&mcp.CapitalDistributionData{CapitalOutBig: 1}) {
		t.Fatal("out field should count")
	}
	if capitalDistributionHasData(&mcp.CapitalDistributionData{}) {
		t.Fatal("all zero should be empty")
	}
	if !capitalDistributionHasData(&mcp.CapitalDistributionData{UpdateTime: "2026-07-17"}) {
		t.Fatal("update_time should count")
	}
}

func TestStockNewsNeedsFallback(t *testing.T) {
	if !stockNewsNeedsFallback("") {
		t.Fatal("empty needs fallback")
	}
	if !stockNewsNeedsFallback("（暂无数据）") {
		t.Fatal("placeholder needs fallback")
	}
	if stockNewsNeedsFallback("**1. Real headline**") {
		t.Fatal("real content should not need fallback")
	}
}

func TestBuildMarketNewsResultEmptyErrors(t *testing.T) {
	res := buildMarketNewsResult("US", "（暂无数据）", "finance-news", nil)
	if res.Status != StatusError {
		t.Fatalf("expected error for empty market news, got %s: %s", res.Status, res.Summary)
	}
}

func TestBuildMarketNewsResultWithContentOK(t *testing.T) {
	res := buildMarketNewsResult("US", "**1. Fed holds rates**\n", "finance-news", nil)
	if res.Status != StatusOK {
		t.Fatalf("expected ok, got %s", res.Status)
	}
}

func TestMarketNewsQuery(t *testing.T) {
	if marketNewsQuery("cn") != "CN 股市 新闻" {
		t.Fatal("cn query")
	}
	if marketNewsQuery("US") != "US stock market news today" {
		t.Fatal("us query")
	}
}
