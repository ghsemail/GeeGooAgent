package tools

// BypassPanelFrequencyClarify answers K-line frequency clarifies from Monday panel ops when set.
type BypassPanelFrequencyClarify func(session any, question string, choices []string) (answer string, ok bool)

var bypassPanelFrequencyClarify BypassPanelFrequencyClarify

func setBypassPanelFrequencyClarify(fn BypassPanelFrequencyClarify) {
	bypassPanelFrequencyClarify = fn
}
