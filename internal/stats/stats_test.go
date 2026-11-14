package stats

import (
	"math"
	"testing"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

func TestComputeWasserstein1D(t *testing.T) {
	// Identical distributions -> distance 0
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{1, 2, 3, 4, 5}
	dist := ComputeWasserstein1D(x, y)
	if dist != 0 {
		t.Errorf("expected 0 for identical distributions, got %f", dist)
	}

	// Shifted by constant 2
	shifted := []float64{3, 4, 5, 6, 7}
	distShifted := ComputeWasserstein1D(x, shifted)
	if math.Abs(distShifted-2.0) > 1e-6 {
		t.Errorf("expected distance 2.0, got %f", distShifted)
	}

	// Empty inputs
	if distEmpty := ComputeWasserstein1D(nil, x); distEmpty != 0 {
		t.Errorf("expected 0 for empty distribution, got %f", distEmpty)
	}
}

func TestBuildMarkovMatrix(t *testing.T) {
	traces := []model.Trace{
		{
			WorkflowID: "wf-1",
			Steps: []model.TraceStep{
				{ToolName: "lookup"},
				{ToolName: "verify"},
				{ToolName: "execute"},
			},
		},
		{
			WorkflowID: "wf-1",
			Steps: []model.TraceStep{
				{ToolName: "lookup"},
				{ToolName: "verify"},
				{ToolName: "execute"},
			},
		},
	}

	matrix := BuildMarkovMatrix(traces, []string{"lookup", "verify", "execute"}, 0.01)
	if len(matrix.States) != 5 { // __START__, lookup, verify, execute, __END__
		t.Fatalf("expected 5 states, got %d", len(matrix.States))
	}

	if matrix.RawCounts[StateStart]["lookup"] != 2 {
		t.Errorf("expected 2 transitions from START to lookup, got %d", matrix.RawCounts[StateStart]["lookup"])
	}
	if matrix.RawCounts["verify"]["execute"] != 2 {
		t.Errorf("expected 2 transitions from verify to execute, got %d", matrix.RawCounts["verify"]["execute"])
	}
	if matrix.RawCounts["execute"][StateEnd] != 2 {
		t.Errorf("expected 2 transitions from execute to END, got %d", matrix.RawCounts["execute"][StateEnd])
	}
}

func TestCompareMarkovMatrices(t *testing.T) {
	cleanTraces := []model.Trace{
		{
			Steps: []model.TraceStep{
				{ToolName: "lookup"},
				{ToolName: "verify"},
				{ToolName: "execute"},
			},
		},
	}
	loopingTraces := []model.Trace{
		{
			Steps: []model.TraceStep{
				{ToolName: "lookup"},
				{ToolName: "lookup"},
				{ToolName: "lookup"},
				{ToolName: "verify"},
				{ToolName: "execute"},
			},
		},
	}

	baseMatrix := BuildMarkovMatrix(cleanTraces, []string{"lookup", "verify", "execute"}, 0.01)
	currMatrix := BuildMarkovMatrix(loopingTraces, []string{"lookup", "verify", "execute"}, 0.01)

	pert, loops := CompareMarkovMatrices(baseMatrix, currMatrix, 0.20)
	if pert.MeanAbsDeviation <= 0 {
		t.Errorf("expected positive matrix perturbation, got %f", pert.MeanAbsDeviation)
	}

	foundLoop := false
	for _, l := range loops {
		if l.ToolName == "lookup" {
			foundLoop = true
			break
		}
	}
	if !foundLoop {
		t.Errorf("expected self-loop anomaly for lookup tool, got %v", loops)
	}
}

func TestComputeShannonEntropy(t *testing.T) {
	countsUniform := map[string]int{"a": 10, "b": 10, "c": 10, "d": 10}
	entropy := ComputeShannonEntropy(countsUniform)
	// For 4 equal states, H = log2(4) = 2.0
	if math.Abs(entropy-2.0) > 1e-2 {
		t.Errorf("expected entropy ~2.0, got %f", entropy)
	}

	countsDeterministic := map[string]int{"a": 100}
	if h := ComputeShannonEntropy(countsDeterministic); h != 0 {
		t.Errorf("expected entropy 0 for single deterministic state, got %f", h)
	}
}
