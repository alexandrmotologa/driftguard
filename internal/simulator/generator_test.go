package simulator

import (
	"testing"
)

func TestGenerator(t *testing.T) {
	gen := NewGenerator(42)

	cleanTrace := gen.GenerateTrace("customer-refund-agent", ScenarioClean)
	if cleanTrace.StepCount != 4 {
		t.Errorf("expected clean trace to have 4 steps, got %d", cleanTrace.StepCount)
	}

	loopTrace := gen.GenerateTrace("customer-refund-agent", ScenarioLoop)
	if loopTrace.StepCount < 6 {
		t.Errorf("expected loop trace to have at least 6 steps, got %d", loopTrace.StepCount)
	}

	bypassTrace := gen.GenerateTrace("customer-refund-agent", ScenarioBypass)
	if bypassTrace.StepCount != 3 {
		t.Errorf("expected bypass trace to have 3 steps, got %d", bypassTrace.StepCount)
	}
	if bypassTrace.Steps[0].ToolName != "lookup_order" || bypassTrace.Steps[1].ToolName != "execute_refund" {
		t.Errorf("expected lookup_order -> execute_refund in bypass scenario")
	}
}
