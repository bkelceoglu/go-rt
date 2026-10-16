package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"

	"github.com/caveman/go-rt/internal/domain"
)

// SaveDatasetPrompts inserts or updates a named prompt set inside the SQLite prompt_bank table.
func (s *SQLiteStore) SaveDatasetPrompts(ctx context.Context, datasetName string, records []domain.DatasetRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := insertPromptBatch(ctx, tx, datasetName, records); err != nil {
		return err
	}
	return tx.Commit()
}

func insertPromptBatch(ctx context.Context, tx *sql.Tx, datasetName string, records []domain.DatasetRecord) error {
	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO prompt_bank (dataset_name, prompt_hash, prompt, category, attack_type, source)
		 VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range records {
		trimmed := strings.TrimSpace(r.Prompt)
		if len(trimmed) <= 5 {
			continue
		}
		h := hashPrompt(datasetName, trimmed)
		if _, err := stmt.ExecContext(ctx, datasetName, h, trimmed, r.Category, r.AttackType, r.Source); err != nil {
			return err
		}
	}
	return nil
}

func hashPrompt(datasetName, prompt string) string {
	sum := sha256.Sum256([]byte(datasetName + ":" + prompt))
	return hex.EncodeToString(sum[:])
}

// LoadDatasetPrompts returns all prompts belonging to datasetName from prompt_bank.
func (s *SQLiteStore) LoadDatasetPrompts(ctx context.Context, datasetName string) ([]domain.DatasetRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT prompt, category, attack_type, source FROM prompt_bank WHERE dataset_name = ? ORDER BY rowid`,
		datasetName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPromptBankRows(rows)
}

func scanPromptBankRows(rows *sql.Rows) ([]domain.DatasetRecord, error) {
	var out []domain.DatasetRecord
	for rows.Next() {
		var r domain.DatasetRecord
		if err := rows.Scan(&r.Prompt, &r.Category, &r.AttackType, &r.Source); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListDatasets returns all distinct dataset names in prompt_bank with their prompt counts.
func (s *SQLiteStore) ListDatasets(ctx context.Context) ([]DatasetSummary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT dataset_name, COUNT(*), MIN(source) FROM prompt_bank GROUP BY dataset_name ORDER BY dataset_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DatasetSummary
	for rows.Next() {
		var d DatasetSummary
		if err := rows.Scan(&d.DatasetName, &d.PromptCount, &d.Source); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
