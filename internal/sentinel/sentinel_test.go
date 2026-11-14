package sentinel

import (
	"testing"

	"github.com/alexandrmotologa/driftguard/internal/config"
	"github.com/alexandrmotologa/driftguard/internal/model"
)

func TestCheckTraceRules(t *testing.T) {
	forbidden := []config.ForbiddenTransition{
		{From: "lookup_order", To: "execute_refund"},
	}

	// Compliant trace: lookup -> check_fraud -> execute_refund
	compliantTrace := model.Trace{
		TraceID: "tr-good",
		Steps: []model.TraceStep{
			{ToolName: "lookup_order"},
			{ToolName: "check_fraud"},
			{ToolName: "execute_refund"},
		},
	}
	violations := CheckTraceRules(compliantTrace, forbidden, nil)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for compliant trace, got %d", len(violations))
	}

	// Violating trace: lookup -> execute_refund
	badTrace := model.Trace{
		TraceID: "tr-bad",
		Steps: []model.TraceStep{
			{ToolName: "lookup_order"},
			{ToolName: "execute_refund"},
		},
	}
	violationsBad := CheckTraceRules(badTrace, forbidden, nil)
	if len(violationsBad) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violationsBad))
	}
	if violationsBad[0].FromTool != "lookup_order" || violationsBad[0].ToTool != "execute_refund" {
		t.Errorf("unexpected violation details: %+v", violationsBad[0])
	}
}

func TestEvaluateWindow_LoopDrift(t *testing.T) {
	cfg := &config.WorkflowConfig{
		ID:                "test-agent",
		RollingWindowRuns: 10,
		Baseline: config.BaselineConfig{
			ExpectedStepsMean:         3.0,
			ExpectedStepsStd:          0.5,
			MaxAllowedWassersteinDist: 1.5,
			MaxToolTransitionDrift:    0.15,
			MaxLoopProbability:        0.20,
		},
		ToolsMonitored: []string{"search", "plan", "execute"},
	}

	cleanTraces := []model.Trace{
		{
			StepCount: 3,
			Steps: []model.TraceStep{
				{ToolName: "search"},
				{ToolName: "plan"},
				{ToolName: "execute"},
			},
		},
	}
	baseline, err := CalibrateFromTraces("test-agent", cleanTraces, cfg)
	if err != nil {
		t.Fatalf("calibration failed: %v", err)
	}

	// Simulate looping traces
	driftingTraces := make([]model.Trace, 10)
	for i := 0; i < 10; i++ {
		driftingTraces[i] = model.Trace{
			TraceID:   "tr-loop",
			StepCount: 8,
			Steps: []model.TraceStep{
				{ToolName: "search"},
				{ToolName: "search"},
				{ToolName: "search"},
				{ToolName: "search"},
				{ToolName: "plan"},
				{ToolName: "execute"},
			},
		}
	}

	eval, inc := EvaluateWindow("test-agent", driftingTraces, baseline, cfg)
	if eval.Status == "NORMAL" {
		t.Errorf("expected status WARNING or CRITICAL for looping traces, got %s", eval.Status)
	}
	if inc == nil {
		t.Fatal("expected incident to be generated")
	}
	if eval.WassersteinDistance <= 1.5 {
		t.Errorf("expected elevated Wasserstein distance > 1.5, got %f", eval.WassersteinDistance)
	}
}
