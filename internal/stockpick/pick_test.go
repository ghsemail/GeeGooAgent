package stockpick_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/stockpick"
)

func TestPickByCode(t *testing.T) {
	items := []map[string]any{
		{"code": "00700.HK", "name": "腾讯控股"},
		{"code": "01698.HK", "name": "腾讯音乐-SW"},
	}
	row, ok := stockpick.AutoPick("0700.HK", items)
	if !ok || row["code"] != "00700.HK" {
		t.Fatalf("picked=%v ok=%v", row, ok)
	}
}

func TestAutoPickTencentWithoutAmbiguity(t *testing.T) {
	items := []map[string]any{
		{"code": "00700.HK", "name": "腾讯控股"},
		{"code": "01698.HK", "name": "腾讯音乐-SW"},
	}
	row, ok := stockpick.AutoPick("腾讯", items)
	if !ok || row["code"] != "00700.HK" {
		t.Fatalf("row=%v ok=%v", row, ok)
	}
}

func TestAutoPickPrimaryAliasTencent(t *testing.T) {
	items := []map[string]any{
		{"code": "01698.HK", "name": "腾讯音乐-SW"},
		{"code": "00700.HK", "name": "腾讯控股"},
		{"code": "TCEHY", "name": "Tencent Holdings ADR"},
	}
	row, ok := stockpick.AutoPick("腾讯", items)
	if !ok || row["code"] != "00700.HK" {
		t.Fatalf("picked=%v ok=%v", row, ok)
	}
}

func TestAutoPickNameRankTencent(t *testing.T) {
	items := []map[string]any{
		{"code": "01698.HK", "name": "腾讯音乐-SW"},
		{"code": "00700.HK", "name": "腾讯控股"},
	}
	row, ok := stockpick.AutoPick("腾讯", items)
	if !ok || row["code"] != "00700.HK" {
		t.Fatalf("picked=%v ok=%v", row, ok)
	}
}
