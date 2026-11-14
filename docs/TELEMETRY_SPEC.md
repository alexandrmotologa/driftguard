# Telemetry Specification

driftguard supports both native OpenTelemetry JSON trace export and a direct JSON trace schema designed for minimal ingestion overhead.

## Direct JSON Trace Schema

Agents can post individual traces to `POST /api/v1/traces` or batches to `POST /api/v1/traces/batch`.

### Trace Object Example

```json
{
  "workflow_id": "customer-refund-agent",
  "trace_id": "tr-9481b0a8-b5ef-4d08-a532-65dbca18a471",
  "session_id": "sess-user-8819",
  "model": "gpt-4o-2024-08-06",
  "timestamp": "2026-11-14T20:15:30Z",
  "duration_ms": 1420,
  "status": "success",
  "step_count": 4,
  "steps": [
    {
      "step_index": 1,
      "tool_name": "lookup_order",
      "tool_input": "{\"order_id\": \"ord-9821\"}",
      "tool_output": "{\"status\": \"delivered\", \"amount\": 120.00}",
      "duration_ms": 320,
      "tokens_used": 180
    },
    {
      "step_index": 2,
      "tool_name": "check_fraud_score",
      "tool_input": "{\"user_id\": \"usr-441\", \"amount\": 120.00}",
      "tool_output": "{\"risk_score\": 0.04, \"recommendation\": \"allow\"}",
      "duration_ms": 210,
      "tokens_used": 140
    },
    {
      "step_index": 3,
      "tool_name": "execute_refund",
      "tool_input": "{\"order_id\": \"ord-9821\", \"amount\": 120.00}",
      "tool_output": "{\"refund_id\": \"ref-5501\", \"status\": \"processed\"}",
      "duration_ms": 540,
      "tokens_used": 210
    },
    {
      "step_index": 4,
      "tool_name": "send_confirmation_email",
      "tool_input": "{\"to\": \"customer@example.com\", \"refund_id\": \"ref-5501\"}",
      "tool_output": "{\"sent\": true}",
      "duration_ms": 350,
      "tokens_used": 190
    }
  ],
  "token_usage": {
    "prompt_tokens": 1250,
    "completion_tokens": 720,
    "total_tokens": 1970
  },
  "metadata": {
    "environment": "production",
    "region": "us-east-1"
  }
}
```

### Batch Ingestion

Batches can be delivered as a JSON array or newline-delimited JSON (JSONL):
```bash
curl -X POST http://localhost:8080/api/v1/traces/batch \
  -H "Content-Type: application/json" \
  -d '[{ ...trace 1... }, { ...trace 2... }]'
```

---

## OpenTelemetry Ingestion

driftguard accepts OTel trace payloads at `POST /v1/traces`.

The receiver extracts:
1. Top-level agent trace span containing attribute `gen_ai.workflow.id` or `service.name`.
2. Child spans with attribute `gen_ai.tool.name` or `traceloop.entity.name == "tool"`.
3. Duration, start time, end time, and token counts from `gen_ai.usage.prompt_tokens` and `gen_ai.usage.completion_tokens`.
