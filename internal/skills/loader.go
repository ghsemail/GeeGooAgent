package skills

import "github.com/ghsemail/GeeGooAgent/internal/workflow"

// RegisterBuiltins registers all built-in skills (pre_market, and placeholders
// for intraday/post_market) into the given registry.
//
// The step functions delegate to workflow.PhaseASteps / workflow.PerStockSteps
// so the skill definitions stay declarative while the step construction lives
// in one place. New skills are added here (or via Register at runtime) rather
// than by branching on skill name in cmd/geegoo or internal/app.
func RegisterBuiltins(r *Registry) {
	r.Register(Spec{
		Name:         "pre_market",
		Description:  "盘前分析：指数 + 市场新闻 + 个股资金/技术/Bot 态度，生成盘前报告并入库",
		PhaseA:       workflow.PhaseASteps,
		PerStock:     workflow.PerStockSteps,
		TemplatePath: "skills/pre_market/template.md",
		ManifestPath: "skills/pre_market/manifest.yaml",
	})
	r.Register(Spec{
		Name:         "intraday",
		Description:  "盘中交易决策报告（占位，待实现步骤）",
		PhaseA:       emptySteps,
		PerStock:     emptySteps,
		ManifestPath: "skills/intraday/manifest.yaml",
	})
	r.Register(Spec{
		Name:         "post_market",
		Description:  "盘后总结报告（占位，待实现步骤）",
		PhaseA:       emptySteps,
		PerStock:     emptySteps,
		ManifestPath: "skills/post_market/manifest.yaml",
	})
}

func emptySteps() []workflow.Step { return nil }
