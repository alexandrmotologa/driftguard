package sentinel

import (
	"fmt"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/config"
	"github.com/alexandrmotologa/driftguard/internal/model"
	"github.com/alexandrmotologa/driftguard/internal/stats"
)

// CalibrateFromTraces analyzes clean historical traces to generate a verified Golden Baseline.
func CalibrateFromTraces(workflowID string, traces []model.Trace, cfg *config.WorkflowConfig) (*model.Baseline, error) {
	if len(traces) == 0 {
		return nil, fmt.Errorf("cannot calibrate baseline from empty trace list")
	}

	stepCounts := make([]float64, len(traces))
	for i, tr := range traces {
		stepCounts[i] = float64(tr.StepCount)
	}

	summary := stats.ComputeSummaryStats(stepCounts)

	// Determine monitored tools
	toolSet := make(map[string]struct{})
	if cfg != nil {
		for _, t := range cfg.ToolsMonitored {
			toolSet[t] = struct{}{}
		}
	}
	for _, tr := range traces {
		for _, s := range tr.Steps {
			if s.ToolName != "" {
				toolSet[s.ToolName] = struct{}{}
			}
		}
	}

	monitoredTools := make([]string, 0, len(toolSet))
	for t := range toolSet {
		monitoredTools = append(monitoredTools, t)
	}

	// Build Golden Markov Matrix
	markovMatrix := stats.BuildMarkovMatrix(traces, monitoredTools, 0.01)

	// Determine thresholds
	maxWasserstein := 2.0
	maxToolDrift := 0.18
	maxLoopProb := 0.20

	if cfg != nil {
		if cfg.Baseline.MaxAllowedWassersteinDist > 0 {
			maxWasserstein = cfg.Baseline.MaxAllowedWassersteinDist
		}
		if cfg.Baseline.MaxToolTransitionDrift > 0 {
			maxToolDrift = cfg.Baseline.MaxToolTransitionDrift
		}
		if cfg.Baseline.MaxLoopProbability > 0 {
			maxLoopProb = cfg.Baseline.MaxLoopProbability
		}
	}

	baseline := &model.Baseline{
		WorkflowID:                workflowID,
		ExpectedStepsMean:         summary.Mean,
		ExpectedStepsStd:          summary.Std,
		StepSamples:               stepCounts,
		MarkovMatrix:              markovMatrix,
		MonitoredTools:            monitoredTools,
		MaxAllowedWassersteinDist: maxWasserstein,
		MaxToolTransitionDrift:    maxToolDrift,
		MaxLoopProbability:        maxLoopProb,
		UpdatedAt:                 time.Now().UTC(),
		SampleCount:               len(traces),
	}

	return baseline, nil
}
