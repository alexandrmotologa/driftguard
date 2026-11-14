package store

import (
	"context"
	"testing"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

func TestStoreOperations(t *testing.T) {
	ctx := context.Background()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer db.Close()

	// 1. Insert and retrieve trace
	tr := model.Trace{
		WorkflowID: "refund-agent",
		TraceID:    "tr-001",
		SessionID:  "sess-123",
		Model:      "claude-3-7-sonnet",
		Timestamp:  time.Now(),
		DurationMs: 1200,
		Status:     "success",
		StepCount:  2,
		Steps: []model.TraceStep{
			{StepIndex: 1, ToolName: "lookup", DurationMs: 500},
			{StepIndex: 2, ToolName: "execute", DurationMs: 700},
		},
		TokenUsage: model.TokenUsage{TotalTokens: 1500},
	}

	if err := db.InsertTrace(ctx, tr); err != nil {
		t.Fatalf("InsertTrace failed: %v", err)
	}

	count, err := db.GetTraceCount(ctx, "refund-agent")
	if err != nil {
		t.Fatalf("GetTraceCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	traces, err := db.GetRecentTraces(ctx, "refund-agent", 10)
	if err != nil {
		t.Fatalf("GetRecentTraces failed: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].TraceID != "tr-001" {
		t.Errorf("expected trace ID tr-001, got %s", traces[0].TraceID)
	}
	if len(traces[0].Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(traces[0].Steps))
	}

	// 2. Save and retrieve baseline
	baseline := model.Baseline{
		WorkflowID:                "refund-agent",
		ExpectedStepsMean:         3.5,
		ExpectedStepsStd:          0.8,
		SampleCount:               100,
		StepSamples:               []float64{3, 4, 3, 4},
		MonitoredTools:            []string{"lookup", "execute"},
		MaxAllowedWassersteinDist: 1.8,
		MaxToolTransitionDrift:    0.15,
		MaxLoopProbability:        0.20,
		UpdatedAt:                 time.Now(),
	}

	if err := db.SaveBaseline(ctx, baseline); err != nil {
		t.Fatalf("SaveBaseline failed: %v", err)
	}

	bRetrieved, err := db.GetBaseline(ctx, "refund-agent")
	if err != nil {
		t.Fatalf("GetBaseline failed: %v", err)
	}
	if bRetrieved == nil {
		t.Fatal("expected baseline to be found")
	}
	if bRetrieved.ExpectedStepsMean != 3.5 {
		t.Errorf("expected mean 3.5, got %f", bRetrieved.ExpectedStepsMean)
	}

	// 3. Save incident
	inc := model.Incident{
		ID:          "inc-001",
		WorkflowID:  "refund-agent",
		Severity:    "CRITICAL",
		AnomalyType: "TOOL_LOOP",
		Title:       "Tool Recursion Alert",
		Summary:     "Agent entered self-loop on lookup tool",
		CreatedAt:   time.Now(),
	}
	if err := db.SaveIncident(ctx, inc); err != nil {
		t.Fatalf("SaveIncident failed: %v", err)
	}

	incidents, err := db.GetRecentIncidents(ctx, "refund-agent", 10)
	if err != nil {
		t.Fatalf("GetRecentIncidents failed: %v", err)
	}
	if len(incidents) != 1 {
		t.Errorf("expected 1 incident, got %d", len(incidents))
	}
}
