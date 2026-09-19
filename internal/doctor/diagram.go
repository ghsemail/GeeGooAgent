package doctor

import "github.com/ghsemail/GeeGooAgent/internal/diagram"

func diagramChecks() []CheckResult {
	out := make([]CheckResult, 0, 2)
	for _, c := range diagram.Checks(".") {
		out = append(out, CheckResult{Name: c.Name, OK: c.OK, Warn: c.Warn, Detail: c.Detail})
	}
	return out
}
