package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

// ReportData represents the view model for the SRE postmortem and drift report.
type ReportData struct {
	WorkflowID   string
	GeneratedAt  string
	Status       string
	Evaluation   model.EvaluationResult
	Incidents    []model.Incident
	TracesCount  int
	BaselineInfo string
}

const htmlReportTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>driftguard SRE Report - {{.WorkflowID}}</title>
<style>
  :root {
    --bg: #090d16;
    --card: #111827;
    --border: #1f2937;
    --text: #f3f4f6;
    --muted: #9ca3af;
    --primary: #3b82f6;
    --danger: #ef4444;
    --warning: #f59e0b;
    --success: #10b981;
  }
  body {
    margin: 0;
    padding: 2rem;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    background: var(--bg);
    color: var(--text);
    line-height: 1.5;
  }
  .container { max-width: 1000px; margin: 0 auto; }
  .header { display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--border); padding-bottom: 1.5rem; }
  h1 { margin: 0; font-size: 1.5rem; font-weight: 600; }
  .status-badge {
    padding: 0.25rem 0.75rem;
    border-radius: 9999px;
    font-size: 0.875rem;
    font-weight: 600;
  }
  .status-NORMAL { background: rgba(16, 185, 129, 0.2); color: var(--success); }
  .status-WARNING { background: rgba(245, 158, 11, 0.2); color: var(--warning); }
  .status-CRITICAL { background: rgba(239, 68, 68, 0.2); color: var(--danger); }
  .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 1rem; margin: 1.5rem 0; }
  .card { background: var(--card); border: 1px solid var(--border); border-radius: 0.5rem; padding: 1.25rem; }
  .card-label { font-size: 0.75rem; text-transform: uppercase; color: var(--muted); letter-spacing: 0.05em; }
  .card-val { font-size: 1.75rem; font-weight: 700; margin-top: 0.25rem; }
  .section { margin-top: 2rem; }
  h2 { font-size: 1.125rem; font-weight: 600; border-bottom: 1px solid var(--border); padding-bottom: 0.5rem; }
  table { width: 100%; border-collapse: collapse; margin-top: 0.75rem; font-size: 0.875rem; }
  th, td { text-align: left; padding: 0.75rem; border-bottom: 1px solid var(--border); }
  th { color: var(--muted); font-weight: 500; }
  .code { font-family: monospace; background: rgba(255, 255, 255, 0.05); padding: 0.2rem 0.4rem; border-radius: 0.25rem; }
</style>
</head>
<body>
<div class="container">
  <div class="header">
    <div>
      <h1>driftguard Sentinel Audit Report</h1>
      <div style="color: var(--muted); font-size: 0.875rem; margin-top: 0.25rem;">
        Workflow: <span class="code">{{.WorkflowID}}</span> | Generated: {{.GeneratedAt}}
      </div>
    </div>
    <span class="status-badge status-{{.Status}}">{{.Status}}</span>
  </div>

  <div class="grid">
    <div class="card">
      <div class="card-label">Wasserstein Distance</div>
      <div class="card-val" style="color: {{if gt .Evaluation.WassersteinDistance 2.0}}var(--danger){{else}}var(--success){{end}};">
        {{printf "%.3f" .Evaluation.WassersteinDistance}}
      </div>
    </div>
    <div class="card">
      <div class="card-label">Mean Step Count</div>
      <div class="card-val">{{printf "%.1f" .Evaluation.StepMean}} ± {{printf "%.1f" .Evaluation.StepStd}}</div>
    </div>
    <div class="card">
      <div class="card-label">Markov Matrix Drift</div>
      <div class="card-val">{{printf "%.4f" .Evaluation.MatrixDeviation.MeanAbsDeviation}}</div>
    </div>
    <div class="card">
      <div class="card-label">Window Runs Evaluated</div>
      <div class="card-val">{{.Evaluation.WindowSize}}</div>
    </div>
  </div>

  <div class="section">
    <h2>Executive Finding</h2>
    <p>{{.Evaluation.Explanation}}</p>
  </div>

  {{if .Incidents}}
  <div class="section">
    <h2>Detected Anomalies & Incidents</h2>
    <table>
      <thead>
        <tr>
          <th>Time</th>
          <th>Severity</th>
          <th>Type</th>
          <th>Summary</th>
        </tr>
      </thead>
      <tbody>
        {{range .Incidents}}
        <tr>
          <td>{{.CreatedAt.Format "15:04:05 UTC"}}</td>
          <td><span class="status-badge status-{{.Severity}}">{{.Severity}}</span></td>
          <td><span class="code">{{.AnomalyType}}</span></td>
          <td>{{.Summary}}</td>
        </tr>
        {{end}}
      </tbody>
    </table>
  </div>
  {{end}}

  {{if .Evaluation.MatrixDeviation.NovelTransitions}}
  <div class="section">
    <h2>Novel Tool Transitions (Unseen in Baseline)</h2>
    <table>
      <thead>
        <tr>
          <th>From Tool</th>
          <th>To Tool</th>
          <th>Current Probability</th>
          <th>Occurrences</th>
        </tr>
      </thead>
      <tbody>
        {{range .Evaluation.MatrixDeviation.NovelTransitions}}
        <tr>
          <td><span class="code">{{.From}}</span></td>
          <td><span class="code">{{.To}}</span></td>
          <td>{{printf "%.1f%%" (mul .CurrentProb 100)}}</td>
          <td>{{.Count}}</td>
        </tr>
        {{end}}
      </tbody>
    </table>
  </div>
  {{end}}

</div>
</body>
</html>`

// ExportHTMLReport writes an executive HTML report to the specified file.
func ExportHTMLReport(path string, eval model.EvaluationResult, incidents []model.Incident, workflowID string) error {
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"mul": func(a, b float64) float64 { return a * b },
	}).Parse(htmlReportTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating report file: %w", err)
	}
	defer f.Close()

	data := ReportData{
		WorkflowID:  workflowID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Status:      eval.Status,
		Evaluation:  eval,
		Incidents:   incidents,
	}

	return tmpl.Execute(f, data)
}

// ExportJSONReport writes the evaluation and incidents data as JSON.
func ExportJSONReport(path string, eval model.EvaluationResult, incidents []model.Incident) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	payload := map[string]any{
		"evaluation": eval,
		"incidents":  incidents,
		"timestamp":  time.Now().UTC(),
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
