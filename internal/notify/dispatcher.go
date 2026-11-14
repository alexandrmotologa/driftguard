package notify

import (
	"context"
	"sync"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/config"
	"github.com/alexandrmotologa/driftguard/internal/model"
)

// Dispatcher coordinates multi-channel incident alerts with deduplication.
type Dispatcher struct {
	slack     *SlackClient
	pagerduty *PagerDutyClient

	mu        sync.Mutex
	lastAlert map[string]time.Time
}

// NewDispatcher creates a new alert dispatcher from configuration.
func NewDispatcher(cfg config.AlertingConfig) *Dispatcher {
	return &Dispatcher{
		slack:     NewSlackClient(cfg.Slack.WebhookURL, cfg.Slack.Channel),
		pagerduty: NewPagerDutyClient(cfg.PagerDuty.RoutingKey, cfg.PagerDuty.Severity),
		lastAlert: make(map[string]time.Time),
	}
}

// Dispatch dispatches an incident alert to all configured sinks with a 5-minute deduplication window.
func (d *Dispatcher) Dispatch(ctx context.Context, inc model.Incident) {
	key := inc.WorkflowID + ":" + inc.AnomalyType

	d.mu.Lock()
	lastTime, exists := d.lastAlert[key]
	if exists && time.Since(lastTime) < 5*time.Minute {
		d.mu.Unlock()
		return
	}
	d.lastAlert[key] = time.Now()
	d.mu.Unlock()

	// Dispatch concurrently
	var wg sync.WaitGroup

	if d.slack != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = d.slack.SendIncident(ctx, inc)
		}()
	}

	if d.pagerduty != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = d.pagerduty.SendIncident(ctx, inc)
		}()
	}

	wg.Wait()
}
