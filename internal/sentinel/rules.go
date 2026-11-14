package sentinel

import (
	"fmt"

	"github.com/alexandrmotologa/driftguard/internal/config"
	"github.com/alexandrmotologa/driftguard/internal/model"
)

// CheckTraceRules inspects an individual trace for forbidden transitions and missing required tools.
func CheckTraceRules(tr model.Trace, forbidden []config.ForbiddenTransition, required []string) []model.RuleViolation {
	var violations []model.RuleViolation

	// 1. Check direct forbidden transitions between consecutive steps
	if len(tr.Steps) >= 2 && len(forbidden) > 0 {
		forbiddenMap := make(map[string]map[string]struct{})
		for _, f := range forbidden {
			if _, ok := forbiddenMap[f.From]; !ok {
				forbiddenMap[f.From] = make(map[string]struct{})
			}
			forbiddenMap[f.From][f.To] = struct{}{}
		}

		for i := 0; i < len(tr.Steps)-1; i++ {
			from := tr.Steps[i].ToolName
			to := tr.Steps[i+1].ToolName
			if targets, ok := forbiddenMap[from]; ok {
				if _, hit := targets[to]; hit {
					violations = append(violations, model.RuleViolation{
						TraceID:   tr.TraceID,
						FromTool:  from,
						ToTool:    to,
						StepIndex: i + 1,
						Reason:    fmt.Sprintf("Forbidden transition: direct edge from '%s' to '%s' violates guardrail policy", from, to),
					})
				}
			}
		}
	}

	// 2. Check required tools
	if len(required) > 0 {
		toolExecuted := make(map[string]bool)
		for _, s := range tr.Steps {
			toolExecuted[s.ToolName] = true
		}
		for _, req := range required {
			if !toolExecuted[req] {
				violations = append(violations, model.RuleViolation{
					TraceID:   tr.TraceID,
					FromTool:  "",
					ToTool:    req,
					StepIndex: len(tr.Steps),
					Reason:    fmt.Sprintf("Missing required security or compliance tool: '%s'", req),
				})
			}
		}
	}

	return violations
}
