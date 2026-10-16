package dataset

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/store"
)

const hfRowsEndpoint = "https://datasets-server.huggingface.co/rows"

type hfRowsResponse struct {
	Rows []struct {
		Row map[string]any `json:"row"`
	} `json:"rows"`
}

func fetchHFSource(ctx context.Context, promptsDir string, src store.DatasetSourceRecord) ([]domain.DatasetRecord, error) {
	records := fetchHFRowsPages(ctx, src, 3)
	if len(records) > 0 {
		return records, nil
	}
	return ReadCSV(filepath.Join(promptsDir, src.DatasetName+".csv"))
}

func fetchHFRowsPages(ctx context.Context, src store.DatasetSourceRecord, pages int) []domain.DatasetRecord {
	var all []domain.DatasetRecord
	for page := range pages {
		endpoint := buildHFRowsURL(src, page*100, 100)
		body, err := fetchURL(ctx, endpoint)
		if err != nil {
			break
		}
		all = append(all, decodeHFRows(body, src)...)
	}
	return all
}

func buildHFRowsURL(src store.DatasetSourceRecord, offset, length int) string {
	q := url.Values{}
	q.Set("dataset", src.URLOrRepo)
	q.Set("config", src.HFConfig)
	q.Set("split", src.HFSplit)
	q.Set("offset", fmt.Sprintf("%d", offset))
	q.Set("length", fmt.Sprintf("%d", length))
	return hfRowsEndpoint + "?" + q.Encode()
}

func decodeHFRows(body string, src store.DatasetSourceRecord) []domain.DatasetRecord {
	var parsed hfRowsResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return nil
	}
	var out []domain.DatasetRecord
	for _, item := range parsed.Rows {
		if !matchesFilterRule(item.Row, src.FilterRule) {
			continue
		}
		if text := extractStringCol(item.Row, src.PromptCol); len(text) > 5 {
			out = append(out, domain.DatasetRecord{Prompt: text, Category: src.DefaultCategory, AttackType: src.AttackType, Source: src.SourceLabel})
		}
	}
	return out
}

func extractStringCol(row map[string]any, primaryCol string) string {
	for _, col := range []string{primaryCol, "prompt", "text", "user_input", "Prompt"} {
		if val, ok := row[col].(string); ok && strings.TrimSpace(val) != "" {
			return strings.TrimSpace(val)
		}
	}
	return ""
}

func matchesFilterRule(row map[string]any, rule string) bool {
	switch rule {
	case "is_safe=false":
		safe, ok := row["is_safe"].(bool)
		return ok && !safe
	case "toxic_or_jb":
		tox, _ := row["toxicity"].(float64)
		jb, _ := row["jailbreaking"].(float64)
		return tox == 1 || jb == 1
	case "label=1":
		lbl, _ := row["label"].(float64)
		return lbl == 1
	case "type=jailbreak":
		t, _ := row["type"].(string)
		return strings.EqualFold(t, "jailbreak")
	default:
		return true
	}
}
