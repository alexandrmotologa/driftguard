# driftguard Architecture

driftguard is an anomaly detection daemon and telemetry sentinel for production AI agents. It detects silent reasoning drift, unexpected tool transition cycles, and security guardrail bypasses that occur when underlying foundation models change behavior.

## System Overview

```
+-------------------------------------------------------------+
|                Production AI Agent Workflows                |
|           (Customer Support, SRE Agents, Finance)           |
+-------------------------------------------------------------+
                               |
                               | OTel Spans / JSON Trace Events
                               v
+-------------------------------------------------------------+
|                      driftguard Daemon                      |
|                  Go Binary (< 25MB RAM Budget)              |
+-------------------------------------------------------------+
    |                          |                           |
    v                          v                           v
+---------------+      +----------------+         +----------------+
| Ingestion API |      | SQLite Storage |         | Sentinel Engine|
| OTel & Webhook|      | WAL Mode       |         | Rolling Window |
+---------------+      +----------------+         +----------------+
                               |                           |
                               +-------------+-------------+
                                             |
                                             v
                             +-------------------------------+
                             |  Statistical Divergence Math  |
                             |  - Wasserstein Step Distance  |
                             |  - Markov Transition Drift    |
                             |  - Forbidden Rule Guard       |
                             +-------------------------------+
                                             |
                               +-------------+-------------+
                               |                           |
                               v                           v
               +-------------------------------+   +---------------+
               |    Multi-Channel Alerting     |   | React Cockpit |
               |    Slack & PagerDuty Events   |   | SSE Real-Time |
               +-------------------------------+   +---------------+
```

## Core Subsystems

### 1. Ingestion Pipeline
The ingestion engine accepts agent execution traces through two interfaces:
- **OpenTelemetry Trace Ingestion** (`POST /v1/traces`): Decodes OpenTelemetry JSON traces containing LLM spans, tool call attributes, and execution steps.
- **Direct Agent Webhook** (`POST /api/v1/traces` and `/api/v1/traces/batch`): High-throughput streaming endpoint for Python, TypeScript, and Go agent runtimes.

Each trace records:
- `workflow_id`: Target agent pipeline identifier.
- `trace_id` and `session_id`: Unique execution identifiers.
- `duration_ms`: End-to-end execution latency.
- `step_count`: Total number of reasoning turns.
- `steps`: Ordered list of steps containing tool name, tool input, tool output, and step latency.
- `token_usage`: Prompt tokens, completion tokens, and total tokens.
- `status`: Completion state (`success`, `error`, `timeout`).

### 2. SQLite Embedded Storage
Traces, baselines, and incident logs are persisted in SQLite using `modernc.org/sqlite` (pure Go, zero cgo).
Key storage properties:
- WAL (Write-Ahead Logging) mode enabled for high-concurrency read/write operations.
- Indexed by `workflow_id` and `timestamp` for sub-millisecond rolling window queries.
- Automatic retention policy prunes historical raw traces older than the configured window while retaining aggregated statistics.

### 3. Statistical Divergence Engine
The divergence engine monitors reasoning drift across two primary dimensions:
- **Step Count Distribution (Wasserstein Distance)**: Measures Earth Mover's Distance between the baseline step distribution and the rolling execution window.
- **Markov Tool Transition Matrix**: Computes transition probability shifts between tools, flags unobserved transition edges, and detects recursive loops.
- **Deterministic Rule Evaluator**: Validates that required guardrail tools (such as fraud checks or approval steps) are not skipped.

### 4. Alert Dispatcher
When drift metrics exceed configured thresholds or a forbidden transition occurs:
- Formatted Slack notifications are dispatched with step histograms and offending tool traces.
- PagerDuty incidents are triggered with automatic deduplication keys.
- Real-time Server-Sent Events (SSE) stream the incident directly to connected SRE cockpits.

### 5. Embedded React 19 Cockpit
The web cockpit is embedded directly into the Go executable via `embed.FS`. SREs can inspect:
- Live transition heatmaps with baseline diff overlays.
- Step count distribution comparison curves.
- Real-time anomaly incident log with exact step diffs.
- 1-click Golden Baseline recalibration controls.
- Synthetic drift injection trigger for integration testing.

## Memory and Performance Budget
- **Memory Footprint**: Designed to remain below 25MB RAM under continuous operation.
- **Trace Ingestion Latency**: Under 3 milliseconds per trace on standard cloud instances.
- **Evaluation Loop Latency**: Under 25 milliseconds for rolling windows of 100 traces.
