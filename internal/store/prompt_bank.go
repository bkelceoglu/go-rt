package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/bkelceoglu/go-rt/internal/domain"
	"gorm.io/gorm/clause"
)

// SaveDatasetPrompts inserts or updates a named prompt set inside the SQLite prompt_bank table via GORM.
func (s *SQLiteStore) SaveDatasetPrompts(ctx context.Context, datasetName string, records []domain.DatasetRecord) error {
	rows := buildPromptBankRows(datasetName, records)
	if len(rows) == 0 {
		return nil
	}
	err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{UpdateAll: true}).
		CreateInBatches(rows, 250).Error
	if err != nil {
		return fmt.Errorf("save dataset prompts for %s: %w", datasetName, err)
	}
	return nil
}

// ReplaceDatasetPrompts replaces all prompts for datasetName in SQLite prompt_bank with the provided records.
func (s *SQLiteStore) ReplaceDatasetPrompts(ctx context.Context, datasetName string, records []domain.DatasetRecord) error {
	if err := s.db.WithContext(ctx).Where("dataset_name = ?", datasetName).Delete(&PromptBankRecord{}).Error; err != nil {
		return fmt.Errorf("clear dataset prompts for %s: %w", datasetName, err)
	}
	return s.SaveDatasetPrompts(ctx, datasetName, records)
}

func buildPromptBankRows(datasetName string, records []domain.DatasetRecord) []PromptBankRecord {
	now := time.Now().UTC().Format(time.RFC3339)
	seen := make(map[string]struct{}, len(records))
	rows := make([]PromptBankRecord, 0, len(records))
	for _, r := range records {
		if row, ok := toUniquePromptBankRecord(datasetName, now, r, seen); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func toUniquePromptBankRecord(datasetName, now string, r domain.DatasetRecord, seen map[string]struct{}) (PromptBankRecord, bool) {
	trimmed := strings.TrimSpace(r.Prompt)
	if len(trimmed) <= 5 {
		return PromptBankRecord{}, false
	}
	h := hashPrompt(datasetName, trimmed)
	if _, exists := seen[h]; exists {
		return PromptBankRecord{}, false
	}
	seen[h] = struct{}{}
	return PromptBankRecord{
		DatasetName: datasetName,
		PromptHash:  h,
		Prompt:      trimmed,
		Category:    r.Category,
		AttackType:  r.AttackType,
		Source:      r.Source,
		CreatedAt:   now,
	}, true
}

func hashPrompt(datasetName, prompt string) string {
	sum := sha256.Sum256([]byte(datasetName + ":" + prompt))
	return hex.EncodeToString(sum[:])
}

// LoadDatasetPrompts returns all prompts belonging to datasetName from prompt_bank.
func (s *SQLiteStore) LoadDatasetPrompts(ctx context.Context, datasetName string) ([]domain.DatasetRecord, error) {
	var rows []PromptBankRecord
	err := s.db.WithContext(ctx).
		Where("dataset_name = ?", datasetName).
		Order("rowid ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load dataset prompts for %s: %w", datasetName, err)
	}
	return mapPromptBankToDatasetRecords(rows), nil
}

func mapPromptBankToDatasetRecords(rows []PromptBankRecord) []domain.DatasetRecord {
	out := make([]domain.DatasetRecord, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.DatasetRecord{
			Prompt:     r.Prompt,
			Category:   r.Category,
			AttackType: r.AttackType,
			Source:     r.Source,
		})
	}
	return out
}

// ListDatasets returns all distinct dataset names in prompt_bank with their prompt counts.
func (s *SQLiteStore) ListDatasets(ctx context.Context) ([]DatasetSummary, error) {
	out := []DatasetSummary{}
	err := s.db.WithContext(ctx).
		Model(&PromptBankRecord{}).
		Select("dataset_name, COUNT(*) as prompt_count, MIN(source) as source").
		Group("dataset_name").
		Order("dataset_name ASC").
		Scan(&out).Error
	if err != nil {
		return nil, fmt.Errorf("list datasets: %w", err)
	}
	return out, nil
}
