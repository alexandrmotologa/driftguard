package simulator

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/model"
	"github.com/google/uuid"
)

// Scenario represents the simulated agent behavior pattern.
type Scenario string

const (
	ScenarioClean       Scenario = "clean"
	ScenarioLoop        Scenario = "loop"
	ScenarioBypass      Scenario = "bypass"
	ScenarioHallucinate Scenario = "hallucinate"
)

// Generator creates realistic agent execution trajectories.
type Generator struct {
	rng *rand.Rand
}

// NewGenerator returns a new deterministic or random trace generator.
func NewGenerator(seed int64) *Generator {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &Generator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// GenerateTrace synthesizes a trace for a given workflow and scenario.
func (g *Generator) GenerateTrace(workflowID string, scenario Scenario) model.Trace {
	now := time.Now().UTC()
	traceID := fmt.Sprintf("tr-%s", uuid.New().String()[:12])
	sessionID := fmt.Sprintf("sess-%d", g.rng.Intn(9000)+1000)

	var steps []model.TraceStep
	modelName := "gpt-4o-2024-08-06"

	switch scenario {
	case ScenarioLoop:
		// Model update triggered reasoning loop on search or verify tool
		modelName = "gpt-4o-checkpoint-update"
		steps = append(steps, model.TraceStep{
			StepIndex:  1,
			ToolName:   "lookup_order",
			DurationMs: int64(g.rng.Intn(150) + 200),
			TokensUsed: g.rng.Intn(50) + 120,
		})

		loopCount := g.rng.Intn(4) + 3 // 3 to 6 repetitive loops
		for i := 0; i < loopCount; i++ {
			steps = append(steps, model.TraceStep{
				StepIndex:  len(steps) + 1,
				ToolName:   "lookup_order",
				DurationMs: int64(g.rng.Intn(100) + 150),
				TokensUsed: g.rng.Intn(40) + 80,
			})
		}

		steps = append(steps,
			model.TraceStep{
				StepIndex:  len(steps) + 1,
				ToolName:   "check_fraud_score",
				DurationMs: int64(g.rng.Intn(100) + 150),
				TokensUsed: g.rng.Intn(30) + 100,
			},
			model.TraceStep{
				StepIndex:  len(steps) + 2,
				ToolName:   "execute_refund",
				DurationMs: int64(g.rng.Intn(200) + 400),
				TokensUsed: g.rng.Intn(60) + 180,
			},
			model.TraceStep{
				StepIndex:  len(steps) + 3,
				ToolName:   "send_confirmation_email",
				DurationMs: int64(g.rng.Intn(100) + 200),
				TokensUsed: g.rng.Intn(40) + 140,
			},
		)

	case ScenarioBypass:
		// Direct transition from lookup_order to execute_refund (skipping check_fraud_score)
		modelName = "claude-3-5-sonnet-regressed"
		steps = []model.TraceStep{
			{
				StepIndex:  1,
				ToolName:   "lookup_order",
				DurationMs: int64(g.rng.Intn(100) + 220),
				TokensUsed: g.rng.Intn(40) + 130,
			},
			{
				StepIndex:  2,
				ToolName:   "execute_refund",
				DurationMs: int64(g.rng.Intn(200) + 450),
				TokensUsed: g.rng.Intn(50) + 160,
			},
			{
				StepIndex:  3,
				ToolName:   "send_confirmation_email",
				DurationMs: int64(g.rng.Intn(100) + 250),
				TokensUsed: g.rng.Intn(40) + 120,
			},
		}

	case ScenarioHallucinate:
		// Agent attempts calling an unmonitored / hallucinated tool
		modelName = "gemini-1.5-pro-preview"
		steps = []model.TraceStep{
			{
				StepIndex:  1,
				ToolName:   "lookup_order",
				DurationMs: int64(g.rng.Intn(100) + 220),
				TokensUsed: g.rng.Intn(40) + 130,
			},
			{
				StepIndex:  2,
				ToolName:   "auto_resolve_v2",
				DurationMs: int64(g.rng.Intn(100) + 150),
				TokensUsed: g.rng.Intn(30) + 90,
				Error:      "ToolNotFoundException: tool auto_resolve_v2 does not exist",
			},
			{
				StepIndex:  3,
				ToolName:   "check_fraud_score",
				DurationMs: int64(g.rng.Intn(100) + 180),
				TokensUsed: g.rng.Intn(30) + 110,
			},
			{
				StepIndex:  4,
				ToolName:   "execute_refund",
				DurationMs: int64(g.rng.Intn(200) + 400),
				TokensUsed: g.rng.Intn(50) + 170,
			},
			{
				StepIndex:  5,
				ToolName:   "send_confirmation_email",
				DurationMs: int64(g.rng.Intn(100) + 200),
				TokensUsed: g.rng.Intn(40) + 130,
			},
		}

	default: // ScenarioClean
		steps = []model.TraceStep{
			{
				StepIndex:  1,
				ToolName:   "lookup_order",
				DurationMs: int64(g.rng.Intn(120) + 220),
				TokensUsed: g.rng.Intn(40) + 140,
			},
			{
				StepIndex:  2,
				ToolName:   "check_fraud_score",
				DurationMs: int64(g.rng.Intn(80) + 160),
				TokensUsed: g.rng.Intn(30) + 110,
			},
			{
				StepIndex:  3,
				ToolName:   "execute_refund",
				DurationMs: int64(g.rng.Intn(180) + 420),
				TokensUsed: g.rng.Intn(50) + 180,
			},
			{
				StepIndex:  4,
				ToolName:   "send_confirmation_email",
				DurationMs: int64(g.rng.Intn(100) + 210),
				TokensUsed: g.rng.Intn(30) + 120,
			},
		}
	}

	var totalDuration int64
	var promptTokens, completionTokens int
	for _, s := range steps {
		totalDuration += s.DurationMs
		completionTokens += s.TokensUsed
	}
	promptTokens = g.rng.Intn(400) + 1000

	return model.Trace{
		WorkflowID: workflowID,
		TraceID:    traceID,
		SessionID:  sessionID,
		Model:      modelName,
		Timestamp:  now,
		DurationMs: totalDuration,
		Status:     "success",
		StepCount:  len(steps),
		Steps:      steps,
		TokenUsage: model.TokenUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
		Metadata: map[string]string{
			"scenario": string(scenario),
			"env":      "production",
		},
	}
}

// GenerateBatch produces an array of simulated traces.
func (g *Generator) GenerateBatch(workflowID string, scenario Scenario, count int) []model.Trace {
	if count <= 0 {
		count = 10
	}
	traces := make([]model.Trace, count)
	for i := 0; i < count; i++ {
		traces[i] = g.GenerateTrace(workflowID, scenario)
	}
	return traces
}
