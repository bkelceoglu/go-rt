package dataset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/caveman/go-rt/internal/store"
)

func TestResolveNamedDatasetAndAllFromSQLite(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := store.OpenSQLite(filepath.Join(tmpDir, "gort.db"))
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	promptsDir := filepath.Join(tmpDir, "prompts")
	customCSV := filepath.Join(promptsDir, "hedehude.csv")

	t.Run("seed and resolve custom csv dataset", func(t *testing.T) {
		sets, err := ResolveDatasets(t.Context(), db, promptsDir, []string{customCSV})
		if err != nil || len(sets) != 1 || sets[0].Name != "hedehude" {
			t.Fatalf("expected single named dataset 'hedehude', got %+v (err=%v)", sets, err)
		}
		if _, err := os.Stat(customCSV); err != nil {
			t.Fatalf("expected auto-created %s on disk: %v", customCSV, err)
		}
	})

	t.Run("resolve all datasets from prompt bank", func(t *testing.T) {
		allSets, err := ResolveDatasets(t.Context(), db, promptsDir, []string{"all"})
		if err != nil || len(allSets) < 5 {
			t.Fatalf("expected at least 5 named sets for 'all', got %d (err=%v)", len(allSets), err)
		}
	})
}
