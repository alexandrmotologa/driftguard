package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/config"
	"github.com/alexandrmotologa/driftguard/internal/model"
	"github.com/alexandrmotologa/driftguard/internal/store"
)

func TestAPIRoutes(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Listen:          ":8080",
			EvalIntervalSec: 60,
		},
		Workflows: []config.WorkflowConfig{
			{
				ID:   "customer-refund-agent",
				Name: "Refund Automation",
				Baseline: config.BaselineConfig{
					ExpectedStepsMean:         4.0,
					ExpectedStepsStd:          1.0,
					MaxAllowedWassersteinDist: 2.0,
					MaxToolTransitionDrift:    0.20,
				},
				ToolsMonitored: []string{"lookup", "refund"},
			},
		},
	}

	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer db.Close()

	srv := NewServer(cfg, db, nil)

	// Ingest a trace
	tr := model.Trace{
		WorkflowID: "customer-refund-agent",
		TraceID:    "tr-test-1",
		Timestamp:  time.Now(),
		DurationMs: 1000,
		StepCount:  2,
		Steps: []model.TraceStep{
			{StepIndex: 1, ToolName: "lookup"},
			{StepIndex: 2, ToolName: "refund"},
		},
	}
	body, _ := json.Marshal(tr)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/traces", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.handleIngestSingleTrace(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}

	// Verify trace in database
	count, err := db.GetTraceCount(context.Background(), "customer-refund-agent")
	if err != nil || count != 1 {
		t.Fatalf("expected count 1, got %d (err: %v)", count, err)
	}

	// Test workflows listing
	reqWf := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	wWf := httptest.NewRecorder()
	srv.handleGetWorkflows(wWf, reqWf)

	if wWf.Code != http.StatusOK {
		t.Errorf("expected 200 for workflows, got %d", wWf.Code)
	}

	var summaries []WorkflowStatusSummary
	if err := json.Unmarshal(wWf.Body.Bytes(), &summaries); err != nil {
		t.Fatalf("failed to decode workflows response: %v", err)
	}
	if len(summaries) != 1 || summaries[0].ID != "customer-refund-agent" {
		t.Errorf("unexpected workflows response: %+v", summaries)
	}

	// Test simulation endpoint
	simReq := SimulateRequest{
		WorkflowID: "customer-refund-agent",
		Scenario:   "clean",
		Count:      5,
	}
	simBody, _ := json.Marshal(simReq)
	reqSim := httptest.NewRequest(http.MethodPost, "/api/v1/simulate", bytes.NewReader(simBody))
	wSim := httptest.NewRecorder()
	srv.handleSimulateTraffic(wSim, reqSim)

	if wSim.Code != http.StatusOK {
		t.Errorf("expected 200 for simulate, got %d", wSim.Code)
	}

	// Count should now be 6
	countAfter, _ := db.GetTraceCount(context.Background(), "customer-refund-agent")
	if countAfter != 6 {
		t.Errorf("expected 6 traces, got %d", countAfter)
	}
}
