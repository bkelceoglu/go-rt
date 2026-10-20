package web_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/store"
	"github.com/caveman/go-rt/internal/web"
)

func TestWebServerCatalogAndConfigWizardAPIs(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := store.OpenSQLite(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	srv := web.NewServer(db, tmpDir)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	verifyIndexPage(t, ts.URL)
	verifyCatalogEndpoint(t, ts.URL)
	verifySaveAndLoadConfig(t, ts.URL)
	verifyCustomDatasetSaveAndLoad(t, ts.URL)
}

func verifyIndexPage(t *testing.T, baseURL string) {
	resp, err := http.Get(baseURL + "/")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on /, got err=%v status=%v", err, resp)
	}
	defer resp.Body.Close()
}

func verifyCatalogEndpoint(t *testing.T, baseURL string) {
	resp, err := http.Get(baseURL + "/api/catalog")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on /api/catalog, got %v", err)
	}
	defer resp.Body.Close()
	var body struct {
		Plugins    []any          `json:"plugins"`
		Strategies []any          `json:"strategies"`
		Cols       map[string]any `json:"collections"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if len(body.Plugins) < 150 || len(body.Strategies) < 10 {
		t.Fatalf("unexpected catalog counts: plugins=%d strategies=%d", len(body.Plugins), len(body.Strategies))
	}
}

func verifySaveAndLoadConfig(t *testing.T, baseURL string) {
	yamlContent := `version: "1.0"
purpose: "Wizard Test"
targets:
  - id: "llamacpp:chat:Qwen2.5-7B-Instruct"
    baseUrl: "http://localhost:8080"
    think: false
`
	payload, _ := json.Marshal(map[string]string{
		"filename": "_wizard_test.yaml",
		"rawYaml":  yamlContent,
	})
	resp, err := http.Post(baseURL+"/api/configs/save", "application/json", bytes.NewReader(payload))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("save config failed: err=%v status=%d", err, resp.StatusCode)
	}
	resp.Body.Close()

	loadResp, err := http.Get(baseURL + "/api/configs/load?file=_wizard_test.yaml")
	if err != nil || loadResp.StatusCode != http.StatusOK {
		t.Fatalf("load config failed: %v", err)
	}
	defer loadResp.Body.Close()
	var loaded struct {
		RawYAML string `json:"rawYaml"`
	}
	_ = json.NewDecoder(loadResp.Body).Decode(&loaded)
	if !strings.Contains(loaded.RawYAML, "Qwen2.5-7B-Instruct") {
		t.Fatalf("unexpected loaded yaml: %s", loaded.RawYAML)
	}
}

func verifyCustomDatasetSaveAndLoad(t *testing.T, baseURL string) {
	reqBody, _ := json.Marshal(map[string]any{
		"name": "custom_picked_set",
		"records": []domain.DatasetRecord{
			{Category: "gort:plugin:rbac", AttackType: "Custom", Prompt: "Test custom prompt 1", Source: "Wizard"},
		},
	})
	resp, err := http.Post(baseURL+"/api/datasets/custom", "application/json", bytes.NewReader(reqBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("custom dataset save failed: %v", err)
	}
	resp.Body.Close()

	getResp, err := http.Get(baseURL + "/api/datasets/custom_picked_set/prompts")
	if err != nil || getResp.StatusCode != http.StatusOK {
		t.Fatalf("custom dataset load failed: %v", err)
	}
	defer getResp.Body.Close()
}
