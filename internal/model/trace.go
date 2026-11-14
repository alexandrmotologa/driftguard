package model

import (
	"time"
)

// Trace represents a single execution trajectory of an autonomous AI agent.
type Trace struct {
	WorkflowID  string            `json:"workflow_id"`
	TraceID     string            `json:"trace_id"`
	SessionID   string            `json:"session_id,omitempty"`
	Model       string            `json:"model,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	DurationMs  int64             `json:"duration_ms"`
	Status      string            `json:"status"` // success, error, timeout
	StepCount   int               `json:"step_count"`
	Steps       []TraceStep       `json:"steps"`
	TokenUsage  TokenUsage        `json:"token_usage"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// TraceStep records an individual tool call or reasoning turn within a trajectory.
type TraceStep struct {
	StepIndex  int    `json:"step_index"`
	ToolName   string `json:"tool_name"`
	ToolInput  string `json:"tool_input,omitempty"`
	ToolOutput string `json:"tool_output,omitempty"`
	DurationMs int64  `json:"duration_ms"`
	TokensUsed int    `json:"tokens_used,omitempty"`
	Error      string `json:"error,omitempty"`
}

// TokenUsage holds token consumption metrics.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Baseline represents the verified Golden Baseline profile for a workflow.
type Baseline struct {
	WorkflowID                string             `json:"workflow_id"`
	ExpectedStepsMean         float64            `json:"expected_steps_mean"`
	ExpectedStepsStd          float64            `json:"expected_steps_std"`
	StepSamples               []float64          `json:"step_samples,omitempty"`
	MarkovMatrix              MarkovMatrix       `json:"markov_matrix"`
	MonitoredTools            []string           `json:"monitored_tools"`
	MaxAllowedWassersteinDist float64            `json:"max_allowed_wasserstein_distance"`
	MaxToolTransitionDrift    float64            `json:"max_tool_transition_drift"`
	MaxLoopProbability        float64            `json:"max_loop_probability"`
	UpdatedAt                 time.Time          `json:"updated_at"`
	SampleCount               int                `json:"sample_count"`
}

// MarkovMatrix represents the transition probability distribution between tools.
type MarkovMatrix struct {
	States           []string                      `json:"states"`
	Probabilities    map[string]map[string]float64 `json:"probabilities"`
	RawCounts        map[string]map[string]int     `json:"raw_counts,omitempty"`
	TotalTransitions int                           `json:"total_transitions"`
}

// MatrixPerturbation quantifies the statistical shift between two Markov matrices.
type MatrixPerturbation struct {
	MeanAbsDeviation float64          `json:"mean_abs_deviation"`
	MaxEdgeDeviation float64          `json:"max_edge_deviation"`
	MaxEdgeFrom      string           `json:"max_edge_from,omitempty"`
	MaxEdgeTo        string           `json:"max_edge_to,omitempty"`
	NovelTransitions []TransitionEdge `json:"novel_transitions,omitempty"`
	PerturbedEdges   []PerturbedEdge  `json:"perturbed_edges,omitempty"`
}

// TransitionEdge represents a directed transition between two tools.
type TransitionEdge struct {
	From         string  `json:"from"`
	To           string  `json:"to"`
	CurrentProb  float64 `json:"current_prob"`
	BaselineProb float64 `json:"baseline_prob"`
	Count        int     `json:"count"`
}

// PerturbedEdge describes a specific edge whose transition probability has shifted.
type PerturbedEdge struct {
	From         string  `json:"from"`
	To           string  `json:"to"`
	Delta        float64 `json:"delta"`
	BaselineProb float64 `json:"baseline_prob"`
	CurrentProb  float64 `json:"current_prob"`
}

// LoopAnomaly captures high-frequency tool repetition.
type LoopAnomaly struct {
	ToolName    string  `json:"tool_name"`
	Probability float64 `json:"probability"`
	Occurrences int     `json:"occurrences"`
}

// RuleViolation represents a deterministic guardrail failure.
type RuleViolation struct {
	TraceID   string `json:"trace_id"`
	FromTool  string `json:"from_tool"`
	ToTool    string `json:"to_tool"`
	Reason    string `json:"reason"`
	StepIndex int    `json:"step_index"`
}

// EvaluationResult captures the output of a periodic rolling window sentinel audit.
type EvaluationResult struct {
	ID                  string             `json:"id"`
	WorkflowID          string             `json:"workflow_id"`
	Timestamp           time.Time          `json:"timestamp"`
	WindowSize          int                `json:"window_size"`
	WassersteinDistance float64            `json:"wasserstein_distance"`
	StepMean            float64            `json:"step_mean"`
	StepStd             float64            `json:"step_std"`
	StepHistogram       []HistogramBin     `json:"step_histogram,omitempty"`
	MatrixDeviation     MatrixPerturbation `json:"matrix_deviation"`
	CurrentMatrix       MarkovMatrix       `json:"current_matrix"`
	Status              string             `json:"status"` // NORMAL, WARNING, CRITICAL
	LoopAnomalies       []LoopAnomaly      `json:"loop_anomalies,omitempty"`
	NovelTransitions    []TransitionEdge   `json:"novel_transitions,omitempty"`
	RuleViolations      []RuleViolation    `json:"rule_violations,omitempty"`
	Explanation         string             `json:"explanation"`
}

// HistogramBin represents a frequency bin for step count distributions.
type HistogramBin struct {
	Step      int     `json:"step"`
	BaselineP float64 `json:"baseline_p"`
	CurrentP  float64 `json:"current_p"`
}

// Incident records a qualified anomaly requiring SRE attention or paging.
type Incident struct {
	ID                   string            `json:"id"`
	WorkflowID           string            `json:"workflow_id"`
	EvaluationID         string            `json:"evaluation_id,omitempty"`
	Severity             string            `json:"severity"` // WARNING, CRITICAL
	AnomalyType          string            `json:"anomaly_type"` // STEP_EXPLOSION, TOOL_LOOP, FORBIDDEN_TRANSITION, MATRIX_PERTURBATION
	Title                string            `json:"title"`
	Summary              string            `json:"summary"`
	TraceIDs             []string          `json:"trace_ids,omitempty"`
	MetricValues         map[string]float64`json:"metric_values,omitempty"`
	OffendingTransitions []TransitionEdge  `json:"offending_transitions,omitempty"`
	Acknowledged         bool              `json:"acknowledged"`
	CreatedAt            time.Time         `json:"created_at"`
}

// SSEEvent represents a real-time event dispatched to web UI clients.
type SSEEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}
