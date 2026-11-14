package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level driftguard configuration.
type Config struct {
	Version   string            `yaml:"version"`
	Server    ServerConfig      `yaml:"server"`
	Workflows []WorkflowConfig  `yaml:"workflows"`
	Alerting  AlertingConfig    `yaml:"alerting"`
}

// ServerConfig holds HTTP and database settings.
type ServerConfig struct {
	Listen           string `yaml:"listen"`
	SQLitePath       string `yaml:"sqlite_path"`
	EvalIntervalSec  int    `yaml:"eval_interval_sec"`
	SSEKeepaliveSec  int    `yaml:"sse_keepalive_sec"`
}

// WorkflowConfig defines monitoring rules for a specific AI agent workflow.
type WorkflowConfig struct {
	ID                   string                 `yaml:"id"`
	Name                 string                 `yaml:"name"`
	RollingWindowRuns    int                    `yaml:"rolling_window_runs"`
	Baseline             BaselineConfig         `yaml:"baseline"`
	ToolsMonitored       []string               `yaml:"tools_monitored"`
	ForbiddenTransitions []ForbiddenTransition  `yaml:"forbidden_transitions"`
	RequiredTools        []string               `yaml:"required_tools,omitempty"`
}

// BaselineConfig defines the golden baseline statistical thresholds.
type BaselineConfig struct {
	ExpectedStepsMean           float64 `yaml:"expected_steps_mean"`
	ExpectedStepsStd            float64 `yaml:"expected_steps_std"`
	MaxAllowedWassersteinDist   float64 `yaml:"max_allowed_wasserstein_distance"`
	MaxToolTransitionDrift      float64 `yaml:"max_tool_transition_drift"`
	MaxLoopProbability          float64 `yaml:"max_loop_probability,omitempty"`
}

// ForbiddenTransition specifies directed tool transitions that represent security or business logic violations.
type ForbiddenTransition struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// AlertingConfig defines external alert sinks.
type AlertingConfig struct {
	Slack     SlackConfig     `yaml:"slack"`
	PagerDuty PagerDutyConfig `yaml:"pagerduty"`
}

// SlackConfig holds incoming webhook details.
type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
}

// PagerDutyConfig holds PagerDuty Events v2 details.
type PagerDutyConfig struct {
	RoutingKey string `yaml:"routing_key"`
	Severity   string `yaml:"severity"`
}

// LoadConfig reads and parses a YAML configuration file with environment variable expansion.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	expanded := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config yaml: %w", err)
	}

	// Apply defaults
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":8080"
	}
	if cfg.Server.SQLitePath == "" {
		cfg.Server.SQLitePath = "./driftguard.db"
	}
	if cfg.Server.EvalIntervalSec <= 0 {
		cfg.Server.EvalIntervalSec = 15
	}
	if cfg.Server.SSEKeepaliveSec <= 0 {
		cfg.Server.SSEKeepaliveSec = 15
	}

	for i := range cfg.Workflows {
		wf := &cfg.Workflows[i]
		if wf.RollingWindowRuns <= 0 {
			wf.RollingWindowRuns = 50
		}
		if wf.Baseline.MaxAllowedWassersteinDist <= 0 {
			wf.Baseline.MaxAllowedWassersteinDist = 2.0
		}
		if wf.Baseline.MaxToolTransitionDrift <= 0 {
			wf.Baseline.MaxToolTransitionDrift = 0.20
		}
		if wf.Baseline.MaxLoopProbability <= 0 {
			wf.Baseline.MaxLoopProbability = 0.25
		}
	}

	return &cfg, nil
}

// expandEnvVars expands ${VAR} or $VAR patterns in the YAML string.
var envVarRegex = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}|\$([a-zA-Z_][a-zA-Z0-9_]*)`)

func expandEnvVars(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		var varName string
		if strings.HasPrefix(match, "${") && strings.HasSuffix(match, "}") {
			varName = match[2 : len(match)-1]
		} else {
			varName = match[1:]
		}
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return match
	})
}
