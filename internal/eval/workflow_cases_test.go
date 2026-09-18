package eval

import "testing"

func TestIndividualWorkflowEvalCases(t *testing.T) {
	cases := IndividualWorkflowEvalCases()
	if len(cases) < 4 {
		t.Fatalf("expected at least 4 workflow eval cases, got %d", len(cases))
	}
	for _, c := range cases {
		if c.Options.Category != "workflow" {
			t.Fatalf("%s category=%s want workflow", c.ID, c.Options.Category)
		}
		if c.Options.Message == "" {
			t.Fatalf("%s missing message", c.ID)
		}
		if len(c.Options.PassKeywords) == 0 {
			t.Fatalf("%s missing pass_keywords", c.ID)
		}
	}
}
