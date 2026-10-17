package strategydev

import (
	"os"
	"testing"
)

// Package tests exercise price-analysis paths; enable the master switch for the suite.
func TestMain(m *testing.M) {
	ConfigurePriceAnalysisEnabled(true)
	os.Exit(m.Run())
}
