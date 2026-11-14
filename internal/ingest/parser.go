package ingest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/model"
	"github.com/google/uuid"
)

// ParseSingleTrace parses a single JSON trace payload.
func ParseSingleTrace(r io.Reader) (*model.Trace, error) {
	var tr model.Trace
	if err := json.NewDecoder(r).Decode(&tr); err != nil {
		return nil, fmt.Errorf("decoding trace json: %w", err)
	}

	NormalizeTrace(&tr)
	return &tr, nil
}

// ParseTraceBatch parses a JSON array of traces.
func ParseTraceBatch(r io.Reader) ([]model.Trace, error) {
	var traces []model.Trace
	if err := json.NewDecoder(r).Decode(&traces); err != nil {
		return nil, fmt.Errorf("decoding trace batch json: %w", err)
	}

	for i := range traces {
		NormalizeTrace(&traces[i])
	}
	return traces, nil
}

// ParseJSONLTraces reads newline-delimited JSON traces from a reader.
func ParseJSONLTraces(r io.Reader) ([]model.Trace, error) {
	var traces []model.Trace
	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var tr model.Trace
		if err := json.Unmarshal([]byte(line), &tr); err != nil {
			return nil, fmt.Errorf("parsing trace line %d: %w", lineNum, err)
		}

		NormalizeTrace(&tr)
		traces = append(traces, tr)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading jsonl: %w", err)
	}

	return traces, nil
}

// NormalizeTrace sets default IDs, timestamps, and calculates step counts if missing.
func NormalizeTrace(tr *model.Trace) {
	if tr.TraceID == "" {
		tr.TraceID = fmt.Sprintf("tr-%s", uuid.New().String())
	}
	if tr.Timestamp.IsZero() {
		tr.Timestamp = time.Now().UTC()
	}
	if tr.Status == "" {
		tr.Status = "success"
	}
	if tr.StepCount <= 0 {
		tr.StepCount = len(tr.Steps)
	}
	for i := range tr.Steps {
		if tr.Steps[i].StepIndex <= 0 {
			tr.Steps[i].StepIndex = i + 1
		}
	}
	if tr.TokenUsage.TotalTokens <= 0 {
		tr.TokenUsage.TotalTokens = tr.TokenUsage.PromptTokens + tr.TokenUsage.CompletionTokens
	}
}

// OTLPTracesPayload models standard OpenTelemetry JSON trace export.
type OTLPTracesPayload struct {
	ResourceSpans []struct {
		Resource struct {
			Attributes []OTLPAttribute `json:"attributes"`
		} `json:"resource"`
		ScopeSpans []struct {
			Spans []OTLPSpan `json:"spans"`
		} `json:"scopeSpans"`
	} `json:"resourceSpans"`
}

type OTLPSpan struct {
	TraceID           string          `json:"traceId"`
	SpanID            string          `json:"spanId"`
	ParentSpanID      string          `json:"parentSpanId"`
	Name              string          `json:"name"`
	Kind              int             `json:"kind"`
	StartTimeUnixNano string          `json:"startTimeUnixNano"`
	EndTimeUnixNano   string          `json:"endTimeUnixNano"`
	Attributes        []OTLPAttribute `json:"attributes"`
	Status            struct {
		Code int `json:"code"`
	} `json:"status"`
}

type OTLPAttribute struct {
	Key   string `json:"key"`
	Value struct {
		StringValue string `json:"stringValue,omitempty"`
		IntValue    string `json:"intValue,omitempty"`
		DoubleValue string `json:"doubleValue,omitempty"`
		BoolValue   bool   `json:"boolValue,omitempty"`
	} `json:"value"`
}

// ParseOTelTraces parses standard OpenTelemetry JSON trace spans into driftguard Trace representations.
func ParseOTelTraces(r io.Reader) ([]model.Trace, error) {
	var payload OTLPTracesPayload
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decoding otel json: %w", err)
	}

	tracesByTraceID := make(map[string]*model.Trace)
	toolSpansByTraceID := make(map[string][]OTLPSpan)

	for _, rs := range payload.ResourceSpans {
		defaultWorkflow := "default-agent"
		for _, attr := range rs.Resource.Attributes {
			if attr.Key == "service.name" || attr.Key == "gen_ai.workflow.id" {
				if attr.Value.StringValue != "" {
					defaultWorkflow = attr.Value.StringValue
				}
			}
		}

		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				tID := span.TraceID
				if tID == "" {
					tID = fmt.Sprintf("tr-%s", uuid.New().String()[:8])
				}

				if _, exists := tracesByTraceID[tID]; !exists {
					tracesByTraceID[tID] = &model.Trace{
						TraceID:    tID,
						WorkflowID: defaultWorkflow,
						Timestamp:  parseUnixNano(span.StartTimeUnixNano),
						Status:     "success",
					}
				}

				// Check span attributes for workflow ID overrides or tool designations
				toolName := ""
				for _, attr := range span.Attributes {
					switch attr.Key {
					case "gen_ai.workflow.id":
						tracesByTraceID[tID].WorkflowID = attr.Value.StringValue
					case "gen_ai.tool.name", "tool.name":
						toolName = attr.Value.StringValue
					case "gen_ai.usage.prompt_tokens":
						if val, err := strconv.Atoi(attr.Value.IntValue); err == nil {
							tracesByTraceID[tID].TokenUsage.PromptTokens += val
						}
					case "gen_ai.usage.completion_tokens":
						if val, err := strconv.Atoi(attr.Value.IntValue); err == nil {
							tracesByTraceID[tID].TokenUsage.CompletionTokens += val
						}
					}
				}

				if toolName != "" || strings.HasPrefix(span.Name, "tool:") || strings.Contains(span.Name, "call_") {
					toolSpansByTraceID[tID] = append(toolSpansByTraceID[tID], span)
				}
			}
		}
	}

	result := make([]model.Trace, 0, len(tracesByTraceID))
	for tID, tr := range tracesByTraceID {
		toolSpans := toolSpansByTraceID[tID]
		for i, ts := range toolSpans {
			tool := ""
			for _, attr := range ts.Attributes {
				if attr.Key == "gen_ai.tool.name" || attr.Key == "tool.name" {
					tool = attr.Value.StringValue
				}
			}
			if tool == "" {
				tool = strings.TrimPrefix(ts.Name, "tool:")
			}

			start := parseUnixNano(ts.StartTimeUnixNano)
			end := parseUnixNano(ts.EndTimeUnixNano)
			dur := end.Sub(start).Milliseconds()
			if dur < 0 {
				dur = 0
			}

			tr.Steps = append(tr.Steps, model.TraceStep{
				StepIndex:  i + 1,
				ToolName:   tool,
				DurationMs: dur,
			})
		}

		tr.StepCount = len(tr.Steps)
		tr.TokenUsage.TotalTokens = tr.TokenUsage.PromptTokens + tr.TokenUsage.CompletionTokens
		NormalizeTrace(tr)
		result = append(result, *tr)
	}

	return result, nil
}

func parseUnixNano(nanoStr string) time.Time {
	if nanoStr == "" {
		return time.Now().UTC()
	}
	nano, err := strconv.ParseInt(nanoStr, 10, 64)
	if err != nil {
		return time.Now().UTC()
	}
	return time.Unix(0, nano).UTC()
}
