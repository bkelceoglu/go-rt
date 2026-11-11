package dataset

import (
	"context"
	"encoding/json"
	"net/url"
	"path/filepath"
	"strconv"
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
	all := []domain.DatasetRecord{}
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
	q.Set("offset", strconv.Itoa(offset))
	q.Set("length", strconv.Itoa(length))
	return hfRowsEndpoint + "?" + q.Encode()
}

func decodeHFRows(body string, src store.DatasetSourceRecord) []domain.DatasetRecord {
	var parsed hfRowsResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return nil
	}
	out := make([]domain.DatasetRecord, 0, len(parsed.Rows))
	for _, item := range parsed.Rows {
		if rec, ok := rowToHFDatasetRecord(item.Row, src); ok {
			out = append(out, rec)
		}
	}
	return out
}

func rowToHFDatasetRecord(row map[string]any, src store.DatasetSourceRecord) (domain.DatasetRecord, bool) {
	if !matchesFilterRule(row, src.FilterRule) {
		return domain.DatasetRecord{}, false
	}
	text := extractStringCol(row, src.PromptCol)
	if len(text) <= 5 {
		return domain.DatasetRecord{}, false
	}
	return domain.DatasetRecord{Prompt: text, Category: src.DefaultCategory, AttackType: src.AttackType, Source: src.SourceLabel}, true
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
		isSafe, ok := row["is_safe"].(bool)
		return ok && !isSafe
	case "toxic_or_jb":
		return isFlagSet(row, "toxicity") || isFlagSet(row, "jailbreaking")
	case "label=1":
		return isFlagSet(row, "label")
	case "type=jailbreak":
		t, _ := row["type"].(string)
		return strings.EqualFold(t, "jailbreak")
	default:
		return true
	}
}

func isFlagSet(row map[string]any, key string) bool {
	val, ok := row[key].(float64)
	return ok && val == 1
}
