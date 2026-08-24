package app

// SynthesisReady reports whether report LLM synthesis (ops 主备) is wired and callable.
func (a *App) SynthesisReady() bool {
	if a == nil || a.SynthesisGateway == nil {
		return false
	}
	if a.Agent == nil {
		return false
	}
	rs := a.Agent.ReportSynthesizer()
	return rs != nil && rs.Available()
}
