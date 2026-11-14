package diff

import (
	"testing"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

func TestComputeTrajectoryDiff_LoopAndOmission(t *testing.T) {
	canonical := []string{"lookup", "fraud_check", "execute", "email"}

	// Trace has loop on lookup and skips fraud_check
	tr := model.Trace{
		TraceID: "tr-bad",
		Steps: []model.TraceStep{
			{ToolName: "lookup"},
			{ToolName: "lookup"},
			{ToolName: "execute"},
			{ToolName: "email"},
		},
	}

	forbidden := map[string]string{"lookup": "execute"}
	diff := ComputeTrajectoryDiff(tr, canonical, forbidden)

	if diff.LoopCount != 1 {
		t.Errorf("expected 1 loop detected, got %d", diff.LoopCount)
	}
	if diff.OmissionCount != 1 {
		t.Errorf("expected 1 omission detected (fraud_check), got %d", diff.OmissionCount)
	}
	if !diff.HasGuardrailBypass {
		t.Errorf("expected guardrail bypass flag to be true")
	}
}
