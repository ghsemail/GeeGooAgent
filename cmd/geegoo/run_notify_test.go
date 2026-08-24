package main

import "testing"

func TestDefaultFeishuNotifyForSkill(t *testing.T) {
	for _, skill := range []string{"premarket_stock", "postmarket_stock"} {
		if !defaultFeishuNotifyForSkill(skill) {
			t.Fatalf("skill %q should default to feishu notify", skill)
		}
	}
	for _, skill := range []string{"premarket_market", "intraday_stock", ""} {
		if defaultFeishuNotifyForSkill(skill) {
			t.Fatalf("skill %q should not default to feishu notify", skill)
		}
	}
}
