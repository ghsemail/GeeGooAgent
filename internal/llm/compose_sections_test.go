package llm

import (
	"strings"
	"testing"
)

func TestNormalizeComposeSectionsGluedTitle(t *testing.T) {
	in := "### 一、信号触发情况在腾讯控股上，probe 显示多次触发。"
	got := NormalizeComposeSections(in)
	if got == in {
		t.Fatalf("expected split, got %q", got)
	}
	if !strings.Contains(got, "### 一、信号触发情况") || !strings.Contains(got, "在腾讯") {
		t.Fatalf("got %q", got)
	}
}
