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

// SlackClient dispatches formatted notifications to a Slack Incoming Webhook.
type SlackClient struct {
	webhookURL string
	channel    string
	httpClient *http.Client
}

// NewSlackClient returns a new Slack client.
func NewSlackClient(webhookURL, channel string) *SlackClient {
	return &SlackClient{
		webhookURL: webhookURL,
		channel:    channel,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

type slackPayload struct {
	Channel     string       `json:"channel,omitempty"`
	Text        string       `json:"text"`
	Blocks      []slackBlock `json:"blocks,omitempty"`
	Attachments []slackAtt   `json:"attachments,omitempty"`
}

type slackBlock struct {
	Type string    `json:"type"`
	Text *slackTxt `json:"text,omitempty"`
}

type slackTxt struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type slackAtt struct {
	Color  string       `json:"color"`
	Blocks []slackBlock `json:"blocks"`
}

// SendIncident formats and delivers a rich incident alert to Slack.
func (s *SlackClient) SendIncident(ctx context.Context, inc model.Incident) error {
	if s.webhookURL == "" {
		return nil
	}

	color := "#f59e0b" // warning yellow
	if inc.Severity == "CRITICAL" {
		color = "#ef4444" // critical red
	}

	headerText := fmt.Sprintf("*[driftguard Sentinel Alert]* %s", inc.Title)
	summaryText := fmt.Sprintf("> *Workflow:* `%s`\n> *Anomaly Type:* `%s`\n> *Summary:* %s",
		inc.WorkflowID, inc.AnomalyType, inc.Summary)

	var metricsLine string
	if len(inc.MetricValues) > 0 {
		var parts []string
		for k, v := range inc.MetricValues {
			parts = append(parts, fmt.Sprintf("*%s:* `%.2f`", k, v))
		}
		metricsLine = fmt.Sprintf("📊 *Metrics:* %s", bytes.Join(bytesFields(parts), []byte(" | ")))
	}

	attBlocks := []slackBlock{
		{
			Type: "section",
			Text: &slackTxt{Type: "mrkdwn", Text: summaryText},
		},
	}

	if metricsLine != "" {
		attBlocks = append(attBlocks, slackBlock{
			Type: "section",
			Text: &slackTxt{Type: "mrkdwn", Text: metricsLine},
		})
	}

	payload := slackPayload{
		Channel: s.channel,
		Text:    fmt.Sprintf("[%s] %s", inc.Severity, inc.Title),
		Blocks: []slackBlock{
			{
				Type: "section",
				Text: &slackTxt{Type: "mrkdwn", Text: headerText},
			},
		},
		Attachments: []slackAtt{
			{
				Color:  color,
				Blocks: attBlocks,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dispatching slack alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}

func bytesFields(fields []string) [][]byte {
	out := make([][]byte, len(fields))
	for i, f := range fields {
		out[i] = []byte(f)
	}
	return out
}
