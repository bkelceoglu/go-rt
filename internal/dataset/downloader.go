package dataset

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bkelceoglu/go-rt/internal/domain"
	"github.com/bkelceoglu/go-rt/internal/store"
	"golang.org/x/sync/errgroup"
)

// EnsureBootstrapped checks if all enabled dataset_sources exist in SQLite prompt_bank; if any are missing, it pulls only the missing ones.
func EnsureBootstrapped(ctx context.Context, db *store.SQLiteStore, promptsDir string) (bool, error) {
	missing, err := findMissingSources(ctx, db)
	if err != nil || len(missing) == 0 {
		return false, err
	}
	if err := os.MkdirAll(promptsDir, 0o755); err != nil {
		return false, fmt.Errorf("create prompts directory %s: %w", promptsDir, err)
	}
	fmt.Printf("[gort] Bootstrap: Detected %d missing datasets in SQLite prompt_bank, downloading...\n", len(missing))
	pullConfiguredSources(ctx, db, promptsDir, missing)
	buildMasterSuites(ctx, db, promptsDir)
	return true, nil
}

func findMissingSources(ctx context.Context, db *store.SQLiteStore) ([]store.DatasetSourceRecord, error) {
	sources, err := db.ListDatasetSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("list dataset sources: %w", err)
	}
	existing, err := db.ListDatasets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list existing datasets: %w", err)
	}
	return filterUnpopulatedSources(sources, existing), nil
}

func filterUnpopulatedSources(sources []store.DatasetSourceRecord, existing []store.DatasetSummary) []store.DatasetSourceRecord {
	present := make(map[string]int, len(existing))
	for _, e := range existing {
		present[e.DatasetName] = e.PromptCount
	}
	missing := []store.DatasetSourceRecord{}
	for _, s := range sources {
		if present[s.DatasetName] == 0 {
			missing = append(missing, s)
		}
	}
	return missing
}

// PullAll reads dataset_sources from SQLite, fetches all remote/HF/builtin sets into SQLite prompt_bank and promptsDir.
func PullAll(ctx context.Context, db *store.SQLiteStore, promptsDir string) ([]string, error) {
	if err := os.MkdirAll(promptsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create prompts directory %s: %w", promptsDir, err)
	}
	sources, err := db.ListDatasetSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("list dataset sources: %w", err)
	}
	saved := pullConfiguredSources(ctx, db, promptsDir, sources)
	saved = append(saved, buildMasterSuites(ctx, db, promptsDir)...)
	slices.Sort(saved)
	return saved, nil
}

func pullConfiguredSources(ctx context.Context, db *store.SQLiteStore, promptsDir string, sources []store.DatasetSourceRecord) []string {
	results := make([]string, len(sources))
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(4)
	for i, src := range sources {
		g.Go(func() error {
			if p, err := syncSingleSource(gCtx, db, promptsDir, src); err == nil {
				results[i] = p
			}
			return nil
		})
	}
	_ = g.Wait()
	return compactPaths(results)
}

func syncSingleSource(ctx context.Context, db *store.SQLiteStore, promptsDir string, src store.DatasetSourceRecord) (string, error) {
	records, err := fetchSourceRecords(ctx, db, promptsDir, src)
	if err != nil || len(records) == 0 {
		return "", fmt.Errorf("no records for %s", src.DatasetName)
	}
	existing, _ := db.LoadDatasetPrompts(ctx, src.DatasetName)
	cleaned := Deduplicate(append(existing, records...))
	if err := db.SaveDatasetPrompts(ctx, src.DatasetName, cleaned); err != nil {
		return "", fmt.Errorf("save dataset prompts %s: %w", src.DatasetName, err)
	}
	csvPath := filepath.Join(promptsDir, src.DatasetName+".csv")
	return csvPath, WriteCSV(csvPath, cleaned)
}

func fetchSourceRecords(ctx context.Context, db *store.SQLiteStore, promptsDir string, src store.DatasetSourceRecord) ([]domain.DatasetRecord, error) {
	switch src.Kind {
	case "csv":
		return fetchCSVSource(ctx, promptsDir, src)
	case "hf":
		return fetchHFSource(ctx, db, promptsDir, src)
	case "json":
		return fetchJSONSource(ctx, db, promptsDir, src)
	case "jsonl":
		return fetchJSONLSource(ctx, db, promptsDir, src)
	case "bipia":
		return fetchBIPIASource(ctx, db, promptsDir, src)
	case "markdown":
		return fetchMarkdownSource(ctx, promptsDir, src)
	default:
		return db.LoadDatasetPrompts(ctx, src.DatasetName)
	}
}

func fetchCSVSource(ctx context.Context, promptsDir string, src store.DatasetSourceRecord) ([]domain.DatasetRecord, error) {
	body, err := fetchURL(ctx, src.URLOrRepo)
	if err == nil {
		if recs, err := parseRemoteCSV(body, src); err == nil && len(recs) > 0 {
			return recs, nil
		}
	}
	return ReadCSV(filepath.Join(promptsDir, src.DatasetName+".csv"))
}

func buildMasterSuites(ctx context.Context, db *store.SQLiteStore, promptsDir string) []string {
	full := collectAllBasePrompts(ctx, db)
	targets := map[string][]domain.DatasetRecord{
		"master_red_team_100":  StratifiedSample(full, 100),
		"master_red_team_1000": StratifiedSample(full, 1000),
		"master_red_team_full": full,
	}
	return persistMasterSuites(ctx, db, promptsDir, targets)
}

func collectAllBasePrompts(ctx context.Context, db *store.SQLiteStore) []domain.DatasetRecord {
	summaries, err := db.ListDatasets(ctx)
	if err != nil {
		return []domain.DatasetRecord{}
	}
	all := []domain.DatasetRecord{}
	for _, s := range summaries {
		if !strings.HasPrefix(s.DatasetName, "master_red_team_") {
			if recs, err := db.LoadDatasetPrompts(ctx, s.DatasetName); err == nil {
				all = append(all, recs...)
			}
		}
	}
	return Deduplicate(all)
}

func persistMasterSuites(ctx context.Context, db *store.SQLiteStore, promptsDir string, suites map[string][]domain.DatasetRecord) []string {
	out := make([]string, 0, len(suites))
	for name, recs := range suites {
		if err := db.ReplaceDatasetPrompts(ctx, name, recs); err != nil {
			continue
		}
		path := filepath.Join(promptsDir, name+".csv")
		if err := WriteCSV(path, recs); err == nil {
			out = append(out, path)
		}
	}
	return out
}

func fetchURL(ctx context.Context, rawURL string) (string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	for attempt := range 3 {
		if body, ok := tryFetchOnce(ctx, client, rawURL); ok {
			return body, nil
		}
		if err := waitRetryBackoff(ctx, attempt); err != nil {
			return "", fmt.Errorf("fetch %s canceled: %w", rawURL, err)
		}
	}
	return "", fmt.Errorf("fetch %s failed after retries", rawURL)
}

func waitRetryBackoff(ctx context.Context, attempt int) error {
	timer := time.NewTimer(time.Duration(attempt+1) * 450 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func tryFetchOnce(ctx context.Context, client *http.Client, rawURL string) (string, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("User-Agent", "GORT-Dataset-Downloader/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	return readHTTP200Body(resp)
}

func readHTTP200Body(resp *http.Response) (string, bool) {
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	data, err := io.ReadAll(resp.Body)
	return string(data), err == nil
}

func parseRemoteCSV(raw string, src store.DatasetSourceRecord) ([]domain.DatasetRecord, error) {
	rows, err := csv.NewReader(strings.NewReader(raw)).ReadAll()
	if err != nil || len(rows) < 2 {
		return nil, fmt.Errorf("parse remote csv: %w", err)
	}
	pIdx, cIdx := findColumnIndices(rows[0], src.PromptCol, src.CategoryCol)
	if pIdx < 0 {
		return nil, fmt.Errorf("column %s not found", src.PromptCol)
	}
	return rowsToRecords(rows[1:], pIdx, cIdx, src), nil
}

func findColumnIndices(header []string, promptCol, catCol string) (int, int) {
	pIdx, cIdx := -1, -1
	for i, col := range header {
		trimmed := strings.TrimSpace(col)
		if strings.EqualFold(trimmed, promptCol) {
			pIdx = i
		}
		if catCol != "" && strings.EqualFold(trimmed, catCol) {
			cIdx = i
		}
	}
	return pIdx, cIdx
}

func rowsToRecords(rows [][]string, pIdx, cIdx int, src store.DatasetSourceRecord) []domain.DatasetRecord {
	out := make([]domain.DatasetRecord, 0, len(rows))
	for _, row := range rows {
		if pIdx >= len(row) {
			continue
		}
		cat := safeCol(row, cIdx, src.DefaultCategory)
		out = append(out, domain.DatasetRecord{Prompt: row[pIdx], Category: cat, AttackType: src.AttackType, Source: src.SourceLabel})
	}
	return Deduplicate(out)
}

func compactPaths(paths []string) []string {
	valid := make([]string, 0, len(paths))
	for _, p := range paths {
		if p != "" {
			valid = append(valid, p)
		}
	}
	return valid
}
