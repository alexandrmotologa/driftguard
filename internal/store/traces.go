package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

// InsertTrace inserts an individual trace and its corresponding steps in a transaction.
func (db *DB) InsertTrace(ctx context.Context, tr model.Trace) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	metaJSON, _ := json.Marshal(tr.Metadata)
	rawJSON, _ := json.Marshal(tr)

	query := `
	INSERT INTO traces (
		trace_id, workflow_id, session_id, model, timestamp, duration_ms,
		status, step_count, prompt_tokens, completion_tokens, total_tokens, metadata_json, raw_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(trace_id) DO UPDATE SET
		duration_ms = excluded.duration_ms,
		status = excluded.status,
		step_count = excluded.step_count,
		total_tokens = excluded.total_tokens
	`

	_, err = tx.ExecContext(ctx, query,
		tr.TraceID, tr.WorkflowID, tr.SessionID, tr.Model, tr.Timestamp.UTC(), tr.DurationMs,
		tr.Status, tr.StepCount, tr.TokenUsage.PromptTokens, tr.TokenUsage.CompletionTokens,
		tr.TokenUsage.TotalTokens, string(metaJSON), string(rawJSON),
	)
	if err != nil {
		return fmt.Errorf("inserting trace %s: %w", tr.TraceID, err)
	}

	// Remove any existing steps for idempotent upsert
	_, _ = tx.ExecContext(ctx, "DELETE FROM trace_steps WHERE trace_id = ?", tr.TraceID)

	stepQuery := `
	INSERT INTO trace_steps (
		trace_id, step_index, tool_name, tool_input, tool_output, duration_ms, tokens_used, error
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	stmt, err := tx.PrepareContext(ctx, stepQuery)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, step := range tr.Steps {
		_, err = stmt.ExecContext(ctx,
			tr.TraceID, step.StepIndex, step.ToolName, step.ToolInput, step.ToolOutput,
			step.DurationMs, step.TokensUsed, step.Error,
		)
		if err != nil {
			return fmt.Errorf("inserting step %d for trace %s: %w", step.StepIndex, tr.TraceID, err)
		}
	}

	return tx.Commit()
}

// InsertTracesBatch inserts a batch of traces in a single transaction.
func (db *DB) InsertTracesBatch(ctx context.Context, traces []model.Trace) error {
	for _, tr := range traces {
		if err := db.InsertTrace(ctx, tr); err != nil {
			return err
		}
	}
	return nil
}

// GetRecentTraces retrieves the most recent N traces for a given workflow, ordered ascending for chronological evaluation.
func (db *DB) GetRecentTraces(ctx context.Context, workflowID string, limit int) ([]model.Trace, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
	SELECT trace_id, workflow_id, session_id, model, timestamp, duration_ms,
	       status, step_count, prompt_tokens, completion_tokens, total_tokens, metadata_json
	FROM traces
	WHERE workflow_id = ?
	ORDER BY timestamp DESC
	LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, query, workflowID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying recent traces: %w", err)
	}
	defer rows.Close()

	var descTraces []model.Trace
	for rows.Next() {
		var tr model.Trace
		var metaStr sql.NullString
		var ts time.Time

		err := rows.Scan(
			&tr.TraceID, &tr.WorkflowID, &tr.SessionID, &tr.Model, &ts, &tr.DurationMs,
			&tr.Status, &tr.StepCount, &tr.TokenUsage.PromptTokens, &tr.TokenUsage.CompletionTokens,
			&tr.TokenUsage.TotalTokens, &metaStr,
		)
		if err != nil {
			return nil, err
		}
		tr.Timestamp = ts.UTC()
		if metaStr.Valid && metaStr.String != "" {
			_ = json.Unmarshal([]byte(metaStr.String), &tr.Metadata)
		}
		descTraces = append(descTraces, tr)
	}

	// Fetch steps for all returned traces
	for i := range descTraces {
		steps, err := db.getTraceSteps(ctx, descTraces[i].TraceID)
		if err != nil {
			return nil, err
		}
		descTraces[i].Steps = steps
	}

	// Reverse to chronological order (oldest to newest in the window)
	chronological := make([]model.Trace, len(descTraces))
	for i, tr := range descTraces {
		chronological[len(descTraces)-1-i] = tr
	}

	return chronological, nil
}

func (db *DB) getTraceSteps(ctx context.Context, traceID string) ([]model.TraceStep, error) {
	query := `
	SELECT step_index, tool_name, tool_input, tool_output, duration_ms, tokens_used, error
	FROM trace_steps
	WHERE trace_id = ?
	ORDER BY step_index ASC
	`

	rows, err := db.conn.QueryContext(ctx, query, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []model.TraceStep
	for rows.Next() {
		var s model.TraceStep
		var input, output, errStr sql.NullString
		err := rows.Scan(&s.StepIndex, &s.ToolName, &input, &output, &s.DurationMs, &s.TokensUsed, &errStr)
		if err != nil {
			return nil, err
		}
		if input.Valid {
			s.ToolInput = input.String
		}
		if output.Valid {
			s.ToolOutput = output.String
		}
		if errStr.Valid {
			s.Error = errStr.String
		}
		steps = append(steps, s)
	}
	return steps, nil
}

// GetTraceCount returns the total number of traces stored for a workflow.
func (db *DB) GetTraceCount(ctx context.Context, workflowID string) (int, error) {
	var count int
	err := db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM traces WHERE workflow_id = ?", workflowID).Scan(&count)
	return count, err
}

// SaveBaseline stores or replaces a Golden Baseline profile.
func (db *DB) SaveBaseline(ctx context.Context, b model.Baseline) error {
	stepSamplesJSON, _ := json.Marshal(b.StepSamples)
	matrixJSON, err := json.Marshal(b.MarkovMatrix)
	if err != nil {
		return err
	}
	toolsJSON, _ := json.Marshal(b.MonitoredTools)

	query := `
	INSERT INTO baselines (
		workflow_id, expected_steps_mean, expected_steps_std, sample_count,
		step_samples_json, markov_matrix_json, monitored_tools_json,
		max_allowed_wasserstein, max_tool_drift, max_loop_prob, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(workflow_id) DO UPDATE SET
		expected_steps_mean = excluded.expected_steps_mean,
		expected_steps_std = excluded.expected_steps_std,
		sample_count = excluded.sample_count,
		step_samples_json = excluded.step_samples_json,
		markov_matrix_json = excluded.markov_matrix_json,
		monitored_tools_json = excluded.monitored_tools_json,
		max_allowed_wasserstein = excluded.max_allowed_wasserstein,
		max_tool_drift = excluded.max_tool_drift,
		max_loop_prob = excluded.max_loop_prob,
		updated_at = excluded.updated_at
	`

	_, err = db.conn.ExecContext(ctx, query,
		b.WorkflowID, b.ExpectedStepsMean, b.ExpectedStepsStd, b.SampleCount,
		string(stepSamplesJSON), string(matrixJSON), string(toolsJSON),
		b.MaxAllowedWassersteinDist, b.MaxToolTransitionDrift, b.MaxLoopProbability,
		b.UpdatedAt.UTC(),
	)
	return err
}

// GetBaseline retrieves the Golden Baseline for a workflow if one exists.
func (db *DB) GetBaseline(ctx context.Context, workflowID string) (*model.Baseline, error) {
	query := `
	SELECT expected_steps_mean, expected_steps_std, sample_count,
	       step_samples_json, markov_matrix_json, monitored_tools_json,
	       max_allowed_wasserstein, max_tool_drift, max_loop_prob, updated_at
	FROM baselines
	WHERE workflow_id = ?
	`

	row := db.conn.QueryRowContext(ctx, query, workflowID)

	var b model.Baseline
	b.WorkflowID = workflowID

	var samplesJSON, matrixJSON, toolsJSON sql.NullString
	var updatedAt time.Time

	err := row.Scan(
		&b.ExpectedStepsMean, &b.ExpectedStepsStd, &b.SampleCount,
		&samplesJSON, &matrixJSON, &toolsJSON,
		&b.MaxAllowedWassersteinDist, &b.MaxToolTransitionDrift, &b.MaxLoopProbability,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	b.UpdatedAt = updatedAt.UTC()

	if samplesJSON.Valid && samplesJSON.String != "" {
		_ = json.Unmarshal([]byte(samplesJSON.String), &b.StepSamples)
	}
	if matrixJSON.Valid && matrixJSON.String != "" {
		_ = json.Unmarshal([]byte(matrixJSON.String), &b.MarkovMatrix)
	}
	if toolsJSON.Valid && toolsJSON.String != "" {
		_ = json.Unmarshal([]byte(toolsJSON.String), &b.MonitoredTools)
	}

	return &b, nil
}

// SaveEvaluation records a sentinel evaluation result.
func (db *DB) SaveEvaluation(ctx context.Context, eval model.EvaluationResult) error {
	resJSON, err := json.Marshal(eval)
	if err != nil {
		return err
	}

	query := `
	INSERT INTO evaluations (
		id, workflow_id, timestamp, window_size, wasserstein_dist,
		step_mean, step_std, matrix_deviation, status, explanation, result_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.conn.ExecContext(ctx, query,
		eval.ID, eval.WorkflowID, eval.Timestamp.UTC(), eval.WindowSize, eval.WassersteinDistance,
		eval.StepMean, eval.StepStd, eval.MatrixDeviation.MeanAbsDeviation, eval.Status,
		eval.Explanation, string(resJSON),
	)
	return err
}

// GetRecentEvaluations retrieves the latest evaluation results for a workflow.
func (db *DB) GetRecentEvaluations(ctx context.Context, workflowID string, limit int) ([]model.EvaluationResult, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `
	SELECT result_json
	FROM evaluations
	WHERE workflow_id = ?
	ORDER BY timestamp DESC
	LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, query, workflowID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var evals []model.EvaluationResult
	for rows.Next() {
		var resJSON string
		if err := rows.Scan(&resJSON); err != nil {
			return nil, err
		}
		var eval model.EvaluationResult
		if err := json.Unmarshal([]byte(resJSON), &eval); err == nil {
			evals = append(evals, eval)
		}
	}
	return evals, nil
}

// SaveIncident persists an anomaly incident.
func (db *DB) SaveIncident(ctx context.Context, inc model.Incident) error {
	traceIDsJSON, _ := json.Marshal(inc.TraceIDs)
	metricJSON, _ := json.Marshal(inc.MetricValues)
	offendingJSON, _ := json.Marshal(inc.OffendingTransitions)

	query := `
	INSERT INTO incidents (
		id, workflow_id, evaluation_id, severity, anomaly_type,
		title, summary, trace_ids_json, metric_values_json,
		offending_transitions_json, acknowledged, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO NOTHING
	`

	_, err := db.conn.ExecContext(ctx, query,
		inc.ID, inc.WorkflowID, inc.EvaluationID, inc.Severity, inc.AnomalyType,
		inc.Title, inc.Summary, string(traceIDsJSON), string(metricJSON),
		string(offendingJSON), inc.Acknowledged, inc.CreatedAt.UTC(),
	)
	return err
}

// GetRecentIncidents retrieves incidents for a workflow or across all workflows.
func (db *DB) GetRecentIncidents(ctx context.Context, workflowID string, limit int) ([]model.Incident, error) {
	if limit <= 0 {
		limit = 50
	}

	var query string
	var args []any

	if workflowID != "" {
		query = `
		SELECT id, workflow_id, evaluation_id, severity, anomaly_type,
		       title, summary, trace_ids_json, metric_values_json,
		       offending_transitions_json, acknowledged, created_at
		FROM incidents
		WHERE workflow_id = ?
		ORDER BY created_at DESC
		LIMIT ?
		`
		args = []any{workflowID, limit}
	} else {
		query = `
		SELECT id, workflow_id, evaluation_id, severity, anomaly_type,
		       title, summary, trace_ids_json, metric_values_json,
		       offending_transitions_json, acknowledged, created_at
		FROM incidents
		ORDER BY created_at DESC
		LIMIT ?
		`
		args = []any{limit}
	}

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []model.Incident
	for rows.Next() {
		var inc model.Incident
		var evalID, traceIDsJSON, metricJSON, offendingJSON sql.NullString
		var createdAt time.Time

		err := rows.Scan(
			&inc.ID, &inc.WorkflowID, &evalID, &inc.Severity, &inc.AnomalyType,
			&inc.Title, &inc.Summary, &traceIDsJSON, &metricJSON,
			&offendingJSON, &inc.Acknowledged, &createdAt,
		)
		if err != nil {
			return nil, err
		}
		if evalID.Valid {
			inc.EvaluationID = evalID.String
		}
		inc.CreatedAt = createdAt.UTC()

		if traceIDsJSON.Valid && traceIDsJSON.String != "" {
			_ = json.Unmarshal([]byte(traceIDsJSON.String), &inc.TraceIDs)
		}
		if metricJSON.Valid && metricJSON.String != "" {
			_ = json.Unmarshal([]byte(metricJSON.String), &inc.MetricValues)
		}
		if offendingJSON.Valid && offendingJSON.String != "" {
			_ = json.Unmarshal([]byte(offendingJSON.String), &inc.OffendingTransitions)
		}

		incidents = append(incidents, inc)
	}
	return incidents, nil
}

// AcknowledgeIncident marks an incident as acknowledged by an SRE.
func (db *DB) AcknowledgeIncident(ctx context.Context, incidentID string) error {
	query := `UPDATE incidents SET acknowledged = 1 WHERE id = ?`
	_, err := db.conn.ExecContext(ctx, query, incidentID)
	return err
}
