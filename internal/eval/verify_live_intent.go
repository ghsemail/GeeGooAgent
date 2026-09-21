package eval

// skipLiveTurnPlanIntentVerify: live Dock Chat no longer emits IntentPlanner telemetry.
func skipLiveTurnPlanIntentVerify(opts TurnPlanCaseOptions) bool {
	opts = opts.Normalize()
	return !opts.PlanOnly
}
