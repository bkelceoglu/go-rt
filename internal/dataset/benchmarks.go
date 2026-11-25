package dataset

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bkelceoglu/go-rt/internal/domain"
	"github.com/bkelceoglu/go-rt/internal/store"
	"golang.org/x/sync/errgroup"
)

const plinyRawBaseURL = "https://raw.githubusercontent.com/elder-plinius/L1B3RT4S/refs/heads/main/"

var defaultPlinyFiles = []string{
	"-MISCELLANEOUS-.mkd", "1337.mkd", "AAA.mkd", "ALIBABA.mkd", "AMAZON.mkd",
	"ANTHROPIC.mkd", "APPLE.mkd", "BRAVE.mkd", "CHATGPT.mkd", "COHERE.mkd",
	"CURSOR.mkd", "DEEPSEEK.mkd", "FETCHAI.mkd", "GOOGLE.mkd", "GRAYSWAN.mkd",
	"GROK-MEGA.mkd", "HUME.mkd", "INCEPTION.mkd", "INFLECTION.mkd", "LIQUIDAI.mkd",
	"META.mkd", "MICROSOFT.mkd", "MIDJOURNEY.mkd", "MISTRAL.mkd", "MOONSHOT.mkd",
	"MULTION.mkd", "NOUS.mkd", "NVIDIA.mkd", "OPENAI.mkd", "PERPLEXITY.mkd",
	"REFLECTION.mkd", "REKA.mkd", "SYSTEMPROMPTS.mkd", "TOKEN80M8.mkd", "TOKENADE.mkd",
	"WINDSURF.mkd", "XAI.mkd", "ZAI.mkd", "ZYPHRA.mkd",
}

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
		TestCasePrompt string `json:"test_case_prompt"`
		UserInput      string `json:"user_input"`
	}
	if err := json.Unmarshal([]byte(body), &items); err != nil {
		return nil
	}
	out := make([]domain.DatasetRecord, 0, len(items))
	for _, it := range items {
		if prompt := formatCyberSecPrompt(it.TestCasePrompt, it.UserInput); prompt != "" {
			out = append(out, domain.DatasetRecord{Prompt: prompt, Category: src.DefaultCategory, AttackType: src.AttackType, Source: src.SourceLabel})
		}
	}
	return out
}

func formatCyberSecPrompt(testCase, userInput string) string {
	u := strings.TrimSpace(userInput)
	t := strings.TrimSpace(testCase)
	if t != "" && u != "" {
		return t + "\n\n" + u
	}
	return u
}

func fetchJSONLSource(ctx context.Context, db *store.SQLiteStore, promptsDir string, src store.DatasetSourceRecord) ([]domain.DatasetRecord, error) {
	urls := []string{
		src.URLOrRepo,
		strings.ReplaceAll(src.URLOrRepo, "attacker_cases_dh.jsonl", "attacker_cases_ds.jsonl"),
	}
	out := []domain.DatasetRecord{}
	for _, u := range urls {
		if body, err := fetchURL(ctx, u); err == nil {
			out = append(out, parseInjecAgentJSONL(body, src)...)
		}
	}
	if len(out) > 0 {
		return out, nil
	}
	return db.LoadDatasetPrompts(ctx, src.DatasetName)
}

func parseInjecAgentJSONL(body string, src store.DatasetSourceRecord) []domain.DatasetRecord {
	out := []domain.DatasetRecord{}
	for line := range strings.SplitSeq(body, "\n") {
		if rec, ok := parseSingleInjecAgentLine(line, src); ok {
			out = append(out, rec)
		}
	}
	return out
}

func parseSingleInjecAgentLine(line string, src store.DatasetSourceRecord) (domain.DatasetRecord, bool) {
	var row map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &row); err != nil {
		return domain.DatasetRecord{}, false
	}
	prompt, _ := row["Attacker Instruction"].(string)
	if strings.TrimSpace(prompt) == "" {
		return domain.DatasetRecord{}, false
	}
	attackType, _ := row["Attack Type"].(string)
	if attackType == "" {
		attackType = src.AttackType
	}
	return domain.DatasetRecord{Prompt: prompt, Category: src.DefaultCategory, AttackType: attackType, Source: src.SourceLabel}, true
}

func fetchBIPIASource(ctx context.Context, db *store.SQLiteStore, promptsDir string, src store.DatasetSourceRecord) ([]domain.DatasetRecord, error) {
	base := "https://raw.githubusercontent.com/microsoft/BIPIA/main/benchmark/"
	files := []string{"text_attack_test.json", "text_attack_train.json", "code_attack_test.json", "code_attack_train.json"}
	out := []domain.DatasetRecord{}
	for _, f := range files {
		if body, err := fetchURL(ctx, base+f); err == nil {
			out = append(out, parseBIPIAJSON(body, src)...)
		}
	}
	if len(out) > 0 {
		return out, nil
	}
	return db.LoadDatasetPrompts(ctx, src.DatasetName)
}

func parseBIPIAJSON(body string, src store.DatasetSourceRecord) []domain.DatasetRecord {
	var data map[string][]string
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil
	}
	out := []domain.DatasetRecord{}
	for cat, prompts := range data {
		for _, p := range prompts {
			if strings.TrimSpace(p) != "" {
				out = append(out, domain.DatasetRecord{Prompt: p, Category: src.DefaultCategory, AttackType: cat + " (" + src.AttackType + ")", Source: src.SourceLabel})
			}
		}
	}
	return out
}

func fetchMarkdownSource(ctx context.Context, promptsDir string, src store.DatasetSourceRecord) ([]domain.DatasetRecord, error) {
	urls := resolvePlinyMarkdownURLs(ctx, src.URLOrRepo)
	recs := fetchAllMarkdownFiles(ctx, urls, src)
	if len(recs) > 0 {
		return recs, nil
	}
	return ReadCSV(filepath.Join(promptsDir, src.DatasetName+".csv"))
}

func resolvePlinyMarkdownURLs(ctx context.Context, urlOrRepo string) []string {
	if strings.HasSuffix(urlOrRepo, ".mkd") {
		return []string{urlOrRepo}
	}
	if body, err := fetchURL(ctx, urlOrRepo); err == nil {
		if urls := parseGitHubContentsMkdURLs(body); len(urls) > 0 {
			return urls
		}
	}
	return buildDefaultPlinyURLs()
}

func parseGitHubContentsMkdURLs(body string) []string {
	var entries []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"download_url"`
	}
	if err := json.Unmarshal([]byte(body), &entries); err != nil {
		return nil
	}
	urls := []string{}
	for _, e := range entries {
		if strings.HasSuffix(strings.ToLower(e.Name), ".mkd") && e.DownloadURL != "" {
			urls = append(urls, e.DownloadURL)
		}
	}
	return urls
}

func buildDefaultPlinyURLs() []string {
	urls := make([]string, 0, len(defaultPlinyFiles))
	for _, name := range defaultPlinyFiles {
		urls = append(urls, plinyRawBaseURL+name)
	}
	return urls
}

func fetchAllMarkdownFiles(ctx context.Context, urls []string, src store.DatasetSourceRecord) []domain.DatasetRecord {
	var mu sync.Mutex
	out := []domain.DatasetRecord{}
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(5)
	for _, u := range urls {
		g.Go(func() error {
			if body, err := fetchURL(gCtx, u); err == nil {
				mu.Lock()
				out = append(out, parsePlinyMarkdown(body, src)...)
				mu.Unlock()
			}
			return nil
		})
	}
	_ = g.Wait()
	return out
}

func parsePlinyMarkdown(md string, src store.DatasetSourceRecord) []domain.DatasetRecord {
	out := []domain.DatasetRecord{}
	for sec := range strings.SplitSeq(md, "# ") {
		_, rest, hasNewline := strings.Cut(strings.TrimSpace(sec), "\n")
		if !hasNewline {
			continue
		}
		prompt := strings.TrimSpace(strings.ReplaceAll(rest, "\n", " "))
		if len(prompt) > 25 {
			out = append(out, domain.DatasetRecord{Prompt: prompt, Category: src.DefaultCategory, AttackType: src.AttackType, Source: src.SourceLabel})
		}
	}
	return out
}
