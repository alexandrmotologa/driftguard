package ingest

import (
	"strings"
	"testing"
)

func TestParseSingleTrace(t *testing.T) {
	raw := `{
		"workflow_id": "refund-bot",
		"steps": [
			{"tool_name": "lookup"},
			{"tool_name": "refund"}
		]
	}`

	tr, err := ParseSingleTrace(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseSingleTrace failed: %v", err)
	}

	if tr.WorkflowID != "refund-bot" {
		t.Errorf("expected workflow_id 'refund-bot', got %s", tr.WorkflowID)
	}
	if tr.StepCount != 2 {
		t.Errorf("expected step_count 2, got %d", tr.StepCount)
	}
	if tr.TraceID == "" {
		t.Errorf("expected generated TraceID, got empty")
	}
}

func TestParseJSONLTraces(t *testing.T) {
	raw := `
# Comment line
{"workflow_id": "agent-1", "steps": [{"tool_name": "step1"}]}
{"workflow_id": "agent-1", "steps": [{"tool_name": "step1"}, {"tool_name": "step2"}]}
`
	traces, err := ParseJSONLTraces(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseJSONLTraces failed: %v", err)
	}

	if len(traces) != 2 {
		t.Fatalf("expected 2 traces, got %d", len(traces))
	}
	if traces[0].StepCount != 1 {
		t.Errorf("expected 1 step for trace 0, got %d", traces[0].StepCount)
	}
	if traces[1].StepCount != 2 {
		t.Errorf("expected 2 steps for trace 1, got %d", traces[1].StepCount)
	}
}

func TestParseOTelTraces(t *testing.T) {
	otelRaw := `{
		"resourceSpans": [
			{
				"resource": {
					"attributes": [
						{"key": "service.name", "value": {"stringValue": "customer-support-agent"}}
					]
				},
				"scopeSpans": [
					{
						"spans": [
							{
								"traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
								"spanId": "00f067aa0ba902b7",
								"name": "tool:lookup_order",
								"startTimeUnixNano": "1731600000000000000",
								"endTimeUnixNano": "1731600000500000000",
								"attributes": [
									{"key": "gen_ai.tool.name", "value": {"stringValue": "lookup_order"}}
								]
							}
						]
					}
				]
			}
		]
	}`

	traces, err := ParseOTelTraces(strings.NewReader(otelRaw))
	if err != nil {
		t.Fatalf("ParseOTelTraces failed: %v", err)
	}

	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].WorkflowID != "customer-support-agent" {
		t.Errorf("expected workflow 'customer-support-agent', got %s", traces[0].WorkflowID)
	}
	if len(traces[0].Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(traces[0].Steps))
	}
	if traces[0].Steps[0].ToolName != "lookup_order" {
		t.Errorf("expected tool name 'lookup_order', got %s", traces[0].Steps[0].ToolName)
	}
}
