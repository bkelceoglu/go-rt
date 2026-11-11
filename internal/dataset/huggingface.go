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

const (
	hfRowsEndpoint = "https://datasets-server.huggingface.co/rows"
	hfSizeEndpoint = "https://datasets-server.huggingface.co/size?dataset="
	hfMaxPages     = 10
)

var hfRequestSem = make(chan struct{}, 1)

type hfRowsResponse struct {
	Rows []struct {
		Row map[string]any `json:"row"`
	} `json:"rows"`
}

type hfSplitInfo struct {
	Config  string `json:"config"`
	Split   string `json:"split"`
	NumRows int    `json:"num_rows"`
}

type hfSizeResponse struct {
	Size struct {
		Splits []hfSplitInfo `json:"splits"`
	} `json:"size"`
}

func fetchHFSource(ctx context.Context, db *store.SQLiteStore, promptsDir string, src store.DatasetSourceRecord) ([]domain.DatasetRecord, error) {
	select {
	case hfRequestSem <- struct{}{}:
		defer func() { <-hfRequestSem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	existing, _ := db.LoadDatasetPrompts(ctx, src.DatasetName)
	records := fetchAllHFSplits(ctx, src, len(existing))
	if len(records) > 0 {
		return records, nil
	}
	return ReadCSV(filepath.Join(promptsDir, src.DatasetName+".csv"))
}

func fetchAllHFSplits(ctx context.Context, src store.DatasetSourceRecord, existingCount int) []domain.DatasetRecord {
	splits := discoverHFSplits(ctx, src)
	all := []domain.DatasetRecord{}
	for _, sp := range splits {
		splitSrc := src
		splitSrc.HFSplit = sp.Split
		startOffset, pages := computeHFPageWindow(sp.NumRows, existingCount)
		all = append(all, fetchHFRowsWindow(ctx, splitSrc, startOffset, pages)...)
	}
	return all
}

func discoverHFSplits(ctx context.Context, src store.DatasetSourceRecord) []hfSplitInfo {
	body, err := fetchURL(ctx, hfSizeEndpoint+url.QueryEscape(src.URLOrRepo))
	if err != nil {
		return []hfSplitInfo{{Config: src.HFConfig, Split: src.HFSplit, NumRows: 1500}}
	}
	var parsed hfSizeResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return []hfSplitInfo{{Config: src.HFConfig, Split: src.HFSplit, NumRows: 1500}}
	}
	return selectTargetSplits(parsed.Size.Splits, src)
}

func selectTargetSplits(splits []hfSplitInfo, src store.DatasetSourceRecord) []hfSplitInfo {
	matched := []hfSplitInfo{}
	for _, sp := range splits {
		if sp.Config == src.HFConfig && isRelevantSplit(sp.Split, src.HFSplit) {
			matched = append(matched, sp)
		}
	}
	if len(matched) == 0 {
		return []hfSplitInfo{{Config: src.HFConfig, Split: src.HFSplit, NumRows: 1500}}
	}
	return matched
}

func isRelevantSplit(candidate, configured string) bool {
	if candidate == configured {
		return true
	}
	return configured == "train" && (candidate == "test" || candidate == "validation")
}

func computeHFPageWindow(numRows, existingCount int) (int, int) {
	totalPages := (max(numRows, 100) + 99) / 100
	if totalPages <= hfMaxPages {
		return 0, totalPages
	}
	maxStart := max(1, numRows-(hfMaxPages*100))
	startOffset := ((existingCount * 3) / 100 * 100) % maxStart
	return startOffset, hfMaxPages
}

func fetchHFRowsWindow(ctx context.Context, src store.DatasetSourceRecord, startOffset, pages int) []domain.DatasetRecord {
	all := []domain.DatasetRecord{}
	if startOffset > 0 {
		if body, err := fetchURL(ctx, buildHFRowsURL(src, 0, 100)); err == nil {
			all = append(all, decodeHFRows(body, src)...)
		}
	}
	for page := range pages {
		body, err := fetchURL(ctx, buildHFRowsURL(src, startOffset+(page*100), 100))
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
