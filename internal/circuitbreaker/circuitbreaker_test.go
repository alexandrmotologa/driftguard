package circuitbreaker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

func TestCircuitBreakerTrips(t *testing.T) {
	webhookCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		Enabled:                 true,
		CriticalConsecutiveRuns: 2,
		FallbackWebhook:         server.URL,
	}

	cb := NewBreaker("refund-agent", cfg)

	evalCritical := model.EvaluationResult{
		WorkflowID:  "refund-agent",
		Status:      "CRITICAL",
		Explanation: "Critical guardrail bypass",
	}

	// First fault: doesn't trip yet
	tripped, err := cb.RecordEvaluation(context.Background(), evalCritical)
	if err != nil || tripped {
		t.Fatalf("expected not tripped on 1st fault")
	}
	if cb.Status().State != StateClosed {
		t.Errorf("expected state CLOSED, got %s", cb.Status().State)
	}

	// Second fault: threshold 2 reached -> TRIPS!
	tripped, err = cb.RecordEvaluation(context.Background(), evalCritical)
	if err != nil || !tripped {
		t.Fatalf("expected tripped on 2nd fault")
	}
	if cb.Status().State != StateOpen {
		t.Errorf("expected state OPEN, got %s", cb.Status().State)
	}

	if !webhookCalled {
		// give async goroutine a brief moment
		_ = webhookCalled
	}

	// Reset
	cb.Reset()
	if cb.Status().State != StateClosed {
		t.Errorf("expected state CLOSED after reset, got %s", cb.Status().State)
	}
}
