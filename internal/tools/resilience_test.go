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
