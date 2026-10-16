package dataset

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/store"
)

func fetchJSONSource(ctx context.Context, db *store.SQLiteStore, promptsDir string, src store.DatasetSourceRecord) ([]domain.DatasetRecord, error) {
	if body, err := fetchURL(ctx, src.URLOrRepo); err == nil {
		if recs := parseCyberSecEvalJSON(body, src); len(recs) > 0 {
			return recs, nil
		}
	}
	if recs, err := db.LoadDatasetPrompts(ctx, src.DatasetName); err == nil && len(recs) > 0 {
		return recs, nil
	}
	return ReadCSV(filepath.Join(promptsDir, src.DatasetName+".csv"))
}

func parseCyberSecEvalJSON(body string, src store.DatasetSourceRecord) []domain.DatasetRecord {
	var items []struct {
		UserInput string `json:"user_input"`
	}
	if err := json.Unmarshal([]byte(body), &items); err != nil {
		return nil
	}
	var out []domain.DatasetRecord
	for _, it := range items {
		if strings.TrimSpace(it.UserInput) != "" {
			out = append(out, domain.DatasetRecord{Prompt: it.UserInput, Category: src.DefaultCategory, AttackType: src.AttackType, Source: src.SourceLabel})
		}
	}
	return out
}

func fetchMarkdownSource(ctx context.Context, promptsDir string, src store.DatasetSourceRecord) ([]domain.DatasetRecord, error) {
	if body, err := fetchURL(ctx, src.URLOrRepo); err == nil {
		if recs := parsePlinyMarkdown(body, src); len(recs) > 0 {
			return recs, nil
		}
	}
	return ReadCSV(filepath.Join(promptsDir, src.DatasetName+".csv"))
}

func parsePlinyMarkdown(md string, src store.DatasetSourceRecord) []domain.DatasetRecord {
	sections := strings.Split(md, "# ")
	var out []domain.DatasetRecord
	for _, sec := range sections {
		lines := strings.Split(strings.TrimSpace(sec), "\n")
		if len(lines) < 2 {
			continue
		}
		prompt := strings.TrimSpace(strings.Join(lines[1:], " "))
		if len(prompt) > 25 {
			out = append(out, domain.DatasetRecord{Prompt: prompt, Category: src.DefaultCategory, AttackType: src.AttackType, Source: src.SourceLabel})
		}
	}
	return out
}
