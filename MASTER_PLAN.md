# `driftguard` — Architecture & Engineering Master Plan
> **Real-Time Trajectory Divergence & Silent Hallucination Sentinel for Production AI**  
> *Target: High-Performance Single-Binary Go Daemon (< 25MB RAM) + Embedded React 19 SRE Cockpit*

---

## 1. Executive Summary & Market Problem

### 1.1 The Silent Reasoning Drift Crisis in Production AI
Modern engineering organizations deploy autonomous multi-step agents into production to handle critical workflows (customer support triage, financial reconciliation, automated coding). However, unlike deterministic software:
1. **Silent Foundation Model Updates**: Cloud model providers (OpenAI, Anthropic, Google) routinely deploy background checkpoint tweaks, RLHF updates, and system prompt adjustments with zero notice.
2. **Behavioral Trajectory Divergence**: A workflow that reliably completed in 4 steps suddenly degrades into a 15-step looping trajectory, calls obsolete tools, or exhibits subtle hallucinations.
3. **The Invalidation of Traditional APMs**: Tools like Datadog, New Relic, and Prometheus only inspect HTTP status codes and CPU usage. Because the LLM API returns `HTTP 200 OK`, traditional monitoring dashboards remain **100% green while the business logic is failing in production**.
4. **Delayed Discovery**: Teams only discover agentic degradation after days of silent damage when angry customers complain or API billing spikes 5x.

### 1.2 The `driftguard` Solution
`driftguard` is an autonomous SRE telemetry sentinel and anomaly detector written in Go with an embedded React 19 visual cockpit:
- **Real-Time Behavioral Telemetry Ingestion**: Ingests agent execution traces via OpenTelemetry (OTel), HTTP proxy, or lightweight SDK hooks.
- **Statistical Trajectory Baseline (Golden Distribution)**: Profiles normal agent execution across four core dimensions:
  - Step Count Distribution ($\mu, \sigma$).
  - Tool Call Transition Probability Matrix ($P(T_j \mid T_i)$).
  - Token Decision Entropy ($H = -\sum p_i \log p_i$).
  - Semantic Goal Convergence (cosine vector similarity).
- **Mathematical Divergence Engine**: Continuously computes **Wasserstein Distance (Earth Mover’s Distance)** and **KL-Divergence** on rolling execution windows.
- **Instant SRE Alerting & Safe Fallback**: Triggers Slack/PagerDuty alerts at the earliest sign of reasoning drift ($p < 0.01$), pinpointing the exact tool transition or prompt step that mutated.

---

## 2. High-Level System Architecture

```
       +--------------------------------------------------------+
       |             Live Production AI Agents                  |
       |       (Customer Ops, Coding Swarms, Data Bots)         |
       +--------------------------------------------------------+
                                   |
                                   | Streaming OTel Traces / Webhook Events
                                   v
       +--------------------------------------------------------+
       |                   driftguard Daemon                    |
       |          (Single Go Binary, Sub-25MB RAM)              |
       +--------------------------------------------------------+
                                   |
                     +-------------+-------------+
                     |                           |
                     v                           v
     +-------------------------------+   +-------------------------------+
     |    Trajectory Feature Extractor|   |    Golden Baseline Registry   |
     | - Step count & tool sequence  |   | - Historical reference profiles
     | - Tool call Markov transitions|   | - Calibrated variance thresholds
     | - Token output entropy metrics|   | - Versioned workflow baselines|
     +-------------------------------+   +-------------------------------+
                     |                           |
                     +-------------+-------------+
                                   |
                                   v
       +--------------------------------------------------------+
       |             Statistical Divergence Engine              |
       |  - Calculates Wasserstein Distance (Step Drift)        |
       |  - Markov Matrix Transition Perturbation Checks        |
       |  - Semantic Cosine Embedding Variance Tracking         |
       +--------------------------------------------------------+
                                   |
                                   v
       +--------------------------------------------------------+
       |             Anomaly Detection & Decision Gate          |
       |  - Normal Behavior: Green                              |
       |  - Drift Warning: Yellow (Log telemetry, sample up)    |
       |  - Critical Drift: Red (Page SRE, Trigger Fallback)    |
       +--------------------------------------------------------+
                                   |
                         +---------+---------+
                         |                   |
                         v                   v
     +-------------------------------+   +-------------------------------+
     |     Slack / PagerDuty Alert   |   |   React 19 SRE Drift Cockpit  |
     |  - Root-cause step diff       |   | - Markov Transition Heatmaps  |
     |  - Step count histogram shift |   | - Live Divergence Timeline    |
     |  - 1-Click Baseline Recalibrate|  | - Incident Postmortem Export  |
     +-------------------------------+   +-------------------------------+
```

---

## 3. Mathematical Divergence Formulation

### 3.1 Step Count Distribution Drift (Wasserstein Distance)
Let $P$ be the golden baseline distribution of trajectory step counts and $Q$ be the rolling empirical distribution of the last 100 production runs:
$$\mathcal{W}_1(P, Q) = \int_{-\infty}^{\infty} |F_P(x) - F_Q(x)| \, dx$$
Where $F_P$ and $F_Q$ are the cumulative distribution functions (CDFs).  
If $\mathcal{W}_1(P, Q) > \theta_{\text{step}}$, a **Step Explosion Alert** fires.

### 3.2 Markov Tool Call Transition Drift
Let $M_{\text{base}}(i, j) = P(T_j \mid T_i)$ represent the transition probability from tool $T_i$ to tool $T_j$. For current window $M_{\text{curr}}$:
$$\Delta M = \frac{1}{|T|^2} \sum_{i} \sum_{j} |M_{\text{curr}}(i, j) - M_{\text{base}}(i, j)|$$
Anomalous transitions indicate uncharacteristic tool loops or hallucinated tool parameters.

---

## 4. Declarative Sentinel Specification (`driftguard.yaml`)

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
        to: "execute_refund" # Must go through check_fraud_score first!

alerting:
  slack:
    webhook_url: "${SLACK_WEBHOOK_URL}"
    channel: "#ai-reliability-alerts"
  pagerduty:
    routing_key: "${PAGERDUTY_KEY}"
    severity: "error"
```

---

## 5. Repository & Directory Structure

```
driftguard/
├── .github/
│   └── workflows/
│       ├── test.yml
│       └── release.yml
├── cmd/
│   └── driftguard/
│       └── main.go                 # CLI entrypoint (serve, calibrate, audit)
├── internal/
│   ├── config/                     # YAML configuration parser
│   ├── ingest/                     # Trace Telemetry Ingestion
│   │   ├── otel_receiver.go        # Native OpenTelemetry gRPC / HTTP receiver
│   │   └── webhook_handler.go      # JSON event stream collector
│   ├── stats/                      # Statistical Divergence Engine
│   │   ├── wasserstein.go          # Earth Mover's Distance algorithm
│   │   ├── markov_matrix.go        # Tool transition graph & probability matrix
│   │   └── entropy.go              # Decision uncertainty calculations
│   ├── sentinel/                   # Anomaly Detection & Gating
│   │   ├── evaluator.go            # Rolling window comparator
│   │   ├── rules_engine.go         # Forbidden transition detector
│   │   └── baseline_calibrator.go  # Automatic golden baseline synthesizer
│   ├── notify/                     # Alert Dispatcher
│   │   ├── slack.go
│   │   └── pagerduty.go
│   ├── store/                      # SQLite Persistence Layer
│   │   ├── db.go
│   │   └── traces.go
│   └── api/                        # HTTP REST & SSE Server
│       ├── handler.go
│       └── routes.go
├── web/                            # Embedded React 19 SRE Cockpit
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   └── src/
│       ├── App.tsx
│       ├── components/
│       │   ├── MarkovHeatmap.tsx   # Visual tool-to-tool transition matrix
│       │   ├── StepHistogram.tsx   # Step count shift curve (Baseline vs Current)
│       │   ├── DivergenceGauge.tsx # Live Wasserstein distance gauge
│       │   └── AnomalyFeed.tsx     # Incident event log with diff details
│       └── pages/
│           ├── Overview.tsx
│           ├── WorkflowDetail.tsx
│           └── Settings.tsx
├── examples/
│   ├── sample_traces.jsonl
│   └── driftguard.yaml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 6. CLI Command Surface

```bash
# Start the driftguard daemon and web SRE cockpit
driftguard serve --config ./driftguard.yaml --port 8080

# Calibrate a new golden baseline from historical traces
driftguard calibrate --workflow customer-refund-agent --input ./traces_clean.jsonl

# Audit recent production runs against baseline from CLI
driftguard audit --workflow customer-refund-agent --runs 100

# Export an executive drift report
driftguard report --format html --output ./drift_report.html
```

---

## 7. Implementation Roadmap (5 Phases)

### Phase 1: OpenTelemetry Ingestion & Trace Schema
- Implement OTel gRPC/HTTP receiver parsing agent spans, step counts, and tool calls.
- Build SQLite database layer storing raw execution trajectories.

### Phase 2: Statistical Divergence Math (Wasserstein & Markov)
- Implement 1D Wasserstein distance algorithm for step distributions.
- Implement Markov transition matrix builder tracking tool-to-tool transitions.

### Phase 3: Anomaly Sentinel & Forbidden Transition Guard
- Implement rolling evaluation worker ticking every 30 seconds.
- Build deterministic rule validator catching prohibited sequences (e.g. bypassing safety tools).

### Phase 4: Slack Dispatcher & Embedded React 19 Cockpit
- Build rich interactive Slack alerts detailing step count shifts and offending transitions.
- Build React 19 visual dashboard with Markov heatmaps and live divergence gauges.

### Phase 5: Verification, Packaging & Documentation
- Test against synthetic drifting agent runs (simulating silent model updates).
- Verify binary footprint under 25MB RAM and execution under 5ms per trace.
- Publish documentation, animated demo GIFs, and sample datasets.
