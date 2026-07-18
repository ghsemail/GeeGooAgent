package verify_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/verify"
)

func TestVerifyAgentLoopParityOnRegistry(t *testing.T) {
	t.Parallel()
	reg := tools.NewRegistry()
	reg.Register(tools.Tool{Name: "delegate_task"})
	reg.Register(tools.Tool{Name: "recall"})
	reg.Register(tools.Tool{Name: "search_code"})
	cards := verify.VerifyAgentLoopParity(reg)
	if !verify.AllAgentLoopPass(cards) {
		for _, c := range cards {
			if !c.Passed {
				t.Fatalf("%s", c.Summary())
			}
		}
	}
}

func TestVerifyAgentLoopParityFailsMissingDelegate(t *testing.T) {
	t.Parallel()
	cards := verify.VerifyAgentLoopParity(tools.NewRegistry())
	if verify.AllAgentLoopPass(cards) {
		t.Fatal("expected failure without tools")
	}
}
