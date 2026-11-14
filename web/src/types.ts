export interface CircuitBreakerStatus {
  workflow_id: string;
  enabled: boolean;
  state: 'CLOSED' | 'OPEN' | 'HALF_OPEN';
  consecutive_faults: number;
  threshold: number;
  last_tripped_at?: string;
  last_trigger_reason?: string;
}

export interface WorkflowSummary {
  id: string;
  name: string;
  status: 'NORMAL' | 'WARNING' | 'CRITICAL';
  trace_count: number;
  wasserstein_distance: number;
  matrix_drift: number;
  latest_evaluation?: EvaluationResult;
  latest_incident?: Incident;
  configured_baseline: boolean;
  circuit_breaker?: CircuitBreakerStatus;
}

export interface TraceStep {
  step_index: number;
  tool_name: string;
  tool_input?: string;
  tool_output?: string;
  duration_ms: number;
  tokens_used?: number;
  error?: string;
}

export interface TokenUsage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

export interface Trace {
  workflow_id: string;
  trace_id: string;
  session_id?: string;
  model?: string;
  timestamp: string;
  duration_ms: number;
  status: string;
  step_count: number;
  steps: TraceStep[];
  token_usage: TokenUsage;
  metadata?: Record<string, string>;
}

export interface MarkovMatrix {
  states: string[];
  probabilities: Record<string, Record<string, number>>;
  raw_counts?: Record<string, Record<string, number>>;
  total_transitions: number;
}

export interface TransitionEdge {
  from: string;
  to: string;
  current_prob: number;
  baseline_prob: number;
  count: number;
}

export interface PerturbedEdge {
  from: string;
  to: string;
  delta: number;
  baseline_prob: number;
  current_prob: number;
}

export interface MatrixPerturbation {
  mean_abs_deviation: number;
  max_edge_deviation: number;
  max_edge_from?: string;
  max_edge_to?: string;
  novel_transitions?: TransitionEdge[];
  perturbed_edges?: PerturbedEdge[];
}

export interface LoopAnomaly {
  tool_name: string;
  probability: number;
  occurrences: number;
}

export interface RuleViolation {
  trace_id: string;
  from_tool: string;
  to_tool: string;
  reason: string;
  step_index: number;
}

export interface HistogramBin {
  step: number;
  baseline_p: number;
  current_p: number;
}

export interface EvaluationResult {
  id: string;
  workflow_id: string;
  timestamp: string;
  window_size: number;
  wasserstein_distance: number;
  micro_wasserstein?: number;
  macro_wasserstein?: number;
  step_mean: number;
  step_std: number;
  step_histogram?: HistogramBin[];
  matrix_deviation: MatrixPerturbation;
  current_matrix: MarkovMatrix;
  status: 'NORMAL' | 'WARNING' | 'CRITICAL';
  loop_anomalies?: LoopAnomaly[];
  novel_transitions?: TransitionEdge[];
  rule_violations?: RuleViolation[];
  explanation: string;
}

export interface Baseline {
  workflow_id: string;
  expected_steps_mean: number;
  expected_steps_std: number;
  sample_count: number;
  step_samples?: number[];
  markov_matrix: MarkovMatrix;
  monitored_tools: string[];
  max_allowed_wasserstein_distance: number;
  max_tool_transition_drift: number;
  max_loop_probability: number;
  updated_at: string;
}

export interface Incident {
  id: string;
  workflow_id: string;
  evaluation_id?: string;
  severity: 'WARNING' | 'CRITICAL';
  anomaly_type: string;
  title: string;
  summary: string;
  trace_ids?: string[];
  metric_values?: Record<string, number>;
  offending_transitions?: TransitionEdge[];
  acknowledged: boolean;
  created_at: string;
}

export interface DiffStep {
  step_index: number;
  expected_tool?: string;
  observed_tool?: string;
  type: 'MATCH' | 'LOOP' | 'OMISSION' | 'INSERTION' | 'REPLACEMENT';
  explanation: string;
}

export interface TrajectoryDiff {
  trace_id: string;
  workflow_id: string;
  canonical_steps: string[];
  observed_steps: string[];
  step_diffs: DiffStep[];
  total_deviations: number;
  loop_count: number;
  omission_count: number;
  has_guardrail_bypass: boolean;
}

export interface TimelinePoint {
  timestamp: string;
  wasserstein_distance: number;
  micro_wasserstein: number;
  matrix_drift: number;
  step_mean: number;
  status: 'NORMAL' | 'WARNING' | 'CRITICAL';
}
