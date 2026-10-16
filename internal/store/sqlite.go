package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/caveman/go-rt/internal/domain"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

//go:embed seed_catalog.sql
var seedCatalogSQL string

// SQLiteStore manages plugins, collections, dataset_sources, prompt_bank, and OTel results in SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// PluginRecord represents a row from the SQLite plugins table.
type PluginRecord struct {
	ID             string          `json:"id"`
	Category       string          `json:"category"`
	Severity       domain.Severity `json:"severity"`
	StrategyExempt bool            `json:"strategyExempt"`
	Description    string          `json:"description"`
}

// DatasetSourceRecord represents a row from the SQLite dataset_sources table.
type DatasetSourceRecord struct {
	DatasetName     string
	Kind            string
	URLOrRepo       string
	HFConfig        string
	HFSplit         string
	PromptCol       string
	CategoryCol     string
	DefaultCategory string
	AttackType      string
	SourceLabel     string
	FilterRule      string
}

// DatasetSummary summarizes a named prompt set stored in prompt_bank.
type DatasetSummary struct {
	DatasetName string `json:"datasetName"`
	PromptCount int    `json:"promptCount"`
	Source      string `json:"source"`
}

// OpenSQLite opens or creates the SQLite database at dbPath and applies the schema and seed catalog.
func OpenSQLite(dbPath string) (*SQLiteStore, error) {
	if dbPath == "" {
		dbPath = ".gort/gort.db"
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", dbPath, err)
	}
	db.SetMaxOpenConns(1)
	s := &SQLiteStore{db: db}
	return s, s.bootstrap()
}

func (s *SQLiteStore) bootstrap() error {
	s.migrateLegacySchema()
	if _, err := s.db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("exec schema.sql: %w", err)
	}
	if _, err := s.db.Exec(seedCatalogSQL); err != nil {
		return fmt.Errorf("exec seed_catalog.sql: %w", err)
	}
	return nil
}

func (s *SQLiteStore) migrateLegacySchema() {
	var count int
	err := s.db.QueryRow(`SELECT count(*) FROM pragma_table_info('eval_results') WHERE name = 'eval_id'`).Scan(&count)
	if err == nil && count == 0 {
		_, _ = s.db.Exec(`DROP TABLE IF EXISTS eval_results; DROP TABLE IF EXISTS eval_runs; DROP TABLE IF EXISTS otel_spans;`)
	}
}

// Close closes the underlying SQLite connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// ListPlugins returns all plugins stored in the SQLite plugins table ordered by ID.
func (s *SQLiteStore) ListPlugins(ctx context.Context) ([]PluginRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, category, severity, strategy_exempt, description FROM plugins ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPluginRows(rows)
}

func scanPluginRows(rows *sql.Rows) ([]PluginRecord, error) {
	var out []PluginRecord
	for rows.Next() {
		var r PluginRecord
		var sev string
		var ex int
		if err := rows.Scan(&r.ID, &r.Category, &sev, &ex, &r.Description); err != nil {
			return nil, err
		}
		r.Severity = domain.Severity(sev)
		r.StrategyExempt = ex == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

// CollectionMap returns all preset collections mapped to their plugin IDs from SQLite.
func (s *SQLiteStore) CollectionMap(ctx context.Context) (map[string][]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT collection_id, plugin_id FROM plugin_collections ORDER BY collection_id, plugin_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]string)
	for rows.Next() {
		var cid, pid string
		if err := rows.Scan(&cid, &pid); err != nil {
			return nil, err
		}
		out[cid] = append(out[cid], pid)
	}
	return out, rows.Err()
}

// ListDatasetSources returns all enabled dataset sources from SQLite.
func (s *SQLiteStore) ListDatasetSources(ctx context.Context) ([]DatasetSourceRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT dataset_name, kind, url_or_repo, hf_config, hf_split, prompt_col, category_col, default_category, attack_type, source_label, filter_rule
		 FROM dataset_sources WHERE enabled = 1 ORDER BY dataset_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDatasetSourceRows(rows)
}

func scanDatasetSourceRows(rows *sql.Rows) ([]DatasetSourceRecord, error) {
	var out []DatasetSourceRecord
	for rows.Next() {
		var r DatasetSourceRecord
		err := rows.Scan(&r.DatasetName, &r.Kind, &r.URLOrRepo, &r.HFConfig, &r.HFSplit,
			&r.PromptCol, &r.CategoryCol, &r.DefaultCategory, &r.AttackType, &r.SourceLabel, &r.FilterRule)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
