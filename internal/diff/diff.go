package diff

import (
	"fmt"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

// DiffStepType categorizes how an observed reasoning step deviates from the canonical baseline.
type DiffStepType string

const (
	DiffMatch       DiffStepType = "MATCH"       // Tool called as expected
	DiffLoop        DiffStepType = "LOOP"        // Tool repeated consecutively (reasoning cycle)
	DiffOmission    DiffStepType = "OMISSION"    // Canonical tool was skipped
	DiffInsertion   DiffStepType = "INSERTION"   // Novel or unexpected tool inserted
	DiffReplacement DiffStepType = "REPLACEMENT" // Substituted tool
)

// DiffStep represents an annotated step comparison.
type DiffStep struct {
	StepIndex    int          `json:"step_index"`
	ExpectedTool string       `json:"expected_tool,omitempty"`
	ObservedTool string       `json:"observed_tool,omitempty"`
	Type         DiffStepType `json:"type"`
	Explanation  string       `json:"explanation"`
}

// TrajectoryDiff provides a side-by-side execution trace diff against canonical baseline behavior.
type TrajectoryDiff struct {
	TraceID           string     `json:"trace_id"`
	WorkflowID        string     `json:"workflow_id"`
	CanonicalSteps    []string   `json:"canonical_steps"`
	ObservedSteps     []string   `json:"observed_steps"`
	StepDiffs         []DiffStep `json:"step_diffs"`
	TotalDeviations   int        `json:"total_deviations"`
	LoopCount         int        `json:"loop_count"`
	OmissionCount     int        `json:"omission_count"`
	HasGuardrailBypass bool      `json:"has_guardrail_bypass"`
}

// ComputeTrajectoryDiff compares an observed trace against canonical baseline tool expectations.
func ComputeTrajectoryDiff(tr model.Trace, canonicalTools []string, forbiddenTargets map[string]string) TrajectoryDiff {
	observed := make([]string, len(tr.Steps))
	for i, s := range tr.Steps {
		observed[i] = s.ToolName
	}

	diffResult := TrajectoryDiff{
		TraceID:        tr.TraceID,
		WorkflowID:     tr.WorkflowID,
		CanonicalSteps: canonicalTools,
		ObservedSteps:  observed,
		StepDiffs:      make([]DiffStep, 0),
	}

	// 1. Walk observed steps and compare
	observedLen := len(observed)
	canonicalLen := len(canonicalTools)

	// Track tool occurrences to detect loops
	var lastSeenTool string
	consecutiveCount := 0

	for i := 0; i < observedLen; i++ {
		currTool := observed[i]
		stepIdx := i + 1

		// Check if this tool transition bypasses a forbidden transition
		if forbiddenTo, exists := forbiddenTargets[currTool]; exists && i+1 < observedLen && observed[i+1] == forbiddenTo {
			diffResult.HasGuardrailBypass = true
		}

		if currTool == lastSeenTool {
			consecutiveCount++
			if consecutiveCount >= 2 {
				diffResult.LoopCount++
				diffResult.TotalDeviations++
				diffResult.StepDiffs = append(diffResult.StepDiffs, DiffStep{
					StepIndex:    stepIdx,
					ExpectedTool: "",
					ObservedTool: currTool,
					Type:         DiffLoop,
					Explanation:  fmt.Sprintf("Self-loop repetition: '%s' called %d times in sequence", currTool, consecutiveCount),
				})
				continue
			}
		} else {
			lastSeenTool = currTool
			consecutiveCount = 1
		}

		// Check if expected canonical tool matches
		if i < canonicalLen && canonicalTools[i] == currTool {
			diffResult.StepDiffs = append(diffResult.StepDiffs, DiffStep{
				StepIndex:    stepIdx,
				ExpectedTool: canonicalTools[i],
				ObservedTool: currTool,
				Type:         DiffMatch,
				Explanation:  "Matches canonical trajectory flow",
			})
			continue
		}

		// If tool is not in canonical sequence at all -> Insertion
		isInCanonical := false
		for _, ct := range canonicalTools {
			if ct == currTool {
				isInCanonical = true
				break
			}
		}

		if !isInCanonical {
			diffResult.TotalDeviations++
			diffResult.StepDiffs = append(diffResult.StepDiffs, DiffStep{
				StepIndex:    stepIdx,
				ExpectedTool: "",
				ObservedTool: currTool,
				Type:         DiffInsertion,
				Explanation:  fmt.Sprintf("Novel tool invocation absent from baseline: '%s'", currTool),
			})
		} else {
			diffResult.TotalDeviations++
			expected := ""
			if i < canonicalLen {
				expected = canonicalTools[i]
			}
			diffResult.StepDiffs = append(diffResult.StepDiffs, DiffStep{
				StepIndex:    stepIdx,
				ExpectedTool: expected,
				ObservedTool: currTool,
				Type:         DiffReplacement,
				Explanation:  fmt.Sprintf("Out-of-order execution: expected '%s', observed '%s'", expected, currTool),
			})
		}
	}

	// 2. Check for missing canonical tools
	toolObservedMap := make(map[string]bool)
	for _, ot := range observed {
		toolObservedMap[ot] = true
	}

	for _, ct := range canonicalTools {
		if !toolObservedMap[ct] {
			diffResult.OmissionCount++
			diffResult.TotalDeviations++
			diffResult.StepDiffs = append(diffResult.StepDiffs, DiffStep{
				StepIndex:    len(diffResult.StepDiffs) + 1,
				ExpectedTool: ct,
				ObservedTool: "",
				Type:         DiffOmission,
				Explanation:  fmt.Sprintf("Canonical tool omitted from trajectory: '%s'", ct),
			})
		}
	}

	return diffResult
}
