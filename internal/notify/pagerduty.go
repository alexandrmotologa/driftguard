package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

// PagerDutyClient dispatches events to PagerDuty Events API v2.
type PagerDutyClient struct {
	routingKey string
	severity   string
	httpClient *http.Client
}

// NewPagerDutyClient returns a new PagerDuty Events v2 client.
func NewPagerDutyClient(routingKey, severity string) *PagerDutyClient {
	if severity == "" {
		severity = "error"
	}
	return &PagerDutyClient{
		routingKey: routingKey,
		severity:   severity,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

type pdPayload struct {
	RoutingKey  string    `json:"routing_key"`
	EventAction string    `json:"event_action"`
	DedupKey    string    `json:"dedup_key"`
	Payload     pdDetails `json:"payload"`
}

type pdDetails struct {
	Summary       string         `json:"summary"`
	Source        string         `json:"source"`
	Severity      string         `json:"severity"`
	CustomDetails map[string]any `json:"custom_details,omitempty"`
}

// SendIncident sends an incident trigger to PagerDuty.
func (pd *PagerDutyClient) SendIncident(ctx context.Context, inc model.Incident) error {
	if pd.routingKey == "" {
		return nil
	}

	dedupKey := fmt.Sprintf("driftguard-%s-%s", inc.WorkflowID, inc.AnomalyType)
	sev := pd.severity
	if inc.Severity == "WARNING" {
		sev = "warning"
	} else if inc.Severity == "CRITICAL" {
		sev = "critical"
	}

	details := map[string]any{
		"workflow_id":   inc.WorkflowID,
		"anomaly_type":  inc.AnomalyType,
		"evaluation_id": inc.EvaluationID,
		"metrics":       inc.MetricValues,
		"offending":     inc.OffendingTransitions,
	}

	event := pdPayload{
		RoutingKey:  pd.routingKey,
		EventAction: "trigger",
		DedupKey:    dedupKey,
		Payload: pdDetails{
			Summary:       fmt.Sprintf("%s: %s", inc.Title, inc.Summary),
			Source:        "driftguard-sentinel",
			Severity:      sev,
			CustomDetails: details,
		},
	}

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://events.pagerduty.com/v2/enqueue", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := pd.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dispatching pagerduty event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pagerduty returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}
