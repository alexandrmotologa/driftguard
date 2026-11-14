# driftguard

Real-time trajectory divergence sentinel and reasoning drift detector for production AI agents.

driftguard is a standalone Go daemon and embedded React 19 SRE cockpit that monitors autonomous language model agents. It ingests execution traces in real time, compares current trajectories against a verified Golden Baseline using 1D Wasserstein distance and Markov tool transition matrices, and alerts SRE teams on Slack and PagerDuty before behavioral regressions reach end users.

## The Problem

Autonomous agents frequently degrade after silent model checkpoint updates, prompt modifications, or RLHF changes. Unlike deterministic software failures that return HTTP 500 errors, degraded agents return HTTP 200 responses with valid JSON while exhibiting subtle failures:

- Infinite or extended tool invocation loops that drain token budgets.
- Bypassed safety or compliance guardrail tools (e.g. skipping fraud evaluation before initiating refunds).
- Hallucinated or mutated tool arguments that fail downstream business logic.
- Increased step variance causing elevated latency and operational costs.

Traditional application monitoring systems inspect server CPU and HTTP status codes, leaving silent reasoning regressions invisible until customer complaints arrive. driftguard provides statistical behavioral monitoring at the trajectory layer.

## Architecture

```
+-------------------------------------------------------------+
|                Live Production AI Agents                    |
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

## Core Capabilities

- **Lightweight Go Runtime**: Single static binary requiring under 25MB of resident memory.
- **1D Wasserstein Distance**: Quantifies step distribution shifts between baseline and rolling windows through exact numerical integration.
- **Dual-Window Trajectory Evaluation**: Evaluates micro-windows (immediate rapid regression) and macro-windows (creeping long-term drift) concurrently.
- **Markov Tool Transition Modeling**: Builds empirical transition matrices $P(T_j \mid T_i)$ to identify novel paths, cycle intensity, and probability perturbations.
- **Automated Circuit Breaker & Rollback Webhook**: Trips automatically when consecutive critical regressions occur, firing an automated fallback webhook to switch model checkpoints or engage safety proxies.
- **Visual Trajectory Diff Engine**: Produces step-by-step differential traces comparing observed agent execution against canonical baseline trajectories.
- **Interactive Directed Tool DAG**: Renders state topology graphs alongside transition heatmaps for intuitive path inspection.
- **Native Prometheus Exporter**: Exposes OpenMetrics at `GET /metrics` for Grafana, Datadog, and Prometheus Alertmanager integration.
- **Ad-Hoc Trace Testing Sandbox**: In-browser trace validator allowing SREs to test candidate traces against baselines before deployment.
- **Deterministic Guardrail Rules**: Enforces strict sequence requirements, preventing agents from skipping validation tools.
- **Real-Time Embedded Cockpit**: React 19 interface embedded directly into the Go binary via `embed.FS` with live Server-Sent Events (SSE).
- **Multi-Protocol Ingestion**: Native OpenTelemetry trace endpoint (`/v1/traces`) and low-overhead JSON webhook endpoints (`/api/v1/traces`).
- **Synthetic Drift Simulator**: Built-in CLI command to inject realistic failure modes (loops, guardrail bypasses, hallucinated tools) for automated testing.
- **Multi-Channel Alerting**: Instant dispatch to Slack incoming webhooks and PagerDuty Events API v2.

## Quickstart

### 1. Installation

Build from source (requires Go 1.22+ and Node.js 20+):

```bash
git clone https://github.com/alexandrmotologa/driftguard.git
cd driftguard

# Build the embedded React frontend and Go daemon
npm --prefix web install
npm --prefix web run build
go build -o bin/driftguard ./cmd/driftguard
```

### 2. Configuration (`driftguard.yaml`)

```yaml
version: "1"
server:
  listen: ":8080"
  sqlite_path: "./driftguard.db"

workflows:
  - id: "customer-refund-agent"
    name: "Customer Refund Automation"
    rolling_window_runs: 100
    
    baseline:
      expected_steps_mean: 3.8
      expected_steps_std: 1.1
      max_allowed_wasserstein_distance: 2.2
      max_tool_transition_drift: 0.15

    tools_monitored:
      - "lookup_order"
      - "check_fraud_score"
      - "execute_refund"
      - "send_confirmation_email"

    forbidden_transitions:
      - from: "lookup_order"
        to: "execute_refund" # Must evaluate check_fraud_score first

    circuit_breaker:
      enabled: true
      critical_consecutive_runs: 3
      fallback_webhook: "https://gateway.internal/api/v1/switch-checkpoint"
      fallback_payload: '{"workflow": "customer-refund-agent", "action": "pin_stable_version"}'

alerting:
  slack:
    webhook_url: "${SLACK_WEBHOOK_URL}"
    channel: "#ai-reliability-alerts"
  pagerduty:
    routing_key: "${PAGERDUTY_KEY}"
    severity: "error"
```

### 3. Start the Daemon

```bash
./bin/driftguard serve --config ./driftguard.yaml --port 8080
```

Open `http://localhost:8080` in your browser to access the SRE Cockpit.

### 4. Inject Traces and Test Drift Detection

In another terminal, use the built-in simulator to generate clean baseline traffic:

```bash
./bin/driftguard sim-drift --workflow customer-refund-agent --scenario clean --count 50
```

Then simulate a model checkpoint update causing an infinite tool loop:

```bash
./bin/driftguard sim-drift --workflow customer-refund-agent --scenario loop --count 25
```

The daemon detects the elevated Wasserstein distance and novel self-loop transition, logs the incident to SQLite, and updates the cockpit in real time.

## Mathematical Formulation

Detailed equations and algorithm steps are documented in [docs/MATHEMATICAL_FOUNDATIONS.md](docs/MATHEMATICAL_FOUNDATIONS.md).

### Step Count Wasserstein Distance
$$\mathcal{W}_1(P, Q) = \int_{-\infty}^{\infty} |F_P(x) - F_Q(x)| \, dx$$
Where $F_P$ and $F_Q$ are empirical cumulative distribution functions.

### Markov Transition Perturbation
$$\Delta M = \frac{1}{|T|^2} \sum_{i} \sum_{j} |M_{\text{curr}}(i, j) - M_{\text{base}}(i, j)|$$
Where $M(i, j) = P(T_j \mid T_i)$ captures sequential tool call dependencies.

## Documentation

- [System Architecture](docs/ARCHITECTURE.md)
- [Mathematical Foundations](docs/MATHEMATICAL_FOUNDATIONS.md)
- [Telemetry Specification](docs/TELEMETRY_SPEC.md)
- [CLI Reference](docs/CLI_REFERENCE.md)

## License

MIT License. See [LICENSE](LICENSE) for details.
