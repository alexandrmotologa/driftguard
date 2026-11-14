package metrics

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

// Registry manages Prometheus metrics for driftguard.
type Registry struct {
	mu sync.RWMutex

	// Traces total
	tracesTotal map[string]int64

	// Wasserstein distance by workflow
	wassersteinDist map[string]float64

	// Markov perturbation by workflow
	markovDrift map[string]float64

	// Step mean by workflow
	stepMean map[string]float64

	// Step std by workflow
	stepStd map[string]float64

	// Active incidents by workflow
	activeIncidents map[string]int

	// Incidents total by workflow:severity:type
	incidentsTotal map[string]int64

	// Tool loop probabilities by workflow:tool
	loopProbabilities map[string]float64

	// Circuit breaker state by workflow: 0=CLOSED, 1=OPEN, 2=HALF_OPEN
	circuitBreakerState map[string]int
}

// NewRegistry initializes an empty metrics registry.
func NewRegistry() *Registry {
	return &Registry{
		tracesTotal:         make(map[string]int64),
		wassersteinDist:     make(map[string]float64),
		markovDrift:         make(map[string]float64),
		stepMean:            make(map[string]float64),
		stepStd:             make(map[string]float64),
		activeIncidents:     make(map[string]int),
		incidentsTotal:      make(map[string]int64),
		loopProbabilities:   make(map[string]float64),
		circuitBreakerState: make(map[string]int),
	}
}

// RecordTrace increments trace count for a workflow.
func (r *Registry) RecordTrace(workflowID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tracesTotal[workflowID]++
}

// RecordEvaluation updates gauge metrics after an evaluation.
func (r *Registry) RecordEvaluation(eval model.EvaluationResult) {
	r.mu.Lock()
	defer r.mu.Unlock()

	wf := eval.WorkflowID
	r.wassersteinDist[wf] = eval.WassersteinDistance
	r.markovDrift[wf] = eval.MatrixDeviation.MeanAbsDeviation
	r.stepMean[wf] = eval.StepMean
	r.stepStd[wf] = eval.StepStd

	for _, l := range eval.LoopAnomalies {
		key := fmt.Sprintf("%s:%s", wf, l.ToolName)
		r.loopProbabilities[key] = l.Probability
	}
}

// RecordIncident records a newly triggered incident.
func (r *Registry) RecordIncident(inc model.Incident) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.activeIncidents[inc.WorkflowID]++
	key := fmt.Sprintf("%s:%s:%s", inc.WorkflowID, inc.Severity, inc.AnomalyType)
	r.incidentsTotal[key]++
}

// ResolveIncident decrements active incidents counter.
func (r *Registry) ResolveIncident(workflowID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.activeIncidents[workflowID] > 0 {
		r.activeIncidents[workflowID]--
	}
}

// SetCircuitBreakerState sets the numeric state of circuit breaker.
func (r *Registry) SetCircuitBreakerState(workflowID string, state int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.circuitBreakerState[workflowID] = state
}

// WritePrometheus writes metrics in standard Prometheus text format.
func (r *Registry) WritePrometheus(w io.Writer) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fmt.Fprintln(w, "# HELP driftguard_traces_total Total number of agent traces ingested.")
	fmt.Fprintln(w, "# TYPE driftguard_traces_total counter")
	for wf, count := range r.tracesTotal {
		fmt.Fprintf(w, "driftguard_traces_total{workflow=\"%s\"} %d\n", wf, count)
	}

	fmt.Fprintln(w, "# HELP driftguard_wasserstein_distance Current 1D Wasserstein distance between baseline and rolling window.")
	fmt.Fprintln(w, "# TYPE driftguard_wasserstein_distance gauge")
	for wf, val := range r.wassersteinDist {
		fmt.Fprintf(w, "driftguard_wasserstein_distance{workflow=\"%s\"} %.4f\n", wf, val)
	}

	fmt.Fprintln(w, "# HELP driftguard_markov_drift Mean absolute perturbation of tool transition matrix.")
	fmt.Fprintln(w, "# TYPE driftguard_markov_drift gauge")
	for wf, val := range r.markovDrift {
		fmt.Fprintf(w, "driftguard_markov_drift{workflow=\"%s\"} %.4f\n", wf, val)
	}

	fmt.Fprintln(w, "# HELP driftguard_step_mean Mean reasoning step count in rolling window.")
	fmt.Fprintln(w, "# TYPE driftguard_step_mean gauge")
	for wf, val := range r.stepMean {
		fmt.Fprintf(w, "driftguard_step_mean{workflow=\"%s\"} %.2f\n", wf, val)
	}

	fmt.Fprintln(w, "# HELP driftguard_active_incidents Number of currently active unresolved incidents.")
	fmt.Fprintln(w, "# TYPE driftguard_active_incidents gauge")
	for wf, val := range r.activeIncidents {
		fmt.Fprintf(w, "driftguard_active_incidents{workflow=\"%s\"} %d\n", wf, val)
	}

	fmt.Fprintln(w, "# HELP driftguard_circuit_breaker_state Circuit breaker status (0=CLOSED, 1=OPEN, 2=HALF_OPEN).")
	fmt.Fprintln(w, "# TYPE driftguard_circuit_breaker_state gauge")
	for wf, val := range r.circuitBreakerState {
		fmt.Fprintf(w, "driftguard_circuit_breaker_state{workflow=\"%s\"} %d\n", wf, val)
	}
}

// Handler returns an HTTP handler serving Prometheus metrics.
func (r *Registry) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		r.WritePrometheus(w)
	}
}
