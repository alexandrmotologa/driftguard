package sentinel

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/config"
	"github.com/alexandrmotologa/driftguard/internal/model"
	"github.com/alexandrmotologa/driftguard/internal/stats"
	"github.com/google/uuid"
)

// EvaluateWindow runs the complete statistical and deterministic audit on a rolling window of traces.
func EvaluateWindow(
	workflowID string,
	traces []model.Trace,
	baseline *model.Baseline,
	cfg *config.WorkflowConfig,
) (model.EvaluationResult, *model.Incident) {
	evalID := fmt.Sprintf("eval-%s", uuid.New().String()[:8])
	now := time.Now().UTC()

	if len(traces) == 0 {
		return model.EvaluationResult{
			ID:          evalID,
			WorkflowID:  workflowID,
			Timestamp:   now,
			Status:      "NORMAL",
			Explanation: "No traces observed in the current evaluation window.",
		}, nil
	}

	// 1. Current step count distribution
	currStepCounts := make([]float64, len(traces))
	for i, tr := range traces {
		currStepCounts[i] = float64(tr.StepCount)
	}
	stepStats := stats.ComputeSummaryStats(currStepCounts)

	// 2. Prepare baseline samples for Wasserstein comparison
	var baseSamples []float64
	var maxWasserstein = 2.0
	var maxToolDrift = 0.18
	var maxLoopProb = 0.20

	if baseline != nil {
		if len(baseline.StepSamples) > 0 {
			baseSamples = baseline.StepSamples
		} else {
			baseSamples = stats.GenerateGaussianSamples(baseline.ExpectedStepsMean, baseline.ExpectedStepsStd, 100)
		}
		if baseline.MaxAllowedWassersteinDist > 0 {
			maxWasserstein = baseline.MaxAllowedWassersteinDist
		}
		if baseline.MaxToolTransitionDrift > 0 {
			maxToolDrift = baseline.MaxToolTransitionDrift
		}
		if baseline.MaxLoopProbability > 0 {
			maxLoopProb = baseline.MaxLoopProbability
		}
	} else if cfg != nil {
		baseSamples = stats.GenerateGaussianSamples(cfg.Baseline.ExpectedStepsMean, cfg.Baseline.ExpectedStepsStd, 100)
		if cfg.Baseline.MaxAllowedWassersteinDist > 0 {
			maxWasserstein = cfg.Baseline.MaxAllowedWassersteinDist
		}
		if cfg.Baseline.MaxToolTransitionDrift > 0 {
			maxToolDrift = cfg.Baseline.MaxToolTransitionDrift
		}
		if cfg.Baseline.MaxLoopProbability > 0 {
			maxLoopProb = cfg.Baseline.MaxLoopProbability
		}
	} else {
		baseSamples = currStepCounts
	}

	// 3. Compute 1D Wasserstein distance
	wassersteinDist := stats.ComputeWasserstein1D(baseSamples, currStepCounts)
	wassersteinDist = math.Round(wassersteinDist*1000) / 1000

	// 4. Markov matrix transition analysis
	var monitoredTools []string
	if baseline != nil && len(baseline.MonitoredTools) > 0 {
		monitoredTools = baseline.MonitoredTools
	} else if cfg != nil && len(cfg.ToolsMonitored) > 0 {
		monitoredTools = cfg.ToolsMonitored
	}

	currMatrix := stats.BuildMarkovMatrix(traces, monitoredTools, 0.01)

	var perturbation model.MatrixPerturbation
	var loops []model.LoopAnomaly
	if baseline != nil && len(baseline.MarkovMatrix.States) > 0 {
		perturbation, loops = stats.CompareMarkovMatrices(baseline.MarkovMatrix, currMatrix, maxLoopProb)
	}

	// 5. Deterministic rule checks (forbidden transitions & required tools)
	var allViolations []model.RuleViolation
	if cfg != nil {
		for _, tr := range traces {
			v := CheckTraceRules(tr, cfg.ForbiddenTransitions, cfg.RequiredTools)
			if len(v) > 0 {
				allViolations = append(allViolations, v...)
			}
		}
	}

	// 6. Step histogram comparison
	histogram := stats.BuildStepHistogram(baseSamples, currStepCounts)

	// 7. Status evaluation and incident qualification
	status := "NORMAL"
	var anomalies []string
	anomalyType := ""
	severity := "NORMAL"

	if len(allViolations) > 0 {
		status = "CRITICAL"
		severity = "CRITICAL"
		anomalyType = "FORBIDDEN_TRANSITION"
		anomalies = append(anomalies, fmt.Sprintf("%d forbidden guardrail transitions detected", len(allViolations)))
	}

	if wassersteinDist > maxWasserstein*1.4 {
		status = "CRITICAL"
		if severity != "CRITICAL" {
			severity = "CRITICAL"
			anomalyType = "STEP_EXPLOSION"
		}
		anomalies = append(anomalies, fmt.Sprintf("Critical step count explosion: Wasserstein distance %0.2f exceeds critical threshold %0.2f", wassersteinDist, maxWasserstein*1.4))
	} else if wassersteinDist > maxWasserstein {
		if status == "NORMAL" {
			status = "WARNING"
			severity = "WARNING"
			anomalyType = "STEP_EXPLOSION"
		}
		anomalies = append(anomalies, fmt.Sprintf("Step count distribution drift: Wasserstein distance %0.2f exceeds threshold %0.2f", wassersteinDist, maxWasserstein))
	}

	if len(loops) > 0 {
		for _, l := range loops {
			if l.Probability > maxLoopProb*1.5 {
				status = "CRITICAL"
				if severity != "CRITICAL" {
					severity = "CRITICAL"
					anomalyType = "TOOL_LOOP"
				}
				anomalies = append(anomalies, fmt.Sprintf("Severe tool self-loop: '%s' triggered with %0.1f%% probability (%d times)", l.ToolName, l.Probability*100, l.Occurrences))
			} else {
				if status == "NORMAL" {
					status = "WARNING"
					severity = "WARNING"
					anomalyType = "TOOL_LOOP"
				}
				anomalies = append(anomalies, fmt.Sprintf("Tool self-loop elevated: '%s' at %0.1f%% probability", l.ToolName, l.Probability*100))
			}
		}
	}

	if perturbation.MeanAbsDeviation > maxToolDrift {
		if status == "NORMAL" {
			status = "WARNING"
			severity = "WARNING"
			anomalyType = "MATRIX_PERTURBATION"
		}
		anomalies = append(anomalies, fmt.Sprintf("Tool transition matrix perturbed by %0.3f (max allowed %0.3f)", perturbation.MeanAbsDeviation, maxToolDrift))
	}

	if len(perturbation.NovelTransitions) > 0 {
		if status == "NORMAL" {
			status = "WARNING"
			severity = "WARNING"
			anomalyType = "NOVEL_TRANSITIONS"
		}
		anomalies = append(anomalies, fmt.Sprintf("%d novel tool transitions observed that were absent in baseline", len(perturbation.NovelTransitions)))
	}

	var explanation string
	if len(anomalies) == 0 {
		explanation = fmt.Sprintf("Trajectory distribution is stable across %d runs. Wasserstein distance is %0.2f (threshold %0.2f).", len(traces), wassersteinDist, maxWasserstein)
	} else {
		explanation = strings.Join(anomalies, "; ")
	}

	result := model.EvaluationResult{
		ID:                  evalID,
		WorkflowID:          workflowID,
		Timestamp:           now,
		WindowSize:          len(traces),
		WassersteinDistance: wassersteinDist,
		StepMean:            stepStats.Mean,
		StepStd:             stepStats.Std,
		StepHistogram:       histogram,
		MatrixDeviation:     perturbation,
		CurrentMatrix:       currMatrix,
		Status:              status,
		LoopAnomalies:       loops,
		NovelTransitions:    perturbation.NovelTransitions,
		RuleViolations:      allViolations,
		Explanation:         explanation,
	}

	var incident *model.Incident
	if status == "WARNING" || status == "CRITICAL" {
		traceIDs := make([]string, 0, len(traces))
		for _, tr := range traces {
			traceIDs = append(traceIDs, tr.TraceID)
		}
		if len(traceIDs) > 10 {
			traceIDs = traceIDs[len(traceIDs)-10:]
		}

		offending := make([]model.TransitionEdge, 0)
		for _, l := range loops {
			offending = append(offending, model.TransitionEdge{
				From:        l.ToolName,
				To:          l.ToolName,
				CurrentProb: l.Probability,
				Count:       l.Occurrences,
			})
		}
		for _, v := range allViolations {
			offending = append(offending, model.TransitionEdge{
				From:  v.FromTool,
				To:    v.ToTool,
				Count: 1,
			})
		}

		incident = &model.Incident{
			ID:           fmt.Sprintf("inc-%s", uuid.New().String()[:8]),
			WorkflowID:   workflowID,
			EvaluationID: evalID,
			Severity:     severity,
			AnomalyType:  anomalyType,
			Title:        fmt.Sprintf("[%s] %s in %s", severity, anomalyType, workflowID),
			Summary:      explanation,
			TraceIDs:     traceIDs,
			MetricValues: map[string]float64{
				"wasserstein_distance": wassersteinDist,
				"matrix_deviation":     perturbation.MeanAbsDeviation,
				"step_mean":            stepStats.Mean,
				"step_std":             stepStats.Std,
			},
			OffendingTransitions: offending,
			Acknowledged:         false,
			CreatedAt:            now,
		}
	}

	return result, incident
}
