package markdownnorm_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/markdownnorm"
)

func TestNormalizeForStorage_SplitsGluedHeadings(t *testing.T) {
	in := "##你好！我是 GeeGoo Agent###1.股票分析 **实时行情-** 查询价格"
	got := markdownnorm.NormalizeForStorage(in)
	if !strings.Contains(got, "\n") {
		t.Fatalf("expected line breaks: %q", got)
	}
	if !strings.Contains(got, "##") {
		t.Fatalf("expected heading marker preserved for web markdown: %q", got)
	}
}

func TestNormalizeForStorage_KeepsPipeTables(t *testing.T) {
	in := "|信号|核心逻辑|\n|------|----------|\n|SAR+MACD|趋势共振|"
	got := markdownnorm.NormalizeForStorage(in)
	if !strings.Contains(got, "|信号|") || !strings.Contains(got, "|SAR+MACD|") {
		t.Fatalf("web storage should keep markdown tables: %q", got)
	}
}

func TestNormalizeForTerminalDisplay_ConvertsTables(t *testing.T) {
	in := "|Bot名称|标的|开关|\n|网格A|00700|开|"
	got := markdownnorm.NormalizeForTerminalDisplay(in)
	if strings.Contains(got, "|Bot名称|") {
		t.Fatalf("terminal path should convert bot summary tables: %q", got)
	}
}
