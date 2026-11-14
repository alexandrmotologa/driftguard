package metrics

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

func TestMetricsRegistry(t *testing.T) {
	reg := NewRegistry()
	reg.RecordTrace("test-wf")
	reg.RecordTrace("test-wf")

	eval := model.EvaluationResult{
		WorkflowID:          "test-wf",
		WassersteinDistance: 1.45,
		StepMean:            4.5,
		StepStd:             0.8,
		MatrixDeviation: model.MatrixPerturbation{
			MeanAbsDeviation: 0.05,
		},
	}
	reg.RecordEvaluation(eval)

	var buf bytes.Buffer
	reg.WritePrometheus(&buf)
	output := buf.String()

	if !strings.Contains(output, "driftguard_traces_total{workflow=\"test-wf\"} 2") {
		t.Errorf("expected traces total metric in output, got: %s", output)
	}
	if !strings.Contains(output, "driftguard_wasserstein_distance{workflow=\"test-wf\"} 1.4500") {
		t.Errorf("expected wasserstein distance metric in output, got: %s", output)
	}
	if !strings.Contains(output, "driftguard_step_mean{workflow=\"test-wf\"} 4.50") {
		t.Errorf("expected step mean metric in output, got: %s", output)
	}
}
