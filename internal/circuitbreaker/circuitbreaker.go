package circuitbreaker

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

// State represents the operational state of the circuit breaker.
type State string

const (
	StateClosed   State = "CLOSED"    // Normal, traffic flowing through active model
	StateOpen     State = "OPEN"      // Tripped, fallback engaged
	StateHalfOpen State = "HALF_OPEN" // Testing recovery
)

// Config defines circuit breaker threshold parameters.
type Config struct {
	Enabled                 bool   `yaml:"enabled" json:"enabled"`
	CriticalConsecutiveRuns int    `yaml:"critical_consecutive_runs" json:"critical_consecutive_runs"`
	FallbackWebhook         string `yaml:"fallback_webhook" json:"fallback_webhook"`
	FallbackPayload         string `yaml:"fallback_payload" json:"fallback_payload"`
}

// Breaker tracks consecutive critical evaluations and coordinates automated self-healing rollbacks.
type Breaker struct {
	mu sync.RWMutex

	workflowID string
	config     Config
	httpClient *http.Client

	state             State
	consecutiveFaults int
	lastTrippedAt     time.Time
	lastTriggerReason string
}

// NewBreaker initializes a circuit breaker for a workflow.
func NewBreaker(workflowID string, cfg Config) *Breaker {
	if cfg.CriticalConsecutiveRuns <= 0 {
		cfg.CriticalConsecutiveRuns = 2
	}
	return &Breaker{
		workflowID: workflowID,
		config:     cfg,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		state:      StateClosed,
	}
}

// BreakerStatus represents the current status view model.
type BreakerStatus struct {
	WorkflowID        string    `json:"workflow_id"`
	Enabled           bool      `json:"enabled"`
	State             State     `json:"state"`
	ConsecutiveFaults int       `json:"consecutive_faults"`
	Threshold         int       `json:"threshold"`
	LastTrippedAt     time.Time `json:"last_tripped_at,omitempty"`
	LastTriggerReason string    `json:"last_trigger_reason,omitempty"`
}

// Status returns the current circuit breaker status.
func (b *Breaker) Status() BreakerStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return BreakerStatus{
		WorkflowID:        b.workflowID,
		Enabled:           b.config.Enabled,
		State:             b.state,
		ConsecutiveFaults: b.consecutiveFaults,
		Threshold:         b.config.CriticalConsecutiveRuns,
		LastTrippedAt:     b.lastTrippedAt,
		LastTriggerReason: b.lastTriggerReason,
	}
}

// RecordEvaluation checks an evaluation result and trips the breaker if threshold is reached.
func (b *Breaker) RecordEvaluation(ctx context.Context, eval model.EvaluationResult) (trippedNow bool, err error) {
	if !b.config.Enabled {
		return false, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if eval.Status == "CRITICAL" {
		b.consecutiveFaults++
		if b.consecutiveFaults >= b.config.CriticalConsecutiveRuns && b.state == StateClosed {
			b.state = StateOpen
			b.lastTrippedAt = time.Now().UTC()
			b.lastTriggerReason = eval.Explanation
			trippedNow = true

			// Dispatch fallback webhook asynchronously
			if b.config.FallbackWebhook != "" {
				go b.dispatchFallback(eval)
			}
			return true, nil
		}
	} else if eval.Status == "NORMAL" {
		if b.state == StateClosed {
			b.consecutiveFaults = 0
		}
	}

	return false, nil
}

// Reset manually restores the circuit breaker to CLOSED state.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.state = StateClosed
	b.consecutiveFaults = 0
	b.lastTriggerReason = ""
}

func (b *Breaker) dispatchFallback(eval model.EvaluationResult) {
	payload := b.config.FallbackPayload
	if payload == "" {
		payload = fmt.Sprintf(`{"workflow_id":"%s","action":"trigger_model_rollback","reason":"%s"}`,
			b.workflowID, eval.Explanation)
	}

	req, err := http.NewRequest(http.MethodPost, b.config.FallbackWebhook, bytes.NewReader([]byte(payload)))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "driftguard-sentinel/1.0")

	resp, err := b.httpClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}
