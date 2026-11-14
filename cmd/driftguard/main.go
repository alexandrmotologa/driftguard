package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/api"
	"github.com/alexandrmotologa/driftguard/internal/config"
	"github.com/alexandrmotologa/driftguard/internal/ingest"
	"github.com/alexandrmotologa/driftguard/internal/model"
	"github.com/alexandrmotologa/driftguard/internal/report"
	"github.com/alexandrmotologa/driftguard/internal/sentinel"
	"github.com/alexandrmotologa/driftguard/internal/simulator"
	"github.com/alexandrmotologa/driftguard/internal/store"
	"github.com/alexandrmotologa/driftguard/web"
	"github.com/spf13/cobra"
)

var (
	cfgPath string
	dbPath  string
	port    int
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "driftguard",
		Short: "Real-time trajectory divergence sentinel and reasoning drift detector for AI agents",
		Long: `driftguard monitors autonomous AI agents for subtle reasoning degradation,
tool invocation loops, and security guardrail bypasses using 1D Wasserstein distance
and Markov transition probability matrices.`,
	}

	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "driftguard.yaml", "Path to YAML configuration file")
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "driftguard.db", "Path to SQLite database")

	rootCmd.AddCommand(newServeCmd())
	rootCmd.AddCommand(newCalibrateCmd())
	rootCmd.AddCommand(newAuditCmd())
	rootCmd.AddCommand(newIngestCmd())
	rootCmd.AddCommand(newSimDriftCmd())
	rootCmd.AddCommand(newReportCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the driftguard sentinel daemon and embedded React cockpit",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig(cfgPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if port > 0 {
				cfg.Server.Listen = fmt.Sprintf(":%d", port)
			}
			if dbPath != "" {
				cfg.Server.SQLitePath = dbPath
			}

			db, err := store.Open(cfg.Server.SQLitePath)
			if err != nil {
				return fmt.Errorf("opening database %s: %w", cfg.Server.SQLitePath, err)
			}
			defer db.Close()

			// Pre-populate initial baselines from config for all defined workflows if not present
			ctx := context.Background()
			for _, wf := range cfg.Workflows {
				existing, _ := db.GetBaseline(ctx, wf.ID)
				if existing == nil {
					base := model.Baseline{
						WorkflowID:                wf.ID,
						ExpectedStepsMean:         wf.Baseline.ExpectedStepsMean,
						ExpectedStepsStd:          wf.Baseline.ExpectedStepsStd,
						MonitoredTools:            wf.ToolsMonitored,
						MaxAllowedWassersteinDist: wf.Baseline.MaxAllowedWassersteinDist,
						MaxToolTransitionDrift:    wf.Baseline.MaxToolTransitionDrift,
						MaxLoopProbability:        wf.Baseline.MaxLoopProbability,
						UpdatedAt:                 time.Now().UTC(),
					}
					_ = db.SaveBaseline(ctx, base)
				}
			}

			server := api.NewServer(cfg, db, web.FS())

			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

			go func() {
				fmt.Printf("\n[driftguard] Sentinel daemon running at http://localhost%s\n", cfg.Server.Listen)
				fmt.Printf("[driftguard] SQLite storage: %s\n", cfg.Server.SQLitePath)
				fmt.Printf("[driftguard] Monitoring %d configured agent workflows\n\n", len(cfg.Workflows))
				if err := server.Start(); err != nil && err != http.ErrServerClosed {
					fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
					os.Exit(1)
				}
			}()

			<-stop
			fmt.Println("\n[driftguard] Shutting down sentinel gracefully...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return server.Shutdown(shutdownCtx)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "HTTP server listen port")
	return cmd
}

func newCalibrateCmd() *cobra.Command {
	var (
		workflowID string
		inputFile  string
		outputFile string
		saveDB     bool
	)

	cmd := &cobra.Command{
		Use:   "calibrate",
		Short: "Calibrate a new Golden Baseline profile from historical verified traces",
		RunE: func(cmd *cobra.Command, args []string) error {
			if workflowID == "" || inputFile == "" {
				return fmt.Errorf("both --workflow and --input are required")
			}

			f, err := os.Open(inputFile)
			if err != nil {
				return fmt.Errorf("opening input file: %w", err)
			}
			defer f.Close()

			traces, err := ingest.ParseJSONLTraces(f)
			if err != nil {
				return fmt.Errorf("parsing traces: %w", err)
			}
			if len(traces) == 0 {
				return fmt.Errorf("no traces parsed from %s", inputFile)
			}

			var wfCfg *config.WorkflowConfig
			if cfg, err := config.LoadConfig(cfgPath); err == nil {
				for i := range cfg.Workflows {
					if cfg.Workflows[i].ID == workflowID {
						wfCfg = &cfg.Workflows[i]
						break
					}
				}
			}

			baseline, err := sentinel.CalibrateFromTraces(workflowID, traces, wfCfg)
			if err != nil {
				return fmt.Errorf("calibration failed: %w", err)
			}

			if saveDB {
				db, err := store.Open(dbPath)
				if err == nil {
					defer db.Close()
					_ = db.SaveBaseline(context.Background(), *baseline)
					fmt.Printf("[driftguard] Saved calibrated baseline for '%s' to %s\n", workflowID, dbPath)
				}
			}

			if outputFile != "" {
				data, err := json.MarshalIndent(baseline, "", "  ")
				if err != nil {
					return err
				}
				if err := os.WriteFile(outputFile, data, 0644); err != nil {
					return fmt.Errorf("writing output file: %w", err)
				}
				fmt.Printf("[driftguard] Exported baseline to %s\n", outputFile)
			}

			fmt.Printf("Baseline Summary for '%s':\n", workflowID)
			fmt.Printf("  Traces Sampled:     %d\n", baseline.SampleCount)
			fmt.Printf("  Mean Steps:         %.2f ± %.2f\n", baseline.ExpectedStepsMean, baseline.ExpectedStepsStd)
			fmt.Printf("  Monitored Tools:    %v\n", baseline.MonitoredTools)
			fmt.Printf("  Max W1 Threshold:   %.2f\n", baseline.MaxAllowedWassersteinDist)
			fmt.Printf("  Max Tool Drift:     %.2f\n", baseline.MaxToolTransitionDrift)
			return nil
		},
	}

	cmd.Flags().StringVarP(&workflowID, "workflow", "w", "", "Target workflow identifier (required)")
	cmd.Flags().StringVarP(&inputFile, "input", "i", "", "Path to clean JSONL trace file (required)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Path to write Golden Baseline JSON")
	cmd.Flags().BoolVar(&saveDB, "save-db", true, "Persist calibrated baseline in SQLite database")
	return cmd
}

func newAuditCmd() *cobra.Command {
	var (
		workflowID  string
		runs        int
		failOnDrift bool
	)

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit recent production agent runs against the Golden Baseline",
		RunE: func(cmd *cobra.Command, args []string) error {
			if workflowID == "" {
				return fmt.Errorf("--workflow is required")
			}
			if runs <= 0 {
				runs = 50
			}

			cfg, err := config.LoadConfig(cfgPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			var wfCfg *config.WorkflowConfig
			for i := range cfg.Workflows {
				if cfg.Workflows[i].ID == workflowID {
					wfCfg = &cfg.Workflows[i]
					break
				}
			}

			db, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			ctx := context.Background()
			traces, err := db.GetRecentTraces(ctx, workflowID, runs)
			if err != nil {
				return fmt.Errorf("retrieving traces: %w", err)
			}
			if len(traces) == 0 {
				return fmt.Errorf("zero traces found for workflow '%s' in %s", workflowID, dbPath)
			}

			baseline, _ := db.GetBaseline(ctx, workflowID)
			eval, _ := sentinel.EvaluateWindow(workflowID, traces, baseline, wfCfg)

			fmt.Println("=========================================================")
			fmt.Printf("  driftguard SRE Sentinel Audit: %s\n", workflowID)
			fmt.Println("=========================================================")
			fmt.Printf("  Evaluation ID:       %s\n", eval.ID)
			fmt.Printf("  Runs Evaluated:      %d\n", eval.WindowSize)
			fmt.Printf("  Sentinel Status:     %s\n", eval.Status)
			fmt.Printf("  Wasserstein Dist:    %.3f (threshold: %.2f)\n",
				eval.WassersteinDistance, baselineWasserstein(baseline, wfCfg))
			fmt.Printf("  Mean Steps:          %.2f ± %.2f\n", eval.StepMean, eval.StepStd)
			fmt.Printf("  Markov Matrix Drift: %.4f\n", eval.MatrixDeviation.MeanAbsDeviation)
			fmt.Printf("  Tool Loops:          %d\n", len(eval.LoopAnomalies))
			fmt.Printf("  Novel Transitions:   %d\n", len(eval.NovelTransitions))
			fmt.Printf("  Rule Violations:     %d\n", len(eval.RuleViolations))
			fmt.Printf("  Findings:            %s\n", eval.Explanation)
			fmt.Println("=========================================================")

			if failOnDrift && eval.Status != "NORMAL" {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&workflowID, "workflow", "w", "", "Target workflow identifier (required)")
	cmd.Flags().IntVarP(&runs, "runs", "n", 50, "Number of recent runs to evaluate")
	cmd.Flags().BoolVar(&failOnDrift, "fail-on-drift", false, "Exit with non-zero code if status is WARNING or CRITICAL")
	return cmd
}

func baselineWasserstein(b *model.Baseline, cfg *config.WorkflowConfig) float64 {
	if b != nil && b.MaxAllowedWassersteinDist > 0 {
		return b.MaxAllowedWassersteinDist
	}
	if cfg != nil && cfg.Baseline.MaxAllowedWassersteinDist > 0 {
		return cfg.Baseline.MaxAllowedWassersteinDist
	}
	return 2.0
}

func newIngestCmd() *cobra.Command {
	var filePath string

	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest a JSONL trace file directly into the local SQLite database",
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return fmt.Errorf("--file is required")
			}

			f, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("opening file: %w", err)
			}
			defer f.Close()

			traces, err := ingest.ParseJSONLTraces(f)
			if err != nil {
				return fmt.Errorf("parsing jsonl traces: %w", err)
			}

			db, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			if err := db.InsertTracesBatch(context.Background(), traces); err != nil {
				return fmt.Errorf("inserting traces: %w", err)
			}

			fmt.Printf("[driftguard] Successfully ingested %d traces from %s into %s\n", len(traces), filePath, dbPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to JSONL trace file (required)")
	return cmd
}

func newSimDriftCmd() *cobra.Command {
	var (
		workflowID string
		scenario   string
		count      int
		apiURL     string
	)

	cmd := &cobra.Command{
		Use:   "sim-drift",
		Short: "Stream synthetic agent trajectories with controllable drift injection",
		RunE: func(cmd *cobra.Command, args []string) error {
			if workflowID == "" {
				workflowID = "customer-refund-agent"
			}
			if count <= 0 {
				count = 30
			}

			gen := simulator.NewGenerator(0)
			sc := simulator.Scenario(scenario)
			traces := gen.GenerateBatch(workflowID, sc, count)

			if apiURL != "" {
				// Send batch to remote running driftguard API
				url := fmt.Sprintf("%s/api/v1/traces/batch", apiURL)
				body, err := json.Marshal(traces)
				if err != nil {
					return err
				}

				resp, err := http.Post(url, "application/json", bytes.NewReader(body))
				if err != nil {
					return fmt.Errorf("posting to api %s: %w", url, err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusAccepted {
					return fmt.Errorf("api returned status %d", resp.StatusCode)
				}
				fmt.Printf("[driftguard] Dispatched %d simulated '%s' traces to %s\n", count, scenario, url)
				return nil
			}

			// Local SQLite direct insertion
			db, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			if err := db.InsertTracesBatch(context.Background(), traces); err != nil {
				return fmt.Errorf("storing traces: %w", err)
			}

			fmt.Printf("[driftguard] Successfully generated and stored %d '%s' traces for '%s'\n", count, scenario, workflowID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&workflowID, "workflow", "w", "customer-refund-agent", "Target workflow ID")
	cmd.Flags().StringVarP(&scenario, "scenario", "s", "loop", "Drift scenario: clean, loop, bypass, hallucinate")
	cmd.Flags().IntVarP(&count, "count", "n", 30, "Number of traces to generate")
	cmd.Flags().StringVar(&apiURL, "api", "", "Optional running driftguard API URL (e.g. http://localhost:8080)")
	return cmd
}

func newReportCmd() *cobra.Command {
	var (
		workflowID string
		format     string
		outputFile string
		runs       int
	)

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Export an executive postmortem HTML or JSON drift report",
		RunE: func(cmd *cobra.Command, args []string) error {
			if workflowID == "" {
				return fmt.Errorf("--workflow is required")
			}
			if runs <= 0 {
				runs = 50
			}

			db, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			ctx := context.Background()
			traces, err := db.GetRecentTraces(ctx, workflowID, runs)
			if err != nil {
				return fmt.Errorf("retrieving traces: %w", err)
			}
			if len(traces) == 0 {
				return fmt.Errorf("zero traces available for workflow '%s'", workflowID)
			}

			var wfCfg *config.WorkflowConfig
			if cfg, err := config.LoadConfig(cfgPath); err == nil {
				for i := range cfg.Workflows {
					if cfg.Workflows[i].ID == workflowID {
						wfCfg = &cfg.Workflows[i]
						break
					}
				}
			}

			baseline, _ := db.GetBaseline(ctx, workflowID)
			eval, _ := sentinel.EvaluateWindow(workflowID, traces, baseline, wfCfg)
			incidents, _ := db.GetRecentIncidents(ctx, workflowID, 20)

			if outputFile == "" {
				if format == "json" {
					outputFile = "drift_report.json"
				} else {
					outputFile = "drift_report.html"
				}
			}

			if format == "json" {
				if err := report.ExportJSONReport(outputFile, eval, incidents); err != nil {
					return err
				}
			} else {
				if err := report.ExportHTMLReport(outputFile, eval, incidents, workflowID); err != nil {
					return err
				}
			}

			fmt.Printf("[driftguard] Executive %s report generated: %s\n", format, outputFile)
			return nil
		},
	}

	cmd.Flags().StringVarP(&workflowID, "workflow", "w", "", "Target workflow identifier (required)")
	cmd.Flags().StringVarP(&format, "format", "f", "html", "Report format: html or json")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output report file path")
	cmd.Flags().IntVarP(&runs, "runs", "n", 50, "Number of recent runs to include")
	return cmd
}
