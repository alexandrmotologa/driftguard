package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("TEST_SLACK_URL", "https://hooks.slack.com/services/TEST/123")
	defer os.Unsetenv("TEST_SLACK_URL")

	yamlContent := `
version: "1"
server:
  listen: ":9090"
  sqlite_path: ":memory:"
workflows:
  - id: "test-agent"
    name: "Test Agent"
    rolling_window_runs: 40
    baseline:
      expected_steps_mean: 4.0
      expected_steps_std: 1.0
      max_allowed_wasserstein_distance: 1.5
      max_tool_transition_drift: 0.12
    tools_monitored:
      - "tool_a"
      - "tool_b"
    forbidden_transitions:
      - from: "tool_a"
        to: "tool_b"
alerting:
  slack:
    webhook_url: "${TEST_SLACK_URL}"
    channel: "#test"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "driftguard.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write tmp config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Server.Listen != ":9090" {
		t.Errorf("expected listen :9090, got %s", cfg.Server.Listen)
	}
	if len(cfg.Workflows) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(cfg.Workflows))
	}
	if cfg.Workflows[0].ID != "test-agent" {
		t.Errorf("expected id 'test-agent', got %s", cfg.Workflows[0].ID)
	}
	if cfg.Alerting.Slack.WebhookURL != "https://hooks.slack.com/services/TEST/123" {
		t.Errorf("env var expansion failed, got %s", cfg.Alerting.Slack.WebhookURL)
	}
}
