package dataset

import (
	"cmp"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/bkelceoglu/go-rt/internal/domain"
	"github.com/bkelceoglu/go-rt/internal/store"
)

// NamedDataset represents a named prompt set retrieved from SQLite prompt_bank.
type NamedDataset struct {
	Name    string
	Records []domain.DatasetRecord
}

// ResolveDatasets resolves one or more named prompt sets (or "all" for every set in prompt_bank).
func ResolveDatasets(ctx context.Context, db *store.SQLiteStore, promptsDir string, args []string) ([]NamedDataset, error) {
	if err := os.MkdirAll(promptsDir, 0o755); err != nil {
		return nil, fmt.Errorf("ensure prompts dir %s: %w", promptsDir, err)
	}
	if len(args) == 1 && strings.EqualFold(args[0], "all") {
		return resolveAllDatasets(ctx, db, promptsDir)
	}
	out := make([]NamedDataset, 0, len(args))
	for _, arg := range args {
		ds, err := resolveSingleDataset(ctx, db, promptsDir, arg)
		if err != nil {
			return nil, err
		}
		out = append(out, ds)
	}
	return out, nil
}

func resolveAllDatasets(ctx context.Context, db *store.SQLiteStore, promptsDir string) ([]NamedDataset, error) {
	syncLocalCSVsToPromptBank(ctx, db, promptsDir)
	summaries, err := db.ListDatasets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list datasets: %w", err)
	}
	out := make([]NamedDataset, 0, len(summaries))
	for _, s := range summaries {
		if recs, err := db.LoadDatasetPrompts(ctx, s.DatasetName); err == nil && len(recs) > 0 {
			out = append(out, NamedDataset{Name: s.DatasetName, Records: recs})
		}
	}
	return out, nil
}

func syncLocalCSVsToPromptBank(ctx context.Context, db *store.SQLiteStore, promptsDir string) {
	present := loadExistingDatasetCounts(ctx, db)
	matches, err := filepath.Glob(filepath.Join(promptsDir, "*.csv"))
	if err != nil {
		return
	}
	for _, path := range matches {
		name := normalizeDatasetName(path)
		if present[name] == 0 {
			importCSVIntoPromptBank(ctx, db, name, path)
		}
	}
}

func loadExistingDatasetCounts(ctx context.Context, db *store.SQLiteStore) map[string]int {
	existing, err := db.ListDatasets(ctx)
	if err != nil {
		return map[string]int{}
	}
	present := make(map[string]int, len(existing))
	for _, s := range existing {
		present[s.DatasetName] = s.PromptCount
	}
	return present
}

func importCSVIntoPromptBank(ctx context.Context, db *store.SQLiteStore, name, path string) {
	recs, err := ReadCSV(path)
	if err != nil || len(recs) == 0 {
		return
	}
	if err := db.SaveDatasetPrompts(ctx, name, recs); err != nil {
		return
	}
}

func resolveSingleDataset(ctx context.Context, db *store.SQLiteStore, promptsDir, arg string) (NamedDataset, error) {
	if strings.HasPrefix(strings.TrimSpace(arg), "-") {
		return NamedDataset{}, fmt.Errorf("invalid dataset name %q (looks like a cli flag)", arg)
	}
	name := resolveIndexOrDatasetName(ctx, db, arg)
	if recs, err := db.LoadDatasetPrompts(ctx, name); err == nil && len(recs) > 0 {
		ensureCSVExport(promptsDir, arg, name, recs)
		return NamedDataset{Name: name, Records: recs}, nil
	}
	return loadOrSeedMissingDataset(ctx, db, promptsDir, arg, name)
}

func resolveIndexOrDatasetName(ctx context.Context, db *store.SQLiteStore, arg string) string {
	trimmed := strings.TrimSpace(arg)
	if idx, err := strconv.Atoi(trimmed); err == nil {
		if list, err := db.ListDatasets(ctx); err == nil && idx >= 1 && idx <= len(list) {
			return list[idx-1].DatasetName
		}
	}
	return normalizeDatasetName(arg)
}

func loadOrSeedMissingDataset(ctx context.Context, db *store.SQLiteStore, promptsDir, arg, name string) (NamedDataset, error) {
	csvPath := resolveCSVPath(promptsDir, arg, name)
	if recs, err := ReadCSV(csvPath); err == nil && len(recs) > 0 {
		if err := db.SaveDatasetPrompts(ctx, name, recs); err != nil {
			return NamedDataset{}, fmt.Errorf("save dataset %s: %w", name, err)
		}
		return NamedDataset{Name: name, Records: recs}, nil
	}
	if !strings.HasSuffix(strings.ToLower(arg), ".csv") {
		return NamedDataset{}, fmt.Errorf("dataset %q not found (run 'gort ls' to list available datasets)", arg)
	}
	return seedCustomCSVDataset(ctx, db, name, csvPath)
}

func seedCustomCSVDataset(ctx context.Context, db *store.SQLiteStore, name, csvPath string) (NamedDataset, error) {
	seed, err := db.LoadDatasetPrompts(ctx, "enterprise_security_suite")
	if err != nil || len(seed) == 0 {
		return NamedDataset{}, fmt.Errorf("dataset %s not found and seed empty", name)
	}
	if err := db.SaveDatasetPrompts(ctx, name, seed); err != nil {
		return NamedDataset{}, fmt.Errorf("save seeded dataset %s: %w", name, err)
	}
	if err := WriteCSV(csvPath, seed); err != nil {
		return NamedDataset{}, fmt.Errorf("write seeded csv %s: %w", csvPath, err)
	}
	return NamedDataset{Name: name, Records: seed}, nil
}

func normalizeDatasetName(arg string) string {
	base := filepath.Base(strings.TrimSpace(arg))
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func resolveCSVPath(promptsDir, arg, name string) string {
	if strings.HasSuffix(strings.ToLower(arg), ".csv") || strings.Contains(arg, "/") {
		return arg
	}
	return filepath.Join(promptsDir, name+".csv")
}

func ensureCSVExport(promptsDir, arg, name string, recs []domain.DatasetRecord) {
	csvPath := resolveCSVPath(promptsDir, arg, name)
	if _, err := os.Stat(csvPath); errors.Is(err, os.ErrNotExist) {
		if err := WriteCSV(csvPath, recs); err != nil {
			return
		}
	}
}

// StratifiedSample selects up to n records balanced evenly across categories in deterministic order.
func StratifiedSample(records []domain.DatasetRecord, n int) []domain.DatasetRecord {
	if n <= 0 || len(records) <= n {
		return records
	}
	byCat := groupRecordsByCategory(records)
	return pickRoundRobin(byCat, n)
}

func groupRecordsByCategory(records []domain.DatasetRecord) map[string][]domain.DatasetRecord {
	grouped := make(map[string][]domain.DatasetRecord)
	for _, r := range records {
		grouped[r.Category] = append(grouped[r.Category], r)
	}
	return grouped
}

func pickRoundRobin(byCat map[string][]domain.DatasetRecord, limit int) []domain.DatasetRecord {
	cats := slices.Sorted(maps.Keys(byCat))
	out := make([]domain.DatasetRecord, 0, limit)
	for len(out) < limit {
		prevLen := len(out)
		out = appendRoundPass(out, byCat, cats, limit)
		if len(out) == prevLen {
			break
		}
	}
	return out
}

func appendRoundPass(out []domain.DatasetRecord, byCat map[string][]domain.DatasetRecord, cats []string, limit int) []domain.DatasetRecord {
	for _, cat := range cats {
		list := byCat[cat]
		if len(list) > 0 && len(out) < limit {
			out = append(out, list[0])
			byCat[cat] = list[1:]
		}
	}
	return out
}

// WriteCSV writes standardized DatasetRecords (`prompt,category,attack_type,source`) to path.
func WriteCSV(path string, records []domain.DatasetRecord) error {
	cleaned := Deduplicate(records)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create csv dir for %s: %w", path, err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create csv %s: %w", path, err)
	}
	defer file.Close()
	return writeCSVRecords(file, cleaned)
}

func writeCSVRecords(file *os.File, records []domain.DatasetRecord) error {
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"prompt", "category", "attack_type", "source"}); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	for _, r := range records {
		if err := writer.Write([]string{r.Prompt, r.Category, r.AttackType, r.Source}); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	writer.Flush()
	return writer.Error()
}

// ReadCSV reads a standardized GORT CSV dataset file into DatasetRecords.
func ReadCSV(path string) ([]domain.DatasetRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open csv %s: %w", path, err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil || len(rows) < 2 {
		return nil, fmt.Errorf("read csv %s: %w", path, err)
	}
	return parseStandardRows(rows[0], rows[1:]), nil
}

type csvColumnMap struct {
	promptIdx   int
	categoryIdx int
	attackIdx   int
	sourceIdx   int
}

func parseStandardRows(header []string, rows [][]string) []domain.DatasetRecord {
	cols := resolveStandardColumns(header)
	out := make([]domain.DatasetRecord, 0, len(rows))
	for _, row := range rows {
		if cols.promptIdx < len(row) {
			out = append(out, buildRecordFromRow(row, cols))
		}
	}
	return Deduplicate(out)
}

func resolveStandardColumns(header []string) csvColumnMap {
	pIdx, cIdx := findColumnIndices(header, "prompt", "category")
	_, aIdx := findColumnIndices(header, "prompt", "attack_type")
	_, sIdx := findColumnIndices(header, "prompt", "source")
	if pIdx < 0 {
		pIdx = 0
	}
	return csvColumnMap{promptIdx: pIdx, categoryIdx: cIdx, attackIdx: aIdx, sourceIdx: sIdx}
}

func buildRecordFromRow(row []string, cols csvColumnMap) domain.DatasetRecord {
	return domain.DatasetRecord{
		Prompt:     strings.TrimSpace(row[cols.promptIdx]),
		Category:   safeCol(row, cols.categoryIdx, "gort:plugin:general"),
		AttackType: safeCol(row, cols.attackIdx, "Adversarial Probe"),
		Source:     safeCol(row, cols.sourceIdx, "Custom-Dataset"),
	}
}

func safeCol(row []string, idx int, fallback string) string {
	if idx >= 0 && idx < len(row) && strings.TrimSpace(row[idx]) != "" {
		return strings.TrimSpace(row[idx])
	}
	return fallback
}

// Deduplicate filters prompts shorter than 6 characters and removes duplicate prompt texts.
func Deduplicate(records []domain.DatasetRecord) []domain.DatasetRecord {
	seen := make(map[string]struct{}, len(records))
	out := make([]domain.DatasetRecord, 0, len(records))
	for _, r := range records {
		trimmed := strings.TrimSpace(r.Prompt)
		if len(trimmed) > 5 {
			if _, exists := seen[trimmed]; !exists {
				seen[trimmed] = struct{}{}
				r.Prompt = trimmed
				out = append(out, r)
			}
		}
	}
	return out
}

// RecordsToTestCases converts standardized DatasetRecords into executable domain.TestCases.
func RecordsToTestCases(records []domain.DatasetRecord, injectVar string) []domain.TestCase {
	injectVar = cmp.Or(injectVar, "prompt")
	cases := make([]domain.TestCase, 0, len(records))
	for i, rec := range records {
		cases = append(cases, recordToSingleTestCase(i, rec, injectVar))
	}
	return cases
}

func recordToSingleTestCase(index int, rec domain.DatasetRecord, injectVar string) domain.TestCase {
	return domain.TestCase{
		ID:     fmt.Sprintf("ds-%04d", index+1),
		Vars:   domain.Vars{injectVar: rec.Prompt},
		Assert: []domain.Assertion{{Type: "gort:grader:rubric", Metric: rec.Category}},
		Metadata: domain.Metadata{
			PluginID:  rec.Category,
			Goal:      rec.Prompt,
			Severity:  domain.SeverityHigh,
			Modifiers: map[string]string{"attack_type": rec.AttackType, "source": rec.Source},
		},
	}
}
