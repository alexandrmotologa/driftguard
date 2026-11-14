# CLI Reference

The `driftguard` command-line utility controls the daemon, audits historical traces, calibrates golden baselines, and generates postmortem incident reports.

## Synopsis

```bash
driftguard [command] [flags]
```

## Available Commands

### 1. `driftguard serve`
Starts the driftguard daemon, ingestion endpoints, statistical sentinel evaluator, and embedded React cockpit.

```bash
driftguard serve --config ./driftguard.yaml --port 8080 --db ./driftguard.db
```

Flags:
- `--config`, `-c`: Path to YAML configuration file (default: `./driftguard.yaml`).
- `--port`, `-p`: HTTP listen port (default: `8080`).
- `--db`: SQLite database path (default: `./driftguard.db`).
- `--no-browser`: Do not automatically log cockpit URL on start.

### 2. `driftguard calibrate`
Analyzes a historical dataset of clean, verified traces and computes a Golden Baseline profile.

```bash
driftguard calibrate --workflow customer-refund-agent --input ./traces_clean.jsonl --output ./baseline.json
```

Flags:
- `--workflow`, `-w`: Target workflow identifier (required).
- `--input`, `-i`: Path to JSONL file containing historical traces (required).
- `--output`, `-o`: Path to write calculated Golden Baseline JSON (optional, defaults to stdout or database).
- `--save-db`: Store the calculated baseline directly in the active SQLite database.

### 3. `driftguard audit`
Audits recent traces against the configured Golden Baseline and prints a statistical discrepancy table to the terminal.

```bash
driftguard audit --workflow customer-refund-agent --runs 100 --db ./driftguard.db
```

Flags:
- `--workflow`, `-w`: Workflow to audit (required).
- `--runs`, `-n`: Number of recent runs to evaluate (default: 100).
- `--db`: Path to SQLite database.
- `--fail-on-drift`: Exit with status code 1 if drift exceeds alert thresholds.

### 4. `driftguard ingest`
Ingests a JSON or JSONL trace file directly into the local SQLite database.

```bash
driftguard ingest --file ./traces.jsonl --db ./driftguard.db
```

Flags:
- `--file`, `-f`: Trace file to ingest (required).
- `--db`: Path to SQLite database.

### 5. `driftguard sim-drift`
Streams synthetic agent traffic with controllable drift injection for testing detection rules, Slack alerts, and the React cockpit.

```bash
driftguard sim-drift --workflow customer-refund-agent --scenario loop --count 60 --interval 200ms
```

Scenarios:
- `clean`: Normal healthy runs adhering to baseline.
- `loop`: Simulates model update causing tool repetition loops.
- `bypass`: Simulates safety guardrail bypass skipping required verification.
- `hallucinate`: Simulates unexpected hallucinated tool invocations.

### 6. `driftguard report`
Generates an executive postmortem HTML or JSON report summarizing detected incidents and trajectory shifts.

```bash
driftguard report --workflow customer-refund-agent --format html --output ./drift_report.html
```
