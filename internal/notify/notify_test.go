package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

func TestSlackClient_SendIncident(t *testing.T) {
	received := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewSlackClient(server.URL, "#test")
	inc := model.Incident{
		ID:          "inc-test",
		WorkflowID:  "wf-test",
		Severity:    "CRITICAL",
		AnomalyType: "TOOL_LOOP",
		Title:       "Test Loop Incident",
		Summary:     "Agent loop detected",
		CreatedAt:   time.Now(),
	}

	err := client.SendIncident(context.Background(), inc)
	if err != nil {
		t.Fatalf("SendIncident failed: %v", err)
	}
	if !received {
		t.Errorf("expected mock server to receive request")
	}
}
