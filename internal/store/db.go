package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database handle.
type DB struct {
	conn *sql.DB
}

// Open initializes or opens the SQLite database at the specified path and runs migrations.
func Open(path string) (*DB, error) {
	if path != ":memory:" {
		dir := filepath.Dir(path)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("creating db directory %s: %w", dir, err)
			}
		}
	}

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", path)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db: %w", err)
	}

	// Restrict connection pool for concurrency safety
	conn.SetMaxOpenConns(1)

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("running db migrations: %w", err)
	}

	return db, nil
}

// Close closes the underlying SQLite database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// migrate creates the necessary tables and indexes if they do not exist.
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS traces (
		trace_id TEXT PRIMARY KEY,
		workflow_id TEXT NOT NULL,
		session_id TEXT,
		model TEXT,
		timestamp DATETIME NOT NULL,
		duration_ms INTEGER NOT NULL,
		status TEXT NOT NULL,
		step_count INTEGER NOT NULL,
		prompt_tokens INTEGER DEFAULT 0,
		completion_tokens INTEGER DEFAULT 0,
		total_tokens INTEGER DEFAULT 0,
		metadata_json TEXT,
		raw_json TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_traces_wf_ts ON traces (workflow_id, timestamp DESC);

	CREATE TABLE IF NOT EXISTS trace_steps (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		trace_id TEXT NOT NULL,
		step_index INTEGER NOT NULL,
		tool_name TEXT NOT NULL,
		tool_input TEXT,
		tool_output TEXT,
		duration_ms INTEGER NOT NULL,
		tokens_used INTEGER DEFAULT 0,
		error TEXT,
		FOREIGN KEY (trace_id) REFERENCES traces(trace_id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_trace_steps_trace ON trace_steps (trace_id, step_index ASC);

	CREATE TABLE IF NOT EXISTS baselines (
		workflow_id TEXT PRIMARY KEY,
		expected_steps_mean REAL NOT NULL,
		expected_steps_std REAL NOT NULL,
		sample_count INTEGER NOT NULL,
		step_samples_json TEXT,
		markov_matrix_json TEXT NOT NULL,
		monitored_tools_json TEXT NOT NULL,
		max_allowed_wasserstein REAL NOT NULL,
		max_tool_drift REAL NOT NULL,
		max_loop_prob REAL NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS evaluations (
		id TEXT PRIMARY KEY,
		workflow_id TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		window_size INTEGER NOT NULL,
		wasserstein_dist REAL NOT NULL,
		step_mean REAL NOT NULL,
		step_std REAL NOT NULL,
		matrix_deviation REAL NOT NULL,
		status TEXT NOT NULL,
		explanation TEXT,
		result_json TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_evaluations_wf_ts ON evaluations (workflow_id, timestamp DESC);

	CREATE TABLE IF NOT EXISTS incidents (
		id TEXT PRIMARY KEY,
		workflow_id TEXT NOT NULL,
		evaluation_id TEXT,
		severity TEXT NOT NULL,
		anomaly_type TEXT NOT NULL,
		title TEXT NOT NULL,
		summary TEXT NOT NULL,
		trace_ids_json TEXT,
		metric_values_json TEXT,
		offending_transitions_json TEXT,
		acknowledged BOOLEAN DEFAULT 0,
		created_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_incidents_wf_ts ON incidents (workflow_id, created_at DESC);
	`

	_, err := db.conn.ExecContext(context.Background(), schema)
	return err
}
