package scheduler

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/stockdigest"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func shouldNotifyJob(job Job) bool {
	if strings.ToLower(strings.TrimSpace(job.Platform)) != "feishu" {
		return false
	}
	switch job.Skill {
	case "premarket_stock", "postmarket_stock":
		return true
	default:
		return false
	}
}

func isPerUserStockNotify(skill string) bool {
	switch skill {
	case "premarket_stock", "postmarket_stock":
		return true
	default:
		return false
	}
}

// ShouldNotifyJobForTest exposes notify gating for unit tests.
func ShouldNotifyJobForTest(job Job) bool { return shouldNotifyJob(job) }

// BuildFeishuSummary formats stock pre/post market digests for Feishu IM.
func BuildFeishuSummary(job Job, result workflow.RunResult) string {
	return stockdigest.Build(job.Skill, job.Market, result)
}
