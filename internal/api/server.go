package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/config"
	"github.com/alexandrmotologa/driftguard/internal/ingest"
	"github.com/alexandrmotologa/driftguard/internal/model"
	"github.com/alexandrmotologa/driftguard/internal/notify"
	"github.com/alexandrmotologa/driftguard/internal/sentinel"
	"github.com/alexandrmotologa/driftguard/internal/simulator"
	"github.com/alexandrmotologa/driftguard/internal/store"
)

// Server coordinates the HTTP REST API, SSE streaming, and background sentinel auditing.
type Server struct {
	cfg        *config.Config
	db         *store.DB
	hub        *EventHub
	dispatcher *notify.Dispatcher
	staticFS   fs.FS
	httpServer *http.Server
	sim        *simulator.Generator

	evalMu sync.Mutex
}

// NewServer initializes a new Server.
func NewServer(cfg *config.Config, db *store.DB, staticFS fs.FS) *Server {
	return &Server{
		cfg:        cfg,
		db:         db,
		hub:        NewEventHub(),
		dispatcher: notify.NewDispatcher(cfg.Alerting),
		staticFS:   staticFS,
		sim:        simulator.NewGenerator(0),
	}
}

// Start begins listening on HTTP and starts background sentinel routines.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Ingestion endpoints
	mux.HandleFunc("POST /api/v1/traces", s.handleIngestSingleTrace)
	mux.HandleFunc("POST /api/v1/traces/batch", s.handleIngestTraceBatch)
	mux.HandleFunc("POST /v1/traces", s.handleIngestOTelTraces)

	// Workflows, Baselines, and Metrics
	mux.HandleFunc("GET /api/v1/workflows", s.handleGetWorkflows)
	mux.HandleFunc("GET /api/v1/workflows/{id}/baseline", s.handleGetBaseline)
	mux.HandleFunc("POST /api/v1/workflows/{id}/recalibrate", s.handleRecalibrateBaseline)
	mux.HandleFunc("GET /api/v1/workflows/{id}/traces", s.handleGetTraces)
	mux.HandleFunc("GET /api/v1/workflows/{id}/metrics", s.handleGetMetrics)
	mux.HandleFunc("GET /api/v1/workflows/{id}/evaluations", s.handleGetEvaluations)
	mux.HandleFunc("GET /api/v1/workflows/{id}/incidents", s.handleGetIncidents)
	mux.HandleFunc("POST /api/v1/incidents/{id}/ack", s.handleAckIncident)

	// Simulation trigger
	mux.HandleFunc("POST /api/v1/simulate", s.handleSimulateTraffic)

	// Real-time SSE stream
	mux.HandleFunc("GET /api/v1/events", s.hub.ServeSSE)

	// SPA Static assets
	if s.staticFS != nil {
		fileServer := http.FileServer(http.FS(s.staticFS))
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" || path == "index.html" {
				r.URL.Path = "/"
				fileServer.ServeHTTP(w, r)
				return
			}

			// Check if file exists in staticFS
			f, err := s.staticFS.Open(path)
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}

			// SPA fallback to index.html for client-side routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
		})
	} else {
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"running","service":"driftguard","version":"1.0.0"}`))
		})
	}

	// CORS wrapper
	handler := corsMiddleware(mux)

	s.httpServer = &http.Server{
		Addr:    s.cfg.Server.Listen,
		Handler: handler,
	}

	// Start background sentinel audit worker
	go s.runSentinelWorker()

	return s.httpServer.ListenAndServe()
}

// Shutdown stops the server gracefully.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// corsMiddleware adds standard development and production CORS headers.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleIngestSingleTrace(w http.ResponseWriter, r *http.Request) {
	tr, err := ingest.ParseSingleTrace(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid trace: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.db.InsertTrace(r.Context(), *tr); err != nil {
		http.Error(w, fmt.Sprintf("storing trace: %v", err), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast("trace_ingested", tr)

	// Trigger async evaluation check
	go s.evaluateWorkflowAsync(tr.WorkflowID)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{"status": "accepted", "trace_id": tr.TraceID})
}

func (s *Server) handleIngestTraceBatch(w http.ResponseWriter, r *http.Request) {
	traces, err := ingest.ParseTraceBatch(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid trace batch: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.db.InsertTracesBatch(r.Context(), traces); err != nil {
		http.Error(w, fmt.Sprintf("storing batch: %v", err), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast("trace_batch_ingested", map[string]any{"count": len(traces)})

	workflows := make(map[string]struct{})
	for _, tr := range traces {
		workflows[tr.WorkflowID] = struct{}{}
	}
	for wf := range workflows {
		go s.evaluateWorkflowAsync(wf)
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{"status": "accepted", "ingested": len(traces)})
}

func (s *Server) handleIngestOTelTraces(w http.ResponseWriter, r *http.Request) {
	traces, err := ingest.ParseOTelTraces(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid otel trace payload: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.db.InsertTracesBatch(r.Context(), traces); err != nil {
		http.Error(w, fmt.Sprintf("storing otel traces: %v", err), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast("trace_batch_ingested", map[string]any{"count": len(traces)})

	for _, tr := range traces {
		go s.evaluateWorkflowAsync(tr.WorkflowID)
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{"status": "accepted", "converted_traces": len(traces)})
}

type WorkflowStatusSummary struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	Status             string                  `json:"status"` // NORMAL, WARNING, CRITICAL
	TraceCount         int                     `json:"trace_count"`
	WassersteinDist    float64                 `json:"wasserstein_distance"`
	MatrixDrift        float64                 `json:"matrix_drift"`
	LatestEvaluation   *model.EvaluationResult `json:"latest_evaluation,omitempty"`
	LatestIncident     *model.Incident         `json:"latest_incident,omitempty"`
	ConfiguredBaseline bool                    `json:"configured_baseline"`
}

func (s *Server) handleGetWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	summaries := make([]WorkflowStatusSummary, 0, len(s.cfg.Workflows))

	for _, wf := range s.cfg.Workflows {
		count, _ := s.db.GetTraceCount(ctx, wf.ID)
		baseline, _ := s.db.GetBaseline(ctx, wf.ID)
		evals, _ := s.db.GetRecentEvaluations(ctx, wf.ID, 1)
		incidents, _ := s.db.GetRecentIncidents(ctx, wf.ID, 1)

		status := "NORMAL"
		var latestEval *model.EvaluationResult
		var wDist, mDrift float64

		if len(evals) > 0 {
			latestEval = &evals[0]
			status = latestEval.Status
			wDist = latestEval.WassersteinDistance
			mDrift = latestEval.MatrixDeviation.MeanAbsDeviation
		}

		var latestInc *model.Incident
		if len(incidents) > 0 {
			latestInc = &incidents[0]
			if !latestInc.Acknowledged && (latestInc.Severity == "CRITICAL" || latestInc.Severity == "WARNING") {
				status = latestInc.Severity
			}
		}

		summaries = append(summaries, WorkflowStatusSummary{
			ID:                 wf.ID,
			Name:               wf.Name,
			Status:             status,
			TraceCount:         count,
			WassersteinDist:    wDist,
			MatrixDrift:        mDrift,
			LatestEvaluation:   latestEval,
			LatestIncident:     latestInc,
			ConfiguredBaseline: baseline != nil,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

func (s *Server) handleGetBaseline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	baseline, err := s.db.GetBaseline(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if baseline == nil {
		// Return config baseline defaults
		for _, wf := range s.cfg.Workflows {
			if wf.ID == id {
				baseline = &model.Baseline{
					WorkflowID:                id,
					ExpectedStepsMean:         wf.Baseline.ExpectedStepsMean,
					ExpectedStepsStd:          wf.Baseline.ExpectedStepsStd,
					MonitoredTools:            wf.ToolsMonitored,
					MaxAllowedWassersteinDist: wf.Baseline.MaxAllowedWassersteinDist,
					MaxToolTransitionDrift:    wf.Baseline.MaxToolTransitionDrift,
					MaxLoopProbability:        wf.Baseline.MaxLoopProbability,
					UpdatedAt:                 time.Now().UTC(),
				}
				break
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(baseline)
}

func (s *Server) handleRecalibrateBaseline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	var wfCfg *config.WorkflowConfig
	for i := range s.cfg.Workflows {
		if s.cfg.Workflows[i].ID == id {
			wfCfg = &s.cfg.Workflows[i]
			break
		}
	}

	traces, err := s.db.GetRecentTraces(ctx, id, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(traces) == 0 {
		http.Error(w, "no historical traces available to calibrate baseline", http.StatusBadRequest)
		return
	}

	baseline, err := sentinel.CalibrateFromTraces(id, traces, wfCfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("calibrating baseline: %v", err), http.StatusInternalServerError)
		return
	}

	if err := s.db.SaveBaseline(ctx, *baseline); err != nil {
		http.Error(w, fmt.Sprintf("saving baseline: %v", err), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast("baseline_updated", baseline)

	// Run immediate re-evaluation with the new baseline
	go s.evaluateWorkflowAsync(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(baseline)
}

func (s *Server) handleGetTraces(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
		limit = val
	}

	traces, err := s.db.GetRecentTraces(r.Context(), id, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(traces)
}

func (s *Server) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	eval, _ := s.runEvaluationForWorkflow(r.Context(), id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eval)
}

func (s *Server) handleGetEvaluations(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	evals, err := s.db.GetRecentEvaluations(r.Context(), id, 30)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(evals)
}

func (s *Server) handleGetIncidents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	incidents, err := s.db.GetRecentIncidents(r.Context(), id, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(incidents)
}

func (s *Server) handleAckIncident(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.db.AcknowledgeIncident(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast("incident_acknowledged", map[string]string{"incident_id": id})
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"acknowledged": true})
}

type SimulateRequest struct {
	WorkflowID string `json:"workflow_id"`
	Scenario   string `json:"scenario"` // clean, loop, bypass, hallucinate
	Count      int    `json:"count"`
}

func (s *Server) handleSimulateTraffic(w http.ResponseWriter, r *http.Request) {
	var req SimulateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid simulation request", http.StatusBadRequest)
		return
	}

	if req.WorkflowID == "" {
		if len(s.cfg.Workflows) > 0 {
			req.WorkflowID = s.cfg.Workflows[0].ID
		} else {
			req.WorkflowID = "customer-refund-agent"
		}
	}
	if req.Count <= 0 {
		req.Count = 10
	}
	if req.Count > 100 {
		req.Count = 100
	}

	traces := s.sim.GenerateBatch(req.WorkflowID, simulator.Scenario(req.Scenario), req.Count)
	if err := s.db.InsertTracesBatch(r.Context(), traces); err != nil {
		http.Error(w, fmt.Sprintf("storing simulated traces: %v", err), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast("traffic_simulated", map[string]any{
		"workflow_id": req.WorkflowID,
		"scenario":    req.Scenario,
		"count":       len(traces),
	})

	go s.evaluateWorkflowAsync(req.WorkflowID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":      "simulated",
		"workflow_id": req.WorkflowID,
		"scenario":    req.Scenario,
		"count":       len(traces),
	})
}

func (s *Server) evaluateWorkflowAsync(workflowID string) {
	s.evalMu.Lock()
	defer s.evalMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	eval, inc := s.runEvaluationForWorkflow(ctx, workflowID)
	if eval == nil {
		return
	}

	_ = s.db.SaveEvaluation(ctx, *eval)
	s.hub.Broadcast("evaluation_completed", eval)

	if inc != nil {
		_ = s.db.SaveIncident(ctx, *inc)
		s.hub.Broadcast("incident_triggered", inc)
		s.dispatcher.Dispatch(ctx, *inc)
	}
}

func (s *Server) runEvaluationForWorkflow(ctx context.Context, workflowID string) (*model.EvaluationResult, *model.Incident) {
	var wfCfg *config.WorkflowConfig
	for i := range s.cfg.Workflows {
		if s.cfg.Workflows[i].ID == workflowID {
			wfCfg = &s.cfg.Workflows[i]
			break
		}
	}

	windowSize := 50
	if wfCfg != nil && wfCfg.RollingWindowRuns > 0 {
		windowSize = wfCfg.RollingWindowRuns
	}

	traces, err := s.db.GetRecentTraces(ctx, workflowID, windowSize)
	if err != nil || len(traces) == 0 {
		return &model.EvaluationResult{
			WorkflowID:  workflowID,
			Timestamp:   time.Now().UTC(),
			Status:      "NORMAL",
			Explanation: "No traces recorded yet.",
		}, nil
	}

	baseline, _ := s.db.GetBaseline(ctx, workflowID)
	eval, inc := sentinel.EvaluateWindow(workflowID, traces, baseline, wfCfg)
	return &eval, inc
}

func (s *Server) runSentinelWorker() {
	interval := time.Duration(s.cfg.Server.EvalIntervalSec) * time.Second
	if interval < 5*time.Second {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		for _, wf := range s.cfg.Workflows {
			s.evaluateWorkflowAsync(wf.ID)
		}
	}
}
